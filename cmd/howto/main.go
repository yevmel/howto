// Command howto turns a plain-language description into a shell command and runs it after confirmation.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/yevmel/howto/internal/config"
	"github.com/yevmel/howto/internal/llm"
	"github.com/yevmel/howto/internal/prompt"
	"github.com/yevmel/howto/internal/runner"
	"github.com/yevmel/howto/internal/ui"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	exitError       = 1
	exitUsage       = 2
	exitInterrupted = 130
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Parse(os.Args[1:], os.Getenv, os.Stderr)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return 0
	case err != nil:
		return exitUsage
	case cfg.Version:
		fmt.Println("howto", version)
		return 0
	}

	out := ui.New(os.Stderr, os.Getenv)
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()

	shell := prompt.Shell(os.Getenv)
	client := llm.New(cfg.URL, cfg.Token)
	var command string
	var interactive bool
	err = out.Spin(ctx, "Thinking…", func(ctx context.Context) error {
		model := cfg.Model
		if model == "" {
			var err error
			if model, err = client.DetectModel(ctx); err != nil {
				return err
			}
		}
		answer, err := client.Complete(ctx, model, prompt.System(prompt.OSDescription(), shell), cfg.Description)
		if err != nil {
			return err
		}
		command, interactive, err = prompt.Command(answer)
		return err
	})
	if interrupted(ctx, err) {
		return exitInterrupted
	}
	if err != nil {
		out.Error(err)
		return exitError
	}

	out.ShowCommand(command)
	question := "Execute this command?"
	if interactive {
		question = "Execute this command? (interactive mode)"
	}
	choice := out.Choose(ctx, os.Stdin, question)
	if choice == ui.Nevermind {
		out.Info("Not executed.")
		return 0
	}

	// From here on, Ctrl-C belongs to the command.
	stopSignals()
	code, err := runner.Run(runner.ShellPath(shell), command, runner.Options{
		Sudo:        choice == ui.ExecuteSudo,
		Interactive: interactive,
	})
	if err != nil {
		out.Error(err)
		return exitError
	}
	return code
}

func interrupted(ctx context.Context, err error) bool {
	return ctx.Err() != nil || errors.Is(err, context.Canceled)
}
