package cleaner

import "testing"

func TestTryParseSize(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"2.5GiB", 2684354560},
		{"1.2G", 1288490188},
		{"0B", 0},
		{"512", 512},
		{"100M", 104857600},
		{"3K", 3072},
		{"", 0},
		{"abc", 0},
	}
	for _, c := range cases {
		if got := tryParseSize(c.in); got != c.want {
			t.Errorf("tryParseSize(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseTrailingSize(t *testing.T) {
	out := "==&gt; finished: 45 packages found (no change to 0), 2.5 GiB freed\n"
	out = "==> finished: 45 packages found (no change to 0), 2.5 GiB freed\n"
	if got := parseTrailingSize(out, "freed"); got != 2684354560 {
		t.Errorf("parseTrailingSize paccache = %d, want 2684354560", got)
	}
	if got := parseTrailingSize("nothing here", "freed"); got != 0 {
		t.Errorf("parseTrailingSize no match = %d, want 0", got)
	}
}

func TestParseLeadingSize(t *testing.T) {
	out := "Vacuuming done, freed 1.2G of archived journals from '/var/log/journal'.\n"
	if got := parseLeadingSize(out, "freed"); got != 1288490188 {
		t.Errorf("parseLeadingSize = %d, want 1288490188", got)
	}
	out0 := "Vacuuming done, freed 0B of archived journals from '/var/log/journal'.\n"
	if got := parseLeadingSize(out0, "freed"); got != 0 {
		t.Errorf("parseLeadingSize zero = %d, want 0", got)
	}
}

func TestParsePaccacheCleaned(t *testing.T) {
	out := "==> finished: 45 packages found (no change to 12), 2.5 GiB freed\n"
	if got := parsePaccacheCleaned(out); got != 33 {
		t.Errorf("parsePaccacheCleaned = %d, want 33", got)
	}
	out2 := "==> finished: 10 packages found (no change to 0), 100 MiB freed\n"
	if got := parsePaccacheCleaned(out2); got != 10 {
		t.Errorf("parsePaccacheCleaned = %d, want 10", got)
	}
}

func TestIsProtectedPath(t *testing.T) {
	protected := []string{
		"/", "/tmp", "/var/tmp",
		"/tmp/.X11-unix", "/tmp/.ICE-unix",
		"/tmp/systemd-private-abc-xyz", "/tmp/snap-private-tmp",
		"/var/tmp/systemd-private-foo",
	}
	for _, p := range protected {
		if !isProtectedPath(p) {
			t.Errorf("isProtectedPath(%q) = false, want true", p)
		}
	}
	allowed := []string{
		"/var/cache/pacman/pkg", "/home/user/.cache/thumbnails",
		"/var/tmp/firefox.old", "/tmp/some-app-cache",
	}
	for _, p := range allowed {
		if isProtectedPath(p) {
			t.Errorf("isProtectedPath(%q) = true, want false", p)
		}
	}
}
