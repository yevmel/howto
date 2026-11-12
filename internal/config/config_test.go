package config

import (
	"errors"
	"flag"
	"io"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestParse(t *testing.T) {
	baseEnv := map[string]string{EnvURL: "http://env/v1", EnvModel: "env-model", EnvToken: "env-token"}

	tests := []struct {
		name string
		args []string
		env  map[string]string
		want Config
	}{
		{
			name: "environment only",
			args: []string{"list", "files"},
			env:  baseEnv,
			want: Config{URL: "http://env/v1", Model: "env-model", Token: "env-token", Description: "list files"},
		},
		{
			name: "flags win over environment",
			args: []string{"--url", "http://flag/v1", "--model=flag-model", "--token", "flag-token", "list", "files"},
			env:  baseEnv,
			want: Config{URL: "http://flag/v1", Model: "flag-model", Token: "flag-token", Description: "list files"},
		},
		{
			name: "empty token flag disables the environment token",
			args: []string{"--token", "", "x"},
			env:  baseEnv,
			want: Config{URL: "http://env/v1", Model: "env-model", Description: "x"},
		},
		{
			name: "model is optional",
			args: []string{"--url", "http://u", "x"},
			env:  nil,
			want: Config{URL: "http://u", Description: "x"},
		},
		{
			name: "double dash allows a leading dash",
			args: []string{"--", "-rf", "explained"},
			env:  baseEnv,
			want: Config{URL: "http://env/v1", Model: "env-model", Token: "env-token", Description: "-rf explained"},
		},
		{
			name: "options after the description belong to it",
			args: []string{"show", "--url", "x"},
			env:  baseEnv,
			want: Config{URL: "http://env/v1", Model: "env-model", Token: "env-token", Description: "show --url x"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args, env(tt.env), io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if *got != tt.want {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{"no description", nil, map[string]string{EnvURL: "http://u"}},
		{"blank description", []string{" "}, map[string]string{EnvURL: "http://u"}},
		{"no url", []string{"x"}, nil},
		{"unknown flag", []string{"--nope", "x"}, map[string]string{EnvURL: "http://u"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.args, env(tt.env), io.Discard)
			var ue *UsageError
			if !errors.As(err, &ue) {
				t.Fatalf("got %v, want a UsageError", err)
			}
		})
	}
}

func TestParseHelpAndVersion(t *testing.T) {
	if _, err := Parse([]string{"--help"}, env(nil), io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("--help: got %v, want flag.ErrHelp", err)
	}
	cfg, err := Parse([]string{"--version"}, env(nil), io.Discard)
	if err != nil || !cfg.Version {
		t.Errorf("--version: got %+v, %v", cfg, err)
	}
}
