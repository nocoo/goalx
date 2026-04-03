// +build darwin

package cli

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ProbePlatformResourceState() (*ResourceState, error) {
	host, err := probeDarwinHostResourceFacts()
	if err != nil {
		// On probe failure, return a permissive default state
		return &ResourceState{
			Version:   1,
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
			Host: &ResourceHostFacts{
				MemTotalBytes:     16 * 1024 * 1024 * 1024, // assume 16GB
				MemAvailableBytes: 8 * 1024 * 1024 * 1024,  // assume 8GB available
			},
			PSI:    &ResourcePSIFacts{},
			Cgroup: &ResourceCgroupFacts{Events: &ResourceEventFacts{}},
			GoalxProcesses: &GoalXProcessFacts{
				WorkerRSSBytes: map[string]int64{},
			},
			State:   resourceStateHealthy,
			Reasons: []string{"darwin_probe_fallback"},
		}, nil
	}
	return &ResourceState{
		Version:   1,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Host:      host,
		PSI:       &ResourcePSIFacts{}, // PSI is Linux-only
		Cgroup:    &ResourceCgroupFacts{Events: &ResourceEventFacts{}}, // cgroup is Linux-only
		GoalxProcesses: &GoalXProcessFacts{
			WorkerRSSBytes: map[string]int64{},
		},
		State:   resourceStateUnknown,
		Reasons: []string{},
	}, nil
}

func probeDarwinHostResourceFacts() (*ResourceHostFacts, error) {
	facts := &ResourceHostFacts{}

	// Get total memory using sysctl
	totalOutput, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return nil, err
	}
	totalBytes, err := strconv.ParseInt(strings.TrimSpace(string(totalOutput)), 10, 64)
	if err != nil {
		return nil, err
	}
	facts.MemTotalBytes = totalBytes

	// Get memory stats using vm_stat
	vmOutput, err := exec.Command("vm_stat").Output()
	if err != nil {
		return nil, err
	}

	// Parse vm_stat output
	pageSize := int64(4096) // default page size
	pageSizeRe := regexp.MustCompile(`page size of (\d+) bytes`)
	if match := pageSizeRe.FindStringSubmatch(string(vmOutput)); len(match) > 1 {
		if ps, err := strconv.ParseInt(match[1], 10, 64); err == nil {
			pageSize = ps
		}
	}

	stats := parseVMStat(string(vmOutput))

	// Calculate available memory (free + inactive + speculative + purgeable)
	// This is an approximation of what macOS considers "available"
	freePages := stats["Pages free"]
	inactivePages := stats["Pages inactive"]
	speculativePages := stats["Pages speculative"]
	purgeablePages := stats["Pages purgeable"]

	availablePages := freePages + inactivePages + speculativePages + purgeablePages
	facts.MemAvailableBytes = availablePages * pageSize

	// macOS doesn't have traditional swap in the same way, but we can check swap usage
	swapOutput, err := exec.Command("sysctl", "-n", "vm.swapusage").Output()
	if err == nil {
		swapInfo := parseSwapUsage(string(swapOutput))
		facts.SwapTotalBytes = swapInfo.total
		facts.SwapFreeBytes = swapInfo.free
	}

	return facts, nil
}

func parseVMStat(output string) map[string]int64 {
	stats := make(map[string]int64)
	re := regexp.MustCompile(`^(.+?):\s+(\d+)`)
	for _, line := range strings.Split(output, "\n") {
		if match := re.FindStringSubmatch(line); len(match) > 2 {
			key := strings.TrimSpace(match[1])
			if val, err := strconv.ParseInt(match[2], 10, 64); err == nil {
				stats[key] = val
			}
		}
	}
	return stats
}

type swapInfo struct {
	total int64
	free  int64
}

func parseSwapUsage(output string) swapInfo {
	info := swapInfo{}
	// Example: "total = 2048.00M  used = 1024.00M  free = 1024.00M"
	re := regexp.MustCompile(`(\w+)\s*=\s*([\d.]+)([MG])`)
	for _, match := range re.FindAllStringSubmatch(output, -1) {
		if len(match) < 4 {
			continue
		}
		name := match[1]
		val, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			continue
		}
		multiplier := int64(1024 * 1024) // M
		if match[3] == "G" {
			multiplier = 1024 * 1024 * 1024
		}
		bytes := int64(val * float64(multiplier))
		switch name {
		case "total":
			info.total = bytes
		case "free":
			info.free = bytes
		}
	}
	return info
}

func collectPlatformProcessFacts(runDir string) (*GoalXProcessFacts, error) {
	// On macOS, we skip per-process RSS collection as /proc doesn't exist
	// Return empty facts - this is acceptable as resource monitoring is best-effort
	return &GoalXProcessFacts{
		WorkerRSSBytes: map[string]int64{},
	}, nil
}
