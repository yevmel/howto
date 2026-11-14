package runner

import "testing"

func TestRunExitStatus(t *testing.T) {
	tests := []struct {
		command string
		want    int
	}{
		{"true", 0},
		{"exit 3", 3},
		{"kill -INT $$", 130},
		{"kill -TERM $$", 143},
	}
	for _, tt := range tests {
		got, err := Run("/bin/sh", tt.command, false)
		if err != nil || got != tt.want {
			t.Errorf("Run(%q) = %d, %v; want %d", tt.command, got, err, tt.want)
		}
	}
}

func TestRunMissingShell(t *testing.T) {
	if _, err := Run("/nonexistent/shell", "true", false); err == nil {
		t.Error("expected an error for a missing shell")
	}
}

func TestShellPath(t *testing.T) {
	if got := ShellPath("/nonexistent/shell"); got != "/bin/sh" {
		t.Errorf("got %q, want /bin/sh", got)
	}
	if got := ShellPath(""); got != "/bin/sh" {
		t.Errorf("got %q, want /bin/sh", got)
	}
	if got := ShellPath("/bin/sh"); got != "/bin/sh" {
		t.Errorf("got %q", got)
	}
}
