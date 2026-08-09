package collectors

import "os"

// HostTag returns the identity metrics should be tagged with — the
// machine's real hostname, so metrics from different hosts don't collide
// under one hardcoded name. Falls back to "unknown" if it can't be
// determined (e.g. a locked-down container).
func HostTag() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "unknown"
	}
	return name
}
