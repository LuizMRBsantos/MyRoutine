package buildinfo

import "testing"

func TestResolveVersion(t *testing.T) {
	cases := []struct{ stamped, sha, want string }{
		{"dev", "", "dev"},                          // local go run
		{"dev", "8704c41abcdef0123", "8704c41"},     // Vercel build
		{"94b0c1e", "8704c41abcdef0123", "94b0c1e"}, // Docker ldflags win
		{"dev", "abc", "dev"},                       // malformed SHA ignored
	}
	for _, c := range cases {
		if got := resolve(c.stamped, c.sha); got != c.want {
			t.Errorf("resolve(%q, %q) = %q, want %q", c.stamped, c.sha, got, c.want)
		}
	}
}
