package prompt

import (
	"errors"
	"runtime"
	"strings"
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct{ in, want string }{
		{"ls -la", "ls -la"},
		{"  ls -la\n", "ls -la"},
		{"```\nls -la\n```", "ls -la"},
		{"```bash\nls -la\n```", "ls -la"},
		{"```sh\nfind . -name '*.go' |\n  wc -l\n```", "find . -name '*.go' |\n  wc -l"},
		{"```bash\nls -la", "ls -la"},
		{"`ls -la`", "ls -la"},
		{"echo `date`", "echo `date`"},
		{"`", "`"},
	}
	for _, tt := range tests {
		if got := Clean(tt.in); got != tt.want {
			t.Errorf("Clean(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		in, want    string
		interactive bool
	}{
		{"```\nls\n```", "ls", false},
		{Interactive + "\nnvm use 20", "nvm use 20", true},
		{Interactive + " nvm use 20", "nvm use 20", true},
		{"```sh\n" + Interactive + "\nnvm ls\n```", "nvm ls", true},
		{Interactive + "\n```\nnvm ls\n```", "nvm ls", true},
	}
	for _, tt := range tests {
		got, interactive, err := Command(tt.in)
		if err != nil || got != tt.want || interactive != tt.interactive {
			t.Errorf("Command(%q) = %q, %v, %v; want %q, %v", tt.in, got, interactive, err, tt.want, tt.interactive)
		}
	}
	for _, in := range []string{"", "  ", "```\n```", NoCommand, "`" + NoCommand + "`", Interactive, Interactive + "\n" + NoCommand} {
		if _, _, err := Command(in); !errors.Is(err, ErrNoCommand) {
			t.Errorf("Command(%q): got %v, want ErrNoCommand", in, err)
		}
	}
}

func TestSystem(t *testing.T) {
	got := System("macOS 26.1", "/bin/zsh")
	for _, want := range []string{"Operating system: macOS 26.1", "Shell: /bin/zsh", "reply with exactly: " + NoCommand, "exactly " + Interactive + " before"} {
		if !strings.Contains(got, want) {
			t.Errorf("system prompt lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "{") {
		t.Errorf("unreplaced placeholder:\n%s", got)
	}
}

func TestShell(t *testing.T) {
	if got := Shell(func(string) string { return "" }); got != "/bin/sh" {
		t.Errorf("fallback: got %q", got)
	}
	if got := Shell(func(string) string { return "/bin/zsh" }); got != "/bin/zsh" {
		t.Errorf("got %q", got)
	}
}

func TestPrettyName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"NAME=\"Ubuntu\"\nPRETTY_NAME=\"Ubuntu 24.04 LTS\"\n", "Ubuntu 24.04 LTS"},
		{"PRETTY_NAME='Alpine Linux v3.20'", "Alpine Linux v3.20"},
		{"PRETTY_NAME=Arch", "Arch"},
		{"NAME=x", ""},
	}
	for _, tt := range tests {
		if got := prettyName(strings.NewReader(tt.in)); got != tt.want {
			t.Errorf("prettyName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestOSDescription(t *testing.T) {
	got := OSDescription()
	if runtime.GOOS == "darwin" && !strings.HasPrefix(got, "macOS ") {
		t.Errorf("got %q, want a macOS version", got)
	}
	t.Logf("OS description: %s", got)
}
