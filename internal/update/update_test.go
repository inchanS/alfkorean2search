package update

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		remote, current string
		want            bool
	}{
		{"v1.0.0", "0.3.1", true},
		{"v1.0.0", "1.0.0", false},
		{"v0.3.1", "1.0.0", false},
		{"v1.2.0", "v1.1.9", true},
		{"1.0.1", "1.0.0", true},
		{"v1.0.0", "", true},            // unknown current -> treat remote as newer
		{"", "1.0.0", false},            // no remote tag -> not newer
		{"v1.0.0-beta", "1.0.0", false}, // pre-release suffix ignored, equal
	}
	for _, c := range cases {
		if got := isNewer(c.remote, c.current); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.remote, c.current, got, c.want)
		}
	}
}

func TestShouldNotify(t *testing.T) {
	cases := []struct {
		name    string
		inf     info
		current string
		want    bool
	}{
		{"newer cached release", info{Version: "v1.1.0"}, "1.0.0", true},
		{"same version -> no notice", info{Version: "v1.0.0"}, "1.0.0", false},
		{"empty cached version", info{Version: ""}, "1.0.0", false},
		// Regression: a stale cache written on an older install (v0.2.1) must not
		// keep nagging after the user has updated to a newer version.
		{"stale older cached version", info{Version: "v0.2.1"}, "1.0.0", false},
		// The stored Available flag must not force a notice on its own; the
		// decision is recomputed from the versions.
		{"stale Available:true ignored", info{Available: true, Version: "v0.2.1"}, "1.0.0", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldNotify(c.inf, c.current); got != c.want {
				t.Errorf("shouldNotify(%+v, %q) = %v, want %v", c.inf, c.current, got, c.want)
			}
		})
	}
}
