package collectors

import (
	"context"
	"strings"
)

// Collector is a named metrics source.
//
// To add a third source (e.g. Postgres, Redis, Kubernetes):
//  1. write a CollectX(ctx context.Context) function in its own file
//  2. add one line to Registry below
// Everything else — flag parsing, the run loop — picks it up automatically.
type Collector struct {
	Name    string
	Collect func(ctx context.Context)
}

// Registry is the list of all available collectors.
var Registry = []Collector{
	{Name: "host", Collect: CollectHost},
	{Name: "docker", Collect: CollectDocker},
}

// Names returns the names of all registered collectors, for help/error text.
func Names() string {
	out := make([]string, len(Registry))
	for i, c := range Registry {
		out[i] = c.Name
	}
	return strings.Join(out, ", ")
}

// Select returns the collectors whose names appear in `names`, preserving
// registry order. An empty `names` selects all collectors. Any names that
// don't match a registered collector are returned as `unknown`.
func Select(names []string) (chosen []Collector, unknown []string) {
	if len(names) == 0 {
		return Registry, nil
	}

	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}

	for _, c := range Registry {
		if want[c.Name] {
			chosen = append(chosen, c)
			delete(want, c.Name) // mark as matched
		}
	}
	for n := range want {
		unknown = append(unknown, n)
	}
	return chosen, unknown
}

// ParseSelection splits a comma-separated list like "host,docker" into names,
// trimming whitespace and dropping empties.
func ParseSelection(s string) []string {
	var names []string
	for _, n := range strings.Split(s, ",") {
		if t := strings.TrimSpace(n); t != "" {
			names = append(names, t)
		}
	}
	return names
}
