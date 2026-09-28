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

Point it at a plugin, a marketplace, a `.claude` directory or a single file. groma works out what each file is from where Claude Code looks for it: `.claude-plugin/plugin.json` is a plugin, `SKILL.md` a skill, a Markdown file under `agents/` an agent, under `commands/` a command, and `hooks/hooks.json`, a manifest's `hooks` or a settings file's `hooks` are hooks. A file found elsewhere is read by its content.

```sh
groma check ./my-plugin
groma check ./my-plugin/agents/reviewer.md
groma check ~/.claude/plugins/marketplaces/my-marketplace
```

```
Checking ./my-plugin · 1 plugin, 2 skills, 3 agents, 1 hooks file

  ✔ plugin  .claude-plugin/plugin.json
  ✖ agent   agents/reviewer.md          1 red flag ↓
  ✔ agent   agents/tester.md
  ✔ agent   agents/writer.md
  ✔ hooks   hooks/hooks.json
  ▲ skill   skills/deploy/SKILL.md      1 warning
      Claude can start a workflow with side effects on its own · side-effects-model-invocable
      › named deploy, without disable-model-invocation: true
      Fix: Add disable-model-invocation: true, unless Claude is meant to run it unprompted.
  ✔ skill   skills/pdf/SKILL.md

Red flags
  ✖ agents/reviewer.md
      Tool Claude Code doesn't have · unknown-tool
      › line 4: tools lists MultiEdit, which Claude Code no longer has; use Edit
      Fix: Use the tool's exact name, as the Claude Code tools reference spells it.

✖ 1 red flag · 1 warning in 7 components
```

groma exits with 0 when it finds no red flag, 1 when it finds at least one, and 2 on error, so a CI job can fail on red flags while warnings stay advice. `check` only reads: it never modifies a file, never runs the code it inspects, and sends nothing over the network. Secrets quoted in evidence, such as tokens in a hook command, are masked.

## Fix what can be fixed

Like RuboCop's autocorrect, `groma fix` corrects what a rule can correct on its own. It shows the diff, then asks before writing anything, and writes nothing when it can't ask, such as in CI:

```sh
groma fix ./my-plugin
groma fix --unsafe ./my-plugin
```

```diff
--- a/plugins/my-plugin/hooks/hooks.json
+++ b/plugins/my-plugin/hooks/hooks.json
@@ -12,7 +12,7 @@
         ]
       },
       {
-        "matcher": "Edit|Write|MultiEdit",
+        "matcher": "Edit|Write",
         "hooks": [
           {
             "type": "command",

4 fixes in 2 files: hook-matcher-dead-alternative 3, reference-no-toc 1
31 more fixes change what runs, when, or with which tools: review with groma fix --unsafe
Apply these changes? [y/N]
```

Edits change the files' text in place, so their layout, comments and quoting stay as they are.

- **Safe fixes** change nothing where the component works today: they remove what Claude Code ignores, make a command robust, or add a table of contents.
  - `hook-matcher-dead-alternative`: drops `MultiEdit` and other removed tools from matchers
  - `unknown-tool`: drops a removed tool whose replacement is already listed
  - `tool-unavailable-to-agents`: drops `AskUserQuestion` and the like from an agent's tools
  - `unquoted-path`: quotes `${CLAUDE_PLUGIN_ROOT}` paths in shell commands
  - `reference-no-toc`: opens a long reference file with a Contents list of its sections
- **Unsafe fixes**, with `--unsafe`, change what runs, when, or with which tools, so review them:
  - `field-typo`: renames `allowed_tools` to `allowed-tools`, and the like
  - `unknown-tool`: replaces a removed tool, or corrects a tool name's case
  - `hook-never-runs`: corrects a matcher or `if` that never matches, such as `bash` or `MultiEdit(…)`
  - `hook-matcher-dead-alternative`: corrects a tool name's case in a matcher
  - `variable-not-substituted`: writes `$CLAUDE_PLUGIN_ROOT/` as `${CLAUDE_PLUGIN_ROOT}/`
  - `missing-script`: makes a script a hook runs directly executable
  - `side-effects-model-invocable`: adds `disable-model-invocation: true`
  - `description-emphatic`: rewrites MUST BE USED as Use proactively, and other capitals in lowercase
  - `hook-timeout-in-milliseconds`: reads a timeout over an hour as milliseconds

## Turn rules off

