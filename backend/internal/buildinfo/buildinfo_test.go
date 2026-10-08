package buildinfo

import "testing"

func TestResolveVersion(t *testing.T) {
	cases := []struct{ stamped, app, sha, want string }{
		{"dev", "", "", "dev"},                                     // local go run
		{"dev", "", "8704c41abcdef0123", "8704c41"},                // Vercel Git build
		{"dev", "a605f53e1b2c3d4", "8704c41abcdef0123", "a605f53"}, // deploy workflow wins
		{"94b0c1e", "a605f53", "8704c41abcdef0123", "94b0c1e"},     // Docker ldflags win
		{"dev", "", "abc", "dev"},                                  // malformed SHA ignored
	}
	for _, c := range cases {
		if got := resolve(c.stamped, c.app, c.sha); got != c.want {
			t.Errorf("resolve(%q, %q, %q) = %q, want %q", c.stamped, c.app, c.sha, got, c.want)
		}
	}
}
