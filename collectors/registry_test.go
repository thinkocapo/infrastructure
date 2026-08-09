package collectors

import (
	"context"
	"reflect"
	"testing"
)

func TestNames(t *testing.T) {
	got := Names()
	want := "host, docker"
	if got != want {
		t.Errorf("Names() = %q, want %q", got, want)
	}
}

func TestParseSelection(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"host", []string{"host"}},
		{"host,docker", []string{"host", "docker"}},
		{" host , docker ", []string{"host", "docker"}},
		{"host,,docker", []string{"host", "docker"}},
		{",,", nil},
	}
	for _, c := range cases {
		got := ParseSelection(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseSelection(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSelect(t *testing.T) {
	t.Run("empty selects all", func(t *testing.T) {
		chosen, unknown := Select(nil)
		if len(unknown) != 0 {
			t.Errorf("unknown = %v, want none", unknown)
		}
		if !reflect.DeepEqual(chosen, Registry) {
			t.Errorf("chosen = %v, want full Registry", chosen)
		}
	})

	t.Run("subset preserves registry order", func(t *testing.T) {
		chosen, unknown := Select([]string{"docker", "host"})
		if len(unknown) != 0 {
			t.Errorf("unknown = %v, want none", unknown)
		}
		if len(chosen) != 2 || chosen[0].Name != "host" || chosen[1].Name != "docker" {
			t.Errorf("chosen = %v, want [host, docker] in registry order", chosen)
		}
	})

	t.Run("unknown names reported, known ones still selected", func(t *testing.T) {
		chosen, unknown := Select([]string{"host", "kubernetes"})
		if len(chosen) != 1 || chosen[0].Name != "host" {
			t.Errorf("chosen = %v, want [host]", chosen)
		}
		if len(unknown) != 1 || unknown[0] != "kubernetes" {
			t.Errorf("unknown = %v, want [kubernetes]", unknown)
		}
	})
}

// TestCollectFuncSignature guards against accidentally reverting Collect to
// the old fire-and-forget `func(ctx)` signature — self-monitoring in
// main.go depends on every collector returning an error.
func TestCollectFuncSignature(t *testing.T) {
	for _, c := range Registry {
		var _ func(context.Context) error = c.Collect
	}
}
