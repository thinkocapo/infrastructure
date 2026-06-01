package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	sentry "github.com/getsentry/sentry-go"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/api/types/container"
)

func CollectDocker(ctx context.Context) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("  [docker] unavailable: %v\n", err)
		return
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		fmt.Printf("  [docker] error listing containers: %v\n", err)
		return
	}

	for _, c := range containers {
		name := c.Names[0][1:] // strip leading "/"

		resp, err := cli.ContainerStats(ctx, c.ID, false)
		if err != nil {
			fmt.Printf("  [docker] error reading stats for %s: %v\n", name, err)
			continue
		}

		var stats dockerStatsJSON
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err := json.Unmarshal(body, &stats); err != nil {
			continue
		}

		cpuPct := calcCPUPercent(stats)
		memUsedMB := float64(stats.MemoryStats.Usage) / 1024 / 1024
		memLimit := float64(stats.MemoryStats.Limit)
		memPct := 0.0
		if memLimit > 0 {
			memPct = (float64(stats.MemoryStats.Usage) / memLimit) * 100
		}

		tags := map[string]string{"source": "docker", "host": "macbook", "container": name}
		sentry.Metrics.Gauge("docker.cpu.percent", cpuPct, sentry.MetricTags(tags))
		sentry.Metrics.Gauge("docker.memory.used_mb", memUsedMB, sentry.MetricTags(tags))
		sentry.Metrics.Gauge("docker.memory.percent", memPct, sentry.MetricTags(tags))

		fmt.Printf("  [docker] %s  cpu=%.1f%%  mem=%.1fMB (%.1f%%)\n", name, cpuPct, memUsedMB, memPct)
	}
}

func calcCPUPercent(stats dockerStatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	numCPUs := float64(stats.CPUStats.OnlineCPUs)
	if numCPUs == 0 {
		numCPUs = 1
	}
	if sysDelta > 0 {
		return (cpuDelta / sysDelta) * numCPUs * 100.0
	}
	return 0.0
}

type dockerStatsJSON struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  int    `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
}
