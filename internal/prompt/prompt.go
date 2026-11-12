// Package prompt builds the system prompt and turns the model's answer into a command.
package prompt

import (
	"errors"
	"strings"
)

// NoCommand is the answer the model gives when a request cannot be a shell command.
const NoCommand = "HOWTO_NO_COMMAND"

// ErrNoCommand means the model did not produce a usable command.
var ErrNoCommand = errors.New("the model could not turn this request into a command")

const systemTemplate = `You translate a user's plain-language request into a single shell command.

Environment:
- Operating system: {os}
- Shell: {shell}

Rules:
- Reply with the command only: no explanation, no Markdown, no code fences, no surrounding quotes.
- The command must work in the given shell on the given operating system.
- If several steps are needed, combine them into one line with && or pipes.
- Prefer standard tools that are installed by default on this system.
- If the request cannot be expressed as a shell command, reply with exactly: {no_command}
`

// System returns the system prompt for the given OS description and shell.
func System(osName, shell string) string {
	return strings.NewReplacer("{os}", osName, "{shell}", shell, "{no_command}", NoCommand).Replace(systemTemplate)
}

// Shell returns the user's shell as reported by $SHELL, or /bin/sh.
func Shell(getenv func(string) string) string {
	if s := getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}

// Command cleans the model's answer and returns the command, or ErrNoCommand.
func Command(answer string) (string, error) {
	cmd := Clean(answer)
	if cmd == "" || cmd == NoCommand {
		return "", ErrNoCommand
	}
	return cmd, nil
}

// Clean strips Markdown code fences or inline backticks a model may add anyway.
func Clean(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")[1:] // drop the opening fence and language tag
		if n := len(lines); n > 0 && strings.HasPrefix(strings.TrimSpace(lines[n-1]), "```") {
			lines = lines[:n-1]
		}
		text = strings.TrimSpace(strings.Join(lines, "\n"))
	}
	if len(text) >= 2 && text[0] == '`' && text[len(text)-1] == '`' {
		text = strings.TrimSpace(strings.Trim(text, "`"))
	}
	return text
}
