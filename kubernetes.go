package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type KubernetesFinder struct {
	criClient        runtimeapi.RuntimeServiceClient
	containerdClient *containerd.Client
	procPath         string
}

func NewKubernetesPIDFinder(
	socketPath string,
	procPath string,
) (*KubernetesFinder, error) {
	slog.Debug("Connecting socket", "socketPath", socketPath)

	// CRI client is used to list pods and containers.
	conn, err := grpc.Dial(fmt.Sprintf("unix://%s", socketPath), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to CRI socket", "error", err)
		return nil, err
	}

	// Containerd client is used to find init PID of containers.
	containerdCClient, err := containerd.New(socketPath)
	if err != nil {
		return nil, err
	}

	return &KubernetesFinder{
		criClient:        runtimeapi.NewRuntimeServiceClient(conn),
		containerdClient: containerdCClient,
		procPath:         procPath,
	}, nil
}

// GetHostPIDs returns a list of host PIDs for processes matching the given namespace, pod, container, and command (comm).
func (k *KubernetesFinder) GetHostPIDs(namespace, pod, container, comm string) ([]int, error) {
	slog.Debug("Getting host PIDs", "namespace", namespace, "pod", pod, "container", container, "comm", comm)

	ctx := context.Background()

	// List all sandboxes.
	podResp, err := k.criClient.ListPodSandbox(ctx, &runtimeapi.ListPodSandboxRequest{})
	if err != nil {
		slog.Error("Failed to list pod sandboxes", "error", err)
		return nil, err
	}

	// Filter sandboxes by namespace and pod name.
	var podUIDs []string
	for _, sb := range podResp.Items {
		if (namespace == "*" || sb.Labels["io.kubernetes.pod.namespace"] == namespace) &&
			(pod == "*" || sb.Labels["io.kubernetes.pod.name"] == pod) {
			podUIDs = append(podUIDs, sb.Labels["io.kubernetes.pod.uid"])
		}
	}
	slog.Debug("Matching pod sandboxes", "num", len(podUIDs))
	if len(podUIDs) == 0 {
		return nil, fmt.Errorf("pod not found in sandboxes (namespace=%s, pod=%s)", namespace, pod)
	}

	// Filter containers of each matching pod by container name.
	var containerUIDs []string
	for _, podUID := range podUIDs {
		containers, err := k.getContainersForPod(ctx, podUID, container)
		if err != nil {
			continue
		}
		for _, c := range containers {
			if c.State != runtimeapi.ContainerState_CONTAINER_RUNNING {
				continue
			}
			containerUIDs = append(containerUIDs, c.Id)
		}
	}
	slog.Debug("Matching container sandboxes", "num", len(containerUIDs))
	if len(containerUIDs) == 0 {
		return nil, fmt.Errorf("no running containers found in the specified pod(s) (namespace=%s, pod=%s, container=%s)", namespace, pod, container)
	}

	// For each container sandbox, get init PID and find all PIDs in the same PID namespace.
	var allPIDs []int
	for _, containerUID := range containerUIDs {
		initPID, err := k.getInitPIDFromContainerd(containerUID)
		if err != nil {
			slog.Error("Failed to get init PID from containerd", "containerID", containerUID, "error", err)
			continue
		}

		pidNS, err := k.getPIDNamespace(initPID)
		if err != nil {
			slog.Error("Failed to get PID namespace", "initPID", initPID, "error", err)
			continue
		}

		pids := k.findPIDsInPIDNamespace(pidNS, comm)
		allPIDs = append(allPIDs, pids...)
	}
	slog.Debug("Matching PIDs in containers", "num", len(allPIDs), "pids", allPIDs)
	if len(allPIDs) == 0 {
		return nil, fmt.Errorf("no PIDs found in the specified container(s) (namespace=%s, pod=%s, container=%s, comm=%s)", namespace, pod, container, comm)
	}
	return allPIDs, nil
}

// getContainersForPod lists containers for a given pod UID and optional container name using CRI API.
func (k *KubernetesFinder) getContainersForPod(ctx context.Context, podUID, container string) ([]*runtimeapi.Container, error) {
	filter := &runtimeapi.ContainerFilter{
		LabelSelector: map[string]string{
			"io.kubernetes.pod.uid": podUID,
		},
	}
	if container != "*" {
		filter.LabelSelector["io.kubernetes.container.name"] = container
	}
	resp, err := k.criClient.ListContainers(ctx, &runtimeapi.ListContainersRequest{Filter: filter})
	if err != nil {
		return nil, err
	}
	return resp.Containers, nil
}

// getInitPIDFromContainerd retrieves the init PID of a container.
func (k *KubernetesFinder) getInitPIDFromContainerd(containerID string) (string, error) {
	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")
	container, err := k.containerdClient.LoadContainer(ctx, containerID)
	if err != nil {
		return "", err
	}
	task, err := container.Task(ctx, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", task.Pid()), nil
}

// getPIDNamespace returns the PID namespace inode for a given PID by reading /proc/<pid>/ns/pid
func (k *KubernetesFinder) getPIDNamespace(pid string) (string, error) {
	nsPath := filepath.Join(k.procPath, pid, "ns", "pid")
	link, err := os.Readlink(nsPath)
	if err != nil {
		return "", err
	}
	// The link is of the form 'pid:[4026531836]'
	var nsid int
	n, err := fmt.Sscanf(link, "pid:[%d]", &nsid)
	if n == 1 && err == nil {
		return fmt.Sprintf("%d", nsid), nil
	}
	return "", fmt.Errorf("unexpected ns link format: %s", link)
}

// findPIDsInPIDNamespace finds all PIDs in /proc that are in the given namespace and match comm (process name)
func (k *KubernetesFinder) findPIDsInPIDNamespace(ns, comm string) []int {
	var pids []int
	entries, err := os.ReadDir(k.procPath)
	if err != nil {
		return pids
	}
	for _, entry := range entries {
		pid := entry.Name()
		pidInt, err := strconv.Atoi(pid)
		if err != nil {
			continue // Not a numeric directory, skip.
		}
		if !entry.IsDir() {
			continue
		}
		if !k.pidNamespaceMatches(pid, ns) {
			continue
		}
		if comm != "*" && !k.commMatches(pid, comm) {
			continue
		}
		pids = append(pids, pidInt)
	}
	return pids
}

// pidNamespaceMatches checks if the PID namespace of the given pid matches the specified namespace inode.
func (k *KubernetesFinder) pidNamespaceMatches(pid, ns string) bool {
	nsPath := filepath.Join(k.procPath, pid, "ns", "pid")
	link, err := os.Readlink(nsPath)
	if err != nil {
		return false
	}
	var pidnsInt int
	n, err := fmt.Sscanf(link, "pid:[%d]", &pidnsInt)
	if n == 1 && err == nil {
		return fmt.Sprintf("%d", pidnsInt) == ns
	}
	return false
}

// commMatches checks if the command name (comm) of the given pid matches the specified comm.
func (k *KubernetesFinder) commMatches(pid, comm string) bool {
	commPath := filepath.Join(k.procPath, pid, "comm")
	data, err := os.ReadFile(commPath)
	if err != nil {
		return false
	}
	procComm := strings.TrimSpace(string(data))
	return procComm == comm
}
