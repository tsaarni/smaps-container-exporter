package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenAddr           = flag.String("addr", ":8080", "Address to listen on for HTTP requests")
	procPath             = flag.String("proc-path", "/proc", "Path where proc is mounted")
	interval             = flag.Duration("scrape-interval", 1*time.Second, "Scrape interval for metrics")
	logLevel             = flag.String("log-level", "info", "Log level: debug, info, warn, error, none")
	containerdSocketPath = flag.String("containerd-sock", "/run/containerd/containerd.sock", "Path to containerd socket")
	processFilter        = flag.String("filter", "default/*/*/*", "Process to monitor in the format namespace/pod/container/command. Use * as a wildcard.")
)

func pollMetrics(finder *KubernetesFinder, namespace, pod, container, command string) {
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		<-ticker.C

		slog.Debug("Polling containerd for matching processes", "namespace", namespace, "pod", pod, "container", container, "command", command)
		pids, err := finder.GetHostPIDs(namespace, pod, container, command)
		if err != nil {
			slog.Error("Failed to get host PIDs", "error", err)
			continue
		}
		if len(pids) == 0 {
			slog.Warn("No matching processes found", "namespace", namespace, "pod", pod, "container", container, "command", command)
			continue
		}

		for _, pid := range pids {
			slog.Debug("Processing smaps for", "pid", "namespace", namespace, "pod", pod, "container", container)
			err := collectAndSetMetrics(pid, namespace, pod, container)
			if err != nil {
				slog.Error("Failed to collect/set metrics", "pid", "error", err)
			}
		}
	}
}

func collectAndSetMetrics(pid int, namespace, pod, container string) error {
	comm, err := findComm(pid)
	if err != nil {
		return err
	}
	smapsPath := filepath.Join(*procPath, strconv.Itoa(pid), "smaps")
	f, err := os.Open(smapsPath)
	if err != nil {
		return err
	}
	defer f.Close()
	mappings, err := ParseSmaps(f)
	if err != nil {
		return err
	}
	for _, m := range mappings {
		setMetrics(comm, m)
	}
	return nil
}

func setMetrics(comm string, m *SmapsMapping) {
	ProcessSmapsSize.WithLabelValues(comm, m.Path).Set(float64(m.SizeBytes))
	ProcessSmapsRss.WithLabelValues(comm, m.Path).Set(float64(m.RssBytes))
	ProcessSmapsPss.WithLabelValues(comm, m.Path).Set(float64(m.PssBytes))
	ProcessSmapsPssDirty.WithLabelValues(comm, m.Path).Set(float64(m.PssDirtyBytes))
	ProcessSmapsSharedClean.WithLabelValues(comm, m.Path).Set(float64(m.SharedCleanBytes))
	ProcessSmapsSharedDirty.WithLabelValues(comm, m.Path).Set(float64(m.SharedDirtyBytes))
	ProcessSmapsPrivateClean.WithLabelValues(comm, m.Path).Set(float64(m.PrivateCleanBytes))
	ProcessSmapsPrivateDirty.WithLabelValues(comm, m.Path).Set(float64(m.PrivateDirtyBytes))
	ProcessSmapsReferenced.WithLabelValues(comm, m.Path).Set(float64(m.ReferencedBytes))
	ProcessSmapsAnonymous.WithLabelValues(comm, m.Path).Set(float64(m.AnonymousBytes))
	ProcessSmapsLazyFree.WithLabelValues(comm, m.Path).Set(float64(m.LazyFreeBytes))
	ProcessSmapsAnonHugePages.WithLabelValues(comm, m.Path).Set(float64(m.AnonHugePagesBytes))
	ProcessSmapsShmemPmdMapped.WithLabelValues(comm, m.Path).Set(float64(m.ShmemPmdMappedBytes))
	ProcessSmapsSharedHugetlb.WithLabelValues(comm, m.Path).Set(float64(m.SharedHugetlbBytes))
	ProcessSmapsPrivateHugetlb.WithLabelValues(comm, m.Path).Set(float64(m.PrivateHugetlbBytes))
	ProcessSmapsSwap.WithLabelValues(comm, m.Path).Set(float64(m.SwapBytes))
	ProcessSmapsSwapPss.WithLabelValues(comm, m.Path).Set(float64(m.SwapPssBytes))
	ProcessSmapsKernelPageSize.WithLabelValues(comm, m.Path).Set(float64(m.KernelPageSizeBytes))
	ProcessSmapsMMUPageSize.WithLabelValues(comm, m.Path).Set(float64(m.MMUPageSizeBytes))
	ProcessSmapsLocked.WithLabelValues(comm, m.Path).Set(float64(m.LockedBytes))
}

func findComm(pid int) (string, error) {
	commPath := filepath.Join(*procPath, strconv.Itoa(pid), "comm")
	data, err := os.ReadFile(commPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func parseLogLevel(level *string) slog.Level {
	switch strings.ToLower(*level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "none":
		return slog.Level(999) // Higher than any defined level.
	default:
		slog.Warn("Unknown log level, defaulting to info", "log-level", *level)
		return slog.LevelInfo
	}
}

func main() {
	flag.Parse()

	slog.SetLogLoggerLevel(parseLogLevel(logLevel))

	// Check that /proc path exists.
	if _, err := os.Stat(*procPath); os.IsNotExist(err) {
		slog.Error("The specified /proc path does not exist", "procPath", *procPath)
		os.Exit(1)
	}

	// Check that containerd socket exists.
	if _, err := os.Stat(*containerdSocketPath); os.IsNotExist(err) {
		slog.Error("The specified containerd socket does not exist", "containerdSocket", *containerdSocketPath)
		os.Exit(1)
	}

	// Check that process filter is valid.
	parts := strings.SplitN(*processFilter, "/", 4)
	if len(parts) != 4 {
		slog.Error("Invalid process filter format. Expected format: namespace/pod/container/command")
		os.Exit(1)
	}

	// Initialize Kubernetes PID finder.
	finder, err := NewKubernetesPIDFinder(*containerdSocketPath, *procPath)
	if err != nil {
		slog.Error("Failed to initialize Kubernetes PID finder", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting smaps-exporter", "listenAddr", *listenAddr, "procPath", *procPath, "scrapeInterval", *interval)

	go pollMetrics(finder, parts[0], parts[1], parts[2], parts[3])

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", http.RedirectHandler("/metrics", http.StatusFound))

	server := &http.Server{
		Addr:    *listenAddr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}
