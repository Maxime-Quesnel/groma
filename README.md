<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/groma-logo-dark.svg">
    <img src="assets/groma-logo-light.svg" alt="groma" width="360">
  </picture>
</p>

<p align="center">
  <strong>Draw the perimeter around your AI agents.</strong>
</p>

## Why

Before a Roman legion set up camp, a surveyor planted a **groma** in the ground and traced the perimeter. Only then did the soldiers dig the ditch and raise the palisade. Nothing was built before the boundaries were drawn.

AI agents skip that step. People install an agent on a VPS, give it a shell, their API keys and a dozen community skills, then expose it to the internet so they can chat with it from their phone. Most never check what it can reach, what it exposes, or what those skills actually do.

`groma` draws the perimeter for you:

- **what gets in**: skills, plugins, hooks and MCP servers, checked before you trust them;
- **what gets out**: ports, dashboards and tunnels your agent exposes to the world;
- **what it can touch**: files, secrets and commands within its reach.

## Install

groma is a single binary with no runtime dependency, for macOS and Linux. No release is published yet, so install it from source with [Go](https://go.dev/dl/) 1.27 or later:

```sh
go install github.com/Maxime-Quesnel/groma/cmd/groma@latest
```

The binary lands in `$(go env GOPATH)/bin`, usually `~/go/bin`. Add that directory to your `PATH` if it isn't there yet:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

Or build it from a clone:

```sh
git clone https://github.com/Maxime-Quesnel/groma.git
cd groma
go build -o groma ./cmd/groma
```

## Usage

Check everything Claude Code loads on your machine: your settings, skills, agents and hooks, every installed plugin, and the `.claude` directory of the project you run it from:

```sh
groma scan
```

Check one plugin, marketplace or skill directory, for instance before publishing or installing it:

```sh
groma scan ./my-plugin
```

groma exits with 0 when it finds nothing, 1 when it has findings and 2 on error, so a CI job can fail on findings. It only reads: it never modifies a file, never runs the code it inspects, and sends nothing over the network.

## Benchmark a plugin's agents

`groma bench` measures, for each agent of a plugin, how precisely Claude routes work to it and how well the agent then does the work. Its ground truth is the plugin's own `claude plugin eval` suite: the cases that require or forbid an agent.

Run it with no argument from a plugin or marketplace directory. It asks for the agents to score (space to check, `a` for all), the plugin first when there are several, and how thorough to be: standard, a quick look, precise, or only the plan. Everything else takes sensible defaults: two runs at once, Sonnet as the judge, and your Claude Code model. Eval runs start from a blank Claude Code config, so groma passes that model on: the model of the Claude Code session you run groma from, even one picked with `/model` for that session only, or else the one `/model` saved for new sessions. `--model` overrides it. The report names the main session's model, and each agent's own model when its frontmatter sets one.

```sh
groma bench
```

It then prints the equivalent command, to run the same benchmark again or in CI:

```sh
groma bench --agent rails-expert --runs 3 --concurrency 2 --judge-model sonnet --scaffold --allow-tools Edit,Write plugins/my-plugin
```

The report gives each agent's precision and recall with 95% intervals, what Claude did instead when it missed, and the share of the work checks passed, split between exact checks and a judge model's verdicts. Unlike `scan`, `bench` runs Claude: through your own Claude Code login, with your model unless you pick another, so it counts against your plan. It works on a copy of the plugin in `./groma-bench/` and never publishes the report.

## What it checks

| Rule | Severity | Flags |
|---|---|---|
| `scan.hidden-unicode` | high | Text a reviewer can't see: Unicode tag characters and variation selectors hiding a message, bidirectional controls, zero-width characters |
| `scan.hook-runs-remote-code` | high | A hook, or the inline shell of a skill or command, that downloads code and runs it, directly or through a script it calls |
| `scan.reads-credentials` | high | Plugin code or a hook that reaches for SSH keys, cloud credentials, registry tokens, the keychain, browser logins or Claude's credentials |
| `scan.preapproves-any-command` | medium | A skill or command whose `allowed-tools` lets Claude run any shell command without asking |

Each finding names the file and the line or command behind it; common secret shapes in evidence, such as tokens and passwords in URLs, are masked. groma is a safety net, not a guarantee: every rule documents its known false positives, and a clean report means no known pattern was found, not that a plugin is safe.