Like `.rubocop.yml`, a `.groma.yml` in the checked directory or any directory above it turns rules off, everywhere or for some paths. Paths are globs relative to the file, where `**` matches any number of directories:

```yaml
disable:
  - description-emphatic
exclude:
  - plugins/legacy/**
reference-no-toc:
  - plugins/my-plugin/skills/big-reference/**
```

A skill, agent or command can also turn rules off for itself, with a comment in its frontmatter, which Claude never reads:

```yaml
---
name: rails-expert
# groma:disable description-emphatic
description: MUST BE USED for Rails changes.
---
```

The report counts what the configuration silenced, so nothing disappears unnoticed. A rule ID groma doesn't know is an error.

## What it checks

Every rule cites its source, the Claude Code documentation or Anthropic guidance it enforces or the risk it guards against, and documents its known false positives next to its code. Besides skills, agents, commands and hooks, groma checks each plugin as a whole, from its manifest.

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
| `hook-approves-every-permission` | A hook that allows every permission prompt, or switches the session to `bypassPermissions` |

### Red flags: what Claude Code won't load, won't run or ignores

| Rule | Flags |
|---|---|
| `frontmatter-unreadable` | YAML that doesn't parse, or a header not on line 1: every field is dropped |
| `agent-not-loaded` | A project agent without a name or description, or any agent named with `:` |
| `component-not-loaded` | Components inside `.claude-plugin/`, agents or commands the manifest doesn't list, a skill folder without `SKILL.md` |
| `plugin-claude-md` | A `CLAUDE.md` at the plugin's root, which Claude Code never loads |
| `reserved-name` | A skill or command named `anthropic-skills`, or a skill folder named `synced` |
| `skill-never-invocable` | `user-invocable: false` with `disable-model-invocation: true`: nobody can run it |
| `field-typo` | `allowed_tools`, `tools` in a skill, `allowed-tools` in an agent: the setting silently doesn't apply |
| `invalid-value` | A model, effort, colour, permission mode or yes/no value Claude Code doesn't accept |
| `field-ignored` | A real field where it doesn't apply, such as `permissionMode` on a plugin agent or `name` on a command |
| `unknown-tool` | A tool Claude Code doesn't have, including renamed or removed ones such as `MultiEdit` |
| `tool-unavailable-to-agents` | `AskUserQuestion` and other tools no subagent ever gets |
| `tool-allowed-and-denied` | A tool in both `tools` and `disallowedTools`, which removes it |
| `disallowed-tool-specifier` | `disallowedTools: Bash(git push *)`, which removes all of Bash |
| `permission-pattern-never-matches` | `Bash(git:* push)`, or a tool-name glob such as `*` or `mcp__slack-*__…` in `allowed-tools` |
| `preloaded-skill-unavailable` | An agent preloading a skill with `disable-model-invocation: true`, or one its plugin doesn't have |
| `duplicate-name` | Two agents, or a skill and a command, with the same name |
| `unknown-component-reference` | `plugin:name`, `/plugin:name` or `@agent-plugin:name` naming nothing in that plugin |
| `variable-not-substituted` | `$CLAUDE_PLUGIN_ROOT/` without braces, `${CLAUDE_PLUGIN_ROOT}` outside a plugin, `$ARGUMENTS.0`, `$IF(` |
| `user-config-reference` | `${user_config.KEY}` the manifest doesn't declare, or a sensitive one in skill text |
| `path-escapes-plugin` | A link or `${CLAUDE_PLUGIN_ROOT}/..` path that leaves the plugin, which breaks once installed |
| `shell-not-preapproved` | Inline shell that `allowed-tools` doesn't cover, which aborts the skill outside auto mode |
| `missing-script` | A hook or inline shell running a script that isn't there or isn't executable |
| `hook-file-invalid` | A hooks file that isn't valid JSON or lacks the `{"hooks": ...}` shape |
| `hook-unknown-event` | A hook on an event that doesn't exist, such as `preToolUse` |
| `hook-handler-invalid` | A hook without a valid type, its required field, or a usable timeout |
| `hook-type-unsupported` | A hook type the event doesn't run, such as a prompt hook on `SessionStart` |
| `hook-field-ignored` | A `matcher` inside a hook, `once` in a hooks file, a header variable missing from `allowedEnvVars` |
| `hook-never-runs` | A matcher that can't match, such as `bash` or `mcp__memory`, or an `if` that never holds |
| `hook-matcher-ignored` | A matcher on an event that fires every time, such as `Stop` |
| `hook-unscoped-plugin-name` | A plugin hook naming its own agent or MCP server without the plugin's prefix |
| `hook-user-config-in-shell` | `${user_config.…}` in a shell-form command, which Claude Code refuses |
| `hook-async-cannot-block` | An `async` hook written to block or decide, which it can't |
| `hook-exit-1-does-not-block` | A guard that denies with `exit 1`, which lets the action through |
| `hook-output-ignored` | `permissionDecision` outside `hookSpecificOutput`, no `hookEventName`, or `"decision": "approve"` |
| `hook-environment-unavailable` | A hook writing to `/dev/tty`, or using `CLAUDE_ENV_FILE` on an event that doesn't get it |
| `hook-relative-script` | A plugin hook running `scripts/x.sh` relative to the user's project instead of the plugin |

