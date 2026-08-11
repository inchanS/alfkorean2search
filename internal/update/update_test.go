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
