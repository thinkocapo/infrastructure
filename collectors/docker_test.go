package collectors

import (
	"os"
	"testing"
)

func TestContainerAllowlist(t *testing.T) {
	t.Run("unset means no filtering", func(t *testing.T) {
		os.Unsetenv("CONTAINERS")
		if got := containerAllowlist(); got != nil {
			t.Errorf("containerAllowlist() = %v, want nil", got)
		}
	})

	t.Run("comma-separated names become the allow set", func(t *testing.T) {
		os.Setenv("CONTAINERS", "postgres, redis")
		defer os.Unsetenv("CONTAINERS")

		got := containerAllowlist()
		want := map[string]bool{"postgres": true, "redis": true}
		if len(got) != len(want) || !got["postgres"] || !got["redis"] {
			t.Errorf("containerAllowlist() = %v, want %v", got, want)
		}
	})
}

func TestCalcCPUPercent(t *testing.T) {
	cases := []struct {
		name  string
		stats dockerStatsJSON
		want  float64
	}{
		{
			name: "typical usage",
			stats: dockerStatsJSON{
				CPUStats: struct {
					CPUUsage struct {
						TotalUsage uint64 `json:"total_usage"`
					} `json:"cpu_usage"`
					SystemUsage uint64 `json:"system_cpu_usage"`
					OnlineCPUs  int    `json:"online_cpus"`
				}{
					CPUUsage:    struct{ TotalUsage uint64 `json:"total_usage"` }{TotalUsage: 2_000_000_000},
					SystemUsage: 10_000_000_000,
					OnlineCPUs:  4,
				},
				PreCPUStats: struct {
					CPUUsage struct {
						TotalUsage uint64 `json:"total_usage"`
					} `json:"cpu_usage"`
					SystemUsage uint64 `json:"system_cpu_usage"`
				}{
					CPUUsage:    struct{ TotalUsage uint64 `json:"total_usage"` }{TotalUsage: 1_000_000_000},
					SystemUsage: 8_000_000_000,
				},
			},
			// cpuDelta=1e9, sysDelta=2e9, numCPUs=4 -> (1e9/2e9)*4*100 = 200
			want: 200.0,
		},
		{
			name: "zero system delta returns zero, not divide-by-zero",
			stats: dockerStatsJSON{
				CPUStats: struct {
					CPUUsage struct {
						TotalUsage uint64 `json:"total_usage"`
					} `json:"cpu_usage"`
					SystemUsage uint64 `json:"system_cpu_usage"`
					OnlineCPUs  int    `json:"online_cpus"`
				}{
					CPUUsage:    struct{ TotalUsage uint64 `json:"total_usage"` }{TotalUsage: 100},
					SystemUsage: 500,
					OnlineCPUs:  2,
				},
				PreCPUStats: struct {
					CPUUsage struct {
						TotalUsage uint64 `json:"total_usage"`
					} `json:"cpu_usage"`
					SystemUsage uint64 `json:"system_cpu_usage"`
				}{
					CPUUsage:    struct{ TotalUsage uint64 `json:"total_usage"` }{TotalUsage: 100},
					SystemUsage: 500,
				},
			},
			want: 0.0,
		},
		{
			name: "zero online CPUs defaults to 1 instead of dividing by zero",
			stats: dockerStatsJSON{
				CPUStats: struct {
					CPUUsage struct {
						TotalUsage uint64 `json:"total_usage"`
					} `json:"cpu_usage"`
					SystemUsage uint64 `json:"system_cpu_usage"`
					OnlineCPUs  int    `json:"online_cpus"`
				}{
					CPUUsage:    struct{ TotalUsage uint64 `json:"total_usage"` }{TotalUsage: 300_000_000},
					SystemUsage: 4_000_000_000,
					OnlineCPUs:  0,
				},
				PreCPUStats: struct {
					CPUUsage struct {
						TotalUsage uint64 `json:"total_usage"`
					} `json:"cpu_usage"`
					SystemUsage uint64 `json:"system_cpu_usage"`
				}{
					CPUUsage:    struct{ TotalUsage uint64 `json:"total_usage"` }{TotalUsage: 100_000_000},
					SystemUsage: 3_000_000_000,
				},
			},
			// cpuDelta=2e8, sysDelta=1e9, numCPUs defaults to 1 -> (2e8/1e9)*1*100 = 20
			want: 20.0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := calcCPUPercent(c.stats)
			if got != c.want {
				t.Errorf("calcCPUPercent() = %v, want %v", got, c.want)
			}
		})
	}
}
