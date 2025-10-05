# smaps-container-exporter

Exports Linux `/proc/[pid]/smaps` memory metrics for containerized processes in a format compatible with Prometheus scraping.

This exporter is designed for Kubernetes environments and collects detailed memory mapping statistics for processes running inside containers. It uses the containerd runtime to discover container processes.

For details on the exported metrics, refer to [metrics.go](./metrics.go) and the official [procfs smaps documentation](https://docs.kernel.org/filesystems/proc.html).

## Command Line Arguments

| Argument           | Default                           | Description                                                                |
| ------------------ | --------------------------------- | -------------------------------------------------------------------------- |
| `-addr`            | `:8080`                           | Address to listen on for HTTP requests                                     |
| `-proc-path`       | `/proc`                           | Path where proc is mounted                                                 |
| `-scrape-interval` | `1s`                              | Scrape interval for metrics                                                |
| `-log-level`       | `info`                            | Log level: debug, info, warn, error, none                                  |
| `-containerd-sock` | `/run/containerd/containerd.sock` | Path to containerd socket                                                  |
| `-filter`          | `default/*/*/*`                   | Process to monitor in the format `<namespace>/<pod>/<container>/<command>` |

The `-filter` argument restricts which processes are scraped.
It uses the format `<namespace>/<pod>/<container>/<command>`, where `*` acts as a wildcard for any value.
The `command` segment should match the process name as listed in `/proc/[pid]/comm`.

Access the metrics at `http://<host>:8080/metrics`.

## Example: Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: smaps-container-exporter
spec:
  selector:
    matchLabels:
      app: smaps-container-exporter
  template:
    metadata:
      labels:
        app: smaps-container-exporter
    spec:
      containers:
        - name: exporter
          image: ghcr.io/tsaarni/smaps-container-exporter:latest
          command:
            - /smaps-container-exporter
          args:
            - --proc-path=/host/proc
            - --containerd-sock=/run/containerd/containerd.sock
            - --filter=default/*/*/*
          ports:
            - containerPort: 8080
          volumeMounts:
            - name: host-proc
              mountPath: /host/proc
              readOnly: true
            - name: containerd-sock
              mountPath: /run/containerd/containerd.sock
              readOnly: true
      volumes:
        - name: host-proc
          hostPath:
            path: /proc
            type: Directory
        - name: containerd-sock
          hostPath:
            path: /run/containerd/containerd.sock
            type: Socket
---
apiVersion: v1
kind: Service
metadata:
  name: smaps-container-exporter
spec:
  ports:
    - port: 8080
  selector:
    app: smaps-container-exporter
```
