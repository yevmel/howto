// Package config reads howto's settings from flags and environment variables.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

// Environment variables that provide defaults for the flags.
const (
	EnvURL   = "HOWTO_LLM_URL"
	EnvModel = "HOWTO_LLM_MODEL"
	EnvToken = "HOWTO_LLM_TOKEN"
)

// Config holds the settings for one invocation.
type Config struct {
	URL         string
	Model       string // empty: detect from the endpoint
	Token       string // empty: no Authorization header
	Description string
	Version     bool
}

// UsageError is a problem with the command line; the caller exits with status 2.
type UsageError struct{ msg string }

func (e *UsageError) Error() string { return e.msg }

// Parse reads flags from args, falling back to environment variables via getenv.
// It returns flag.ErrHelp when help was requested; the usage text is then already written to out.
func Parse(args []string, getenv func(string) string, out io.Writer) (*Config, error) {
	var cfg Config
	fs := flag.NewFlagSet("howto", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&cfg.URL, "url", getenv(EnvURL), "OpenAI-compatible endpoint base URL (default: $"+EnvURL+")")
	fs.StringVar(&cfg.Model, "model", getenv(EnvModel), "model name (default: $"+EnvModel+", else first model from /models)")
	fs.StringVar(&cfg.Token, "token", getenv(EnvToken), `token sent as "Authorization: Bearer <token>" (default: $`+EnvToken+")")
	fs.BoolVar(&cfg.Version, "version", false, "print the version and exit")
	fs.Usage = func() { usage(fs) }

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, err
		}
		// The flag package has already printed the error and usage.
		return nil, &UsageError{msg: err.Error()}
	}
	if cfg.Version {
		return &cfg, nil
	}

	cfg.Description = strings.TrimSpace(strings.Join(fs.Args(), " "))
	if cfg.Description == "" {
		return nil, usageError(fs, "please describe the command you need")
	}
	if cfg.URL == "" {
		return nil, usageError(fs, "no endpoint configured; use --url or set "+EnvURL)
	}
	return &cfg, nil
}

func usageError(fs *flag.FlagSet, msg string) error {
	fmt.Fprintf(fs.Output(), "howto: error: %s\n\nRun 'howto --help' for usage.\n", msg)
	return &UsageError{msg: msg}
}

func usage(fs *flag.FlagSet) {
	w := fs.Output()
	fmt.Fprint(w, `Usage: howto [options] <description of what you want to do>

Describe what you want to do; get a shell command and optionally run it.

Example:
  howto find all files larger than 100 MB in my home directory

Options (a flag wins over its environment variable):
  --url URL        OpenAI-compatible endpoint base URL        [$`+EnvURL+`]
  --model MODEL    model name; default: first from /models    [$`+EnvModel+`]
  --token TOKEN    sent as "Authorization: Bearer <token>"    [$`+EnvToken+`]
  --version        print the version and exit
  -h, --help       show this help

Options must come before the description. Use -- if the description starts with '-'.
Prefer $`+EnvToken+` over --token: a flag value ends up in shell history.
`)
}