### Warnings: works, but not as Claude Code's documentation recommends

| Rule | Flags |
|---|---|
| `unknown-field` | A field Claude Code's documentation doesn't list, such as `version` |
| `field-without-effect` | `when_to_use` or `paths` on a skill Claude can't invoke, a `timeout` on an async hook |
| `name-format` | A name that isn't lowercase with hyphens |
| `skill-name-mismatch` | A skill named differently from its folder |
| `root-skill-unnamed` | A `SKILL.md` at the plugin's root without a `name` |
| `description-missing` | No description to route on |
| `description-too-long` | Over 1,024 characters, or cut by Claude Code's 1,536-character listing |
| `description-no-trigger` | A description that doesn't say when to use it |
| `description-voice` | A description in the first or second person |
| `description-emphatic` | MUST, CRITICAL or ALWAYS in capitals in a description, which current models over-trigger on |
| `skill-listing-budget` | A plugin whose skill descriptions alone overflow Claude Code's listing |
| `trigger-in-body` | A "When to use" section in a skill's body, which loads only after Claude chose it |
| `body-empty` | Nothing after the frontmatter |
| `body-too-long` | A skill or command body over 500 lines |
| `body-addressed-to-user` | A body that opens with "This command will…" instead of instructions for Claude |
| `emphasis-overused` | Five or more lines of MUST, NEVER, ALWAYS or CRITICAL in capitals |
| `time-sensitive-text` | "Before August 2025, …": instructions that depend on the date |
| `broken-link` | A link or `${CLAUDE_SKILL_DIR}` path to a file that isn't there |
| `nested-reference` | A reference reachable only through another reference |
| `reference-no-toc` | A reference file over 100 lines without a table of contents |
| `non-portable-path` | A path into someone's home directory, or with backslashes |
| `agent-tools-unrestricted` | An agent that can use every tool |
| `agent-memory-grants-write` | `memory` on a read-only agent, which gives it Write and Edit |
| `agent-prompt-voice` | An agent prompt that opens in the first person |
| `agent-no-output-format` | An agent prompt that never says what to return |
| `argument-hint-missing` | A component that takes arguments without an `argument-hint` |
| `side-effects-model-invocable` | A deploy, release or push workflow Claude can start on its own |
| `unquoted-path` | `${CLAUDE_PLUGIN_ROOT}` left unquoted in a shell command |
| `stop-hook-can-loop` | A Stop hook that can block without reading `stop_hook_active` |
| `hook-matcher-dead-alternative` | Part of a matcher that can never match, such as `MultiEdit` |
| `hook-deprecated-decision` | A PreToolUse hook using the deprecated top-level `decision` |
| `hook-agent-experimental` | An agent hook, which Claude Code marks experimental |
| `hook-timeout-in-milliseconds` | A timeout over an hour, likely meant in milliseconds |

## Status and roadmap

groma is young and moves fast; no release is published yet. It started as a security scanner for AI agents, with `scan`, `expose` and a routing benchmark, `bench`, and turned into a linter for Claude Code plugins once real plugins showed that most problems are in how their components are written. The earlier commands are kept on the branches `scan-and-bench` and `expose-agent-port-public`, and `scan`'s security rules live on in `check`.

Next: JSON and SARIF output with a GitHub Action, published binaries, checks for `.mcp.json` servers and settings permissions, and `groma check` on everything Claude Code loads. The full roadmap and the reasons behind it are in [docs/VISION.md](docs/VISION.md).

groma is a safety net, not a guarantee. It checks against what Claude Code's documentation says today, so a field or tool added by a newer release may show up as unknown until groma learns it; and a clean report means no known mistake was found, not that a plugin is safe.
