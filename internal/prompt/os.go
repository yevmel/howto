package prompt

import (
	"bufio"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// OSDescription describes the operating system for the model, e.g. "macOS 26.1" or "Linux (Ubuntu 24.04 LTS)".
func OSDescription() string {
	switch runtime.GOOS {
	case "darwin":
		if v := macOSVersion(); v != "" {
			return "macOS " + v
		}
		return "macOS"
	case "linux":
		for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
			if f, err := os.Open(path); err == nil {
				name := prettyName(f)
				_ = f.Close()
				if name != "" {
					return "Linux (" + name + ")"
				}
			}
		}
		return "Linux"
	default:
		return runtime.GOOS
	}
}

// prettyName reads PRETTY_NAME from an os-release file.
func prettyName(r io.Reader) string {
	s := bufio.NewScanner(r)
	for s.Scan() {
		value, ok := strings.CutPrefix(strings.TrimSpace(s.Text()), "PRETTY_NAME=")
		if !ok {
			continue
		}
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
		return strings.Trim(value, `"'`)
	}
	return ""
}
