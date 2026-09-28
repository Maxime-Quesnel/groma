<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/groma-logo-dark.svg">
    <img src="assets/groma-logo-light.svg" alt="groma" width="360">
  </picture>
</p>

<p align="center">
  <strong>Check how your Claude Code skills, agents, commands and hooks are written.</strong>
</p>

## Why

Before a Roman legion set up camp, a surveyor planted a **groma** and traced the lines. Nothing was built until they were straight.

A Claude Code plugin is a handful of Markdown and JSON files, and Claude Code forgives almost everything in them without a word. A misspelled field is ignored. A tool that no longer exists is dropped. A hook on `preToolUse` never fires. An agent named `shop:reviewer` is never loaded. A `permissionMode` on a plugin agent doesn't apply. The plugin looks fine, and part of it quietly doesn't work, or does more than its author meant.

`groma check` reads every skill, agent, command and hook of a plugin and holds them against Claude Code's documentation and Anthropic's authoring guidance. It lists what isn't written the way it should be, and ends with the red flags: what Claude Code won't load, won't run or ignores, and what puts the user at risk.

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

Point it at a plugin, a marketplace, a `.claude` directory or a single file. groma works out what each file is from where Claude Code looks for it: `SKILL.md` is a skill, a Markdown file under `agents/` an agent, under `commands/` a command, and `hooks/hooks.json`, a manifest's `hooks` or a settings file's `hooks` are hooks. A file found elsewhere is read by its content.

```sh
groma check ./my-plugin
groma check ./my-plugin/agents/reviewer.md
groma check ~/.claude/plugins/marketplaces/my-marketplace
```

```
Checking ./my-plugin · 2 skills, 3 agents, 1 hooks file

  ✖ agent   agents/reviewer.md      1 red flag ↓
  ✔ agent   agents/tester.md
  ✔ agent   agents/writer.md
  ✔ hooks   hooks/hooks.json
  ▲ skill   skills/deploy/SKILL.md  1 warning
      Claude can start a workflow with side effects on its own · side-effects-model-invocable
      › named deploy, without disable-model-invocation: true
      Fix: Add disable-model-invocation: true, unless Claude is meant to run it unprompted.
  ✔ skill   skills/pdf/SKILL.md

Red flags
  ✖ agents/reviewer.md
      Tool Claude Code doesn't have · unknown-tool
      › line 4: tools lists MultiEdit, which Claude Code no longer has; use Edit
      Fix: Use the tool's exact name, as the Claude Code tools reference spells it.

✖ 1 red flag · 1 warning in 6 components
```

groma exits with 0 when it finds no red flag, 1 when it finds at least one, and 2 on error, so a CI job can fail on red flags while warnings stay advice. It only reads: it never modifies a file, never runs the code it inspects, and sends nothing over the network. Secrets quoted in evidence, such as tokens in a hook command, are masked.

## What it checks

Every rule cites its source, the Claude Code documentation or Anthropic guidance it enforces or the risk it guards against, and documents its known false positives next to its code.

### Red flags: security

| Rule | Flags |
|---|---|
| `hidden-unicode` | Text a reviewer can't see: tag characters and variation selectors hiding a message, bidirectional controls, zero-width characters |
| `runs-remote-code` | A hook or an inline shell command that downloads code and runs it, directly or through a script it calls |
| `reads-credentials` | A hook, inline shell or shipped script that reaches for SSH keys, cloud credentials, tokens, the keychain or browser logins |
| `preapproves-any-command` | `allowed-tools` that lets Claude run any shell command without asking |
| `arguments-in-shell` | `$ARGUMENTS` or `$1` substituted, unquoted, into a command that runs without a prompt |
| `prompt-injection` | Text telling Claude to ignore its instructions or to keep something from the user |
| `bypass-permissions` | An agent that runs every tool without asking |

### Red flags: what Claude Code won't load, won't run or ignores

| Rule | Flags |
|---|---|
| `frontmatter-unreadable` | YAML that doesn't parse, or a header not on line 1: every field is dropped |
| `agent-not-loaded` | An agent without a name or description, or named with `:` |
| `field-typo` | `allowed_tools`, `tools` in a skill, `allowed-tools` in an agent: the setting silently doesn't apply |
| `invalid-value` | A model, effort, colour, permission mode or yes/no value Claude Code doesn't accept |
| `field-ignored` | A real field where it doesn't apply, such as `permissionMode` on a plugin agent or `name` on a command |
| `unknown-tool` | A tool Claude Code doesn't have, including renamed or removed ones such as `MultiEdit` |
| `tool-unavailable-to-agents` | `AskUserQuestion` and other tools no subagent ever gets |
| `duplicate-name` | Two agents, or a skill and a command, with the same name |
| `shell-not-preapproved` | Inline shell that `allowed-tools` doesn't cover, which aborts the skill outside auto mode |
| `missing-script` | A hook or inline shell running a script that isn't there or isn't executable |
| `hook-file-invalid` | A hooks file that isn't valid JSON or lacks the `{"hooks": ...}` shape |
| `hook-unknown-event` | A hook on an event that doesn't exist, such as `preToolUse` |
| `hook-handler-invalid` | A hook without a valid type, its required field, or a usable timeout |
| `hook-field-ignored` | A `matcher` inside a hook, `once` in a hooks file, a header variable missing from `allowedEnvVars` |
| `hook-never-runs` | A lowercase tool matcher, or an `if` condition on an event that never evaluates it |
| `hook-matcher-ignored` | A matcher on an event that fires every time, such as `Stop` |
| `hook-user-config-in-shell` | `${user_config.…}` in a shell-form command, which Claude Code refuses |

### Warnings: works, but not as Claude Code's documentation recommends

| Rule | Flags |
|---|---|
| `unknown-field` | A field Claude Code's documentation doesn't list, such as `version` |
| `name-format` | A name that isn't lowercase with hyphens |
| `description-missing` | No description to route on |
| `description-too-long` | Over 1,024 characters, or cut by Claude Code's 1,536-character listing |
| `description-no-trigger` | A description that doesn't say when to use it |
| `description-voice` | A description in the first or second person |
| `body-empty` | Nothing after the frontmatter |
| `body-too-long` | A skill or command body over 500 lines |
| `broken-link` | A link or `${CLAUDE_SKILL_DIR}` path to a file that isn't there |
| `nested-reference` | A reference reachable only through another reference |
| `non-portable-path` | A path into someone's home directory, or with backslashes |
| `agent-tools-unrestricted` | An agent that can use every tool |
| `argument-hint-missing` | A component that takes arguments without an `argument-hint` |
| `side-effects-model-invocable` | A deploy, release or push workflow Claude can start on its own |
| `unquoted-path` | `${CLAUDE_PLUGIN_ROOT}` left unquoted in a shell command |
| `stop-hook-can-loop` | A Stop hook that can block without reading `stop_hook_active` |

groma is a safety net, not a guarantee. It checks against what Claude Code's documentation says today, so a field or tool added by a newer release may show up as unknown until groma learns it; and a clean report means no known mistake was found, not that a plugin is safe.
