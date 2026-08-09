package collectors

import (
	"os"
	"testing"
)

func TestHostTagMatchesOSHostname(t *testing.T) {
	want, err := os.Hostname()
	if err != nil {
		t.Skipf("os.Hostname() unavailable in this environment: %v", err)
	}
	if got := HostTag(); got != want {
		t.Errorf("HostTag() = %q, want %q (os.Hostname())", got, want)
	}
}

func TestHostTagNeverEmpty(t *testing.T) {
	if HostTag() == "" {
		t.Error("HostTag() returned empty string — should fall back to \"unknown\"")
	}
}
