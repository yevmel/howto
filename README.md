# howto

**Describe what you want to do — get a shell command, and run it after you confirm.**

```console
$ howto find all files larger than 100 MB in my home directory

╭──────────────────────────────────────────────────────────╮
│ find ~ -type f -size +100M -exec ls -lh {} + 2>/dev/null │
╰──────────────────────────────────────────────────────────╯

Execute this command?
❯ Execute (Enter)
  Execute with sudo
  Nevermind (Esc)
```

`howto` sends your description to any OpenAI-compatible LLM endpoint — a local [Ollama](https://ollama.com), LM Studio, vLLM or a hosted API — and shows the suggested command with syntax highlighting. Nothing runs until you choose to execute it.

## Features

- **Safe by default** — every command needs an explicit choice. Keys typed while the model is still thinking are discarded, so you can't confirm a command you haven't seen.
- **Fits your system** — the model is told your OS (e.g. `macOS 26.1`, `Linux (Ubuntu 24.04 LTS)`) and your shell, and is asked for tools installed by default.
- **Works with any OpenAI-compatible endpoint** — local or hosted; the model can be detected automatically.
- **Nice terminal UI** — a spinner while waiting, the command highlighted in a box, colors that adapt to light and dark terminals. Respects [`NO_COLOR`](https://no-color.org).
- **Script-friendly** — `howto`'s own output goes to stderr; stdout carries only the command's output. The command's exit status becomes `howto`'s exit status.
- **Single static binary** — no runtime, no dependencies. macOS and Linux, amd64 and arm64.

## Installation

### With Homebrew

On macOS and Linux:

```sh
brew install yevmel/tap/howto
```

Upgrade later with `brew upgrade howto`.

### From a release

Download the archive for your platform from the [releases page](../../releases), unpack it and put `howto` on your `PATH`:

```sh
tar -xzf howto_*_darwin_arm64.tar.gz
mv howto /usr/local/bin/
```

### With Go

Requires [Go](https://go.dev) 1.27 or newer. The binary lands in `$(go env GOPATH)/bin`, usually `~/go/bin`:

```sh
go install github.com/yevmel/howto/cmd/howto@latest
```

### From source

```sh
git clone https://github.com/yevmel/howto.git
cd howto
make build
cp bin/howto /usr/local/bin/
```

## Quick start

Point `howto` at an endpoint, for example a local Ollama:

```sh
export HOWTO_LLM_URL=http://localhost:11434/v1
export HOWTO_LLM_MODEL=llama3.1   # optional: defaults to the first model the endpoint lists
```

Then ask:

```sh
howto show the 10 biggest directories under the current folder
howto list all processes listening on a TCP port
howto convert every .png in this folder to .jpg
```

Put the `export` lines in your shell profile (`~/.zshrc`, `~/.bashrc`) to keep the configuration.

## Configuration

Each setting can be given as a flag or as an environment variable. **A flag wins over the environment variable.**

| Flag | Environment variable | Required | Description |
|---|---|---|---|
| `--url URL` | `HOWTO_LLM_URL` | yes | Base URL of an OpenAI-compatible endpoint, e.g. `http://localhost:11434/v1`. A full `…/chat/completions` URL works too. |
| `--model MODEL` | `HOWTO_LLM_MODEL` | no | Model name. Defaults to the first model reported by the endpoint's `/models` route. |
| `--token TOKEN` | `HOWTO_LLM_TOKEN` | no | Sent as `Authorization: Bearer <token>`. |
| `--version` | | | Print the version and exit. |
| `-h`, `--help` | | | Show usage and exit. |

> [!TIP]
> Prefer `HOWTO_LLM_TOKEN` over `--token`: flag values end up in your shell history and in process listings.

Options must come before the description. Use `--` if the description itself starts with a dash:

```sh
howto --model qwen2.5-coder -- -rf in rm: delete the build directory without prompting
```

Other environment variables:

| Variable | Effect |
|---|---|
| `NO_COLOR` | Turns off colors and syntax highlighting. |
| `SHELL` | Shell named to the model and used to run the command (`/bin/sh` if unset). |

## How it works

1. Your description is sent to `<url>/chat/completions` together with a system prompt naming your OS and shell. The model is asked to reply with a single command only.
2. Code fences or backticks the model adds anyway are stripped. If the request can't be expressed as a shell command, `howto` says so instead of guessing.
3. The command is shown with a menu: pick *Execute*, *Execute with sudo* or *Nevermind* with the arrow keys (or `j`/`k`) and press Enter. Esc and Ctrl-C cancel.
4. The command runs in your shell with your terminal attached. Ctrl-C goes to the command.

When stdin is not a terminal, `howto` reads a whole line instead: `y`/`yes` executes, `s`/`sudo` executes with sudo, anything else cancels. When stderr is not a terminal, the spinner, box and colors are left out.

## Exit status

| Situation | Status |
|---|---|
| Command executed | the command's exit status (with sudo: sudo's exit status) |
| Command killed by a signal | 128 + signal number (130 for Ctrl-C) |
| Ctrl-C while waiting for the model | 130 |
| Not confirmed | 0 |
| Endpoint, model or response error | 1 |
| Usage error (no description, no URL, unknown flag) | 2 |

## Development

| Command | Effect |
|---|---|
| `make build` | Build `bin/howto` (set the version with `make build VERSION=1.2.3`) |
| `make test` | Run the tests |
| `make lint` | Run `go vet` and [golangci-lint](https://golangci-lint.run) |
| `make fmt` | Format the code |
| `make snapshot` | Build release archives for all platforms into `dist/` with [GoReleaser](https://goreleaser.com) |

Project layout:

```
cmd/howto/          entry point
internal/config/    flags and environment variables
internal/llm/       OpenAI-compatible client
internal/prompt/    system prompt, OS and shell detection, answer cleanup
internal/ui/        spinner, command box, execution menu
internal/runner/    runs the command in your shell
```

Releases are built by GoReleaser from a version tag:

```sh
git tag v1.0.0
goreleaser release --clean
```
