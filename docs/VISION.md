# Vision

Before a Roman legion set up camp, a surveyor planted a groma and traced straight lines. Nothing was built until they were right. groma does the same for the pieces of a Claude Code plugin: it checks that each skill, agent, command and hook is written the way Claude Code reads it, before anyone installs it.

## The problem

A Claude Code plugin is a folder of Markdown and JSON: skills that Claude loads when a task calls for them, subagents it delegates to, slash commands, and hooks that run on their own around every tool call. Authors write them by hand, from documentation spread over half a dozen pages that changes with every release.

Claude Code forgives nearly every mistake silently:

- a misspelled field (`allowed_tools`), or a real field in the wrong kind of component (`tools` in a skill), is ignored, so the setting never applies;
- a tool that was renamed or removed (`MultiEdit`, `LS`) is dropped from the list;
- a hook on a misspelled event, or with a lowercase matcher, never fires, including a guard meant to block a dangerous command;
- an agent named `shop:reviewer`, or with no description, is skipped, and only the debug log says so;
- `permissionMode` and `hooks` on a plugin agent don't apply, so an agent the author restricted isn't;
- inline shell that `allowed-tools` doesn't cover makes the skill abort for every user outside auto mode.

Nothing in the session shows it. The plugin looks fine, and part of it doesn't work, or does more than its author meant. And plugins also carry the security risks any code does: a hook that pipes a download into a shell, a skill that pre-approves every command, arguments substituted straight into a shell line.

## Who it's for

- **Authors of Claude Code plugins** who want to check their skills, agents, commands and hooks before publishing them, on their machine and in CI. This is the audience.
- **People installing plugins** who want to know what they are about to trust.

## Positioning

- **`claude plugin validate`** checks the manifest, that frontmatter and hooks files parse, and hook event names. It doesn't check frontmatter field names, tools, values, matchers, scripts, or anything about how a component is written.
- **`claude plugin eval`** runs a plugin against test cases and grades what it does. It answers "does the plugin do its job?". groma is the static counterpart: it reads everything the plugin ships, runs none of it, and answers "is it written the way Claude Code reads it, and what else could it do?". A plugin author runs both in CI.
- **Community linters** (agnix, claudelint, skillsaw and many more) cover parts of this. groma's line is to combine the writing checks with the security ones in one report, to ground every rule in Claude Code's documentation, to keep Claude Code's vocabulary in one file that follows its releases, and to hold its own principles: no network call, nothing executed, secrets always masked, no dependency.

groma is not a sandbox, not a runtime monitor, and not a guarantee.

## What groma does

A linter for Claude Code plugins, in the spirit of RuboCop for Ruby: rules that encode the documented way of writing each component, a report, and fixes for what can be fixed mechanically.

- **`groma check <path>`** finds each component of a plugin, a marketplace, a `.claude` directory or a single file where Claude Code looks for it, including the plugin itself through its manifest, and checks it. The report lists every component with its warnings, and ends with the red flags:
  - **Red flags**: a security risk (hidden Unicode, download-and-run, credential reads, unrestricted shell, arguments in shell, prompt injection, bypassed permissions), or something Claude Code won't load, won't run or ignores.
  - **Warnings**: what works but goes against the documentation's advice (descriptions that don't say when to use the component, bodies over 500 lines, broken links, agents with every tool, emphatic wording).

  It exits with 1 on a red flag, so CI can block a release on them while warnings stay advice.
- **`groma fix <path>`** corrects what a rule can correct on its own, shows the diff and asks before writing. Safe fixes change nothing where a component works today; `--unsafe` adds the ones that change what runs, when, or with which tools.
- **The groma plugin for Claude Code** runs the checks on each component right after Claude edits it, and tells Claude what to fix, so components come out right as they're written.
- **`.groma.yml`** turns rules off, everywhere or for some paths, and a `# groma:disable <rule>` comment does it for one component, for a team whose conventions differ from a rule on purpose.

## Design choices

- **One binary, no runtime to install.** Written in Go with the standard library only, distributed as static binaries with checksums.
- **No configuration needed.** It works with none; `.groma.yml` only turns rules off.
- **No account, no cloud, no telemetry.** Nothing leaves the machine.
- **Rules are code and fixtures.** Each rule is an isolated file with an example it flags, a near-miss it passes, its known false positives and the documentation it enforces, so anyone can read why a finding fired.
- **Fixes never write unseen.** `groma fix` shows every change and waits for a yes; without a terminal to ask in, it writes nothing.

## How groma got here

groma started as a security scanner for self-hosted agents: `groma expose` for what an always-on agent exposes on a VPS, then `groma scan` for what a plugin could do, then `groma bench`, which measured how precisely Claude routes work to each agent of a plugin by running its eval suite. Using them on real plugins showed that most of what goes wrong is how the components are written, silently, and that no tool held them against Claude Code's own documentation. groma became that tool: `check`, then `fix`.

What was set aside is kept, not deleted: `scan` and `bench` on the branch `scan-and-bench`, the first `expose` slice on `expose-agent-port-public`. `scan`'s four security rules live on in `check`.

## Roadmap

**Done**

- `groma check`: 77 rules over plugins, skills, agents, commands and hooks.
- `groma fix`, safe and `--unsafe`, with diff and confirmation.
- `.groma.yml` and `groma:disable` comments.
- v0.1.0: static binaries for Linux and macOS, with checksums and build provenance.
- v0.2.0: a Claude Code plugin whose hook checks each component as Claude writes it, and a `/groma:check` skill.

**Next**

1. Machine-readable output, JSON then SARIF, and a GitHub Action, so findings show on pull requests.
2. A Homebrew tap, now that releases publish static binaries with checksums and provenance.
3. More of what plugins ship: the hooks of skills' and agents' frontmatter, `.mcp.json` servers, the permissions of settings files.
4. `groma check` with no path, for everything Claude Code loads on the machine.
5. A scheduled job that compares groma's list of Claude Code fields, tools and events with the documentation, so the rules follow each release.

**Later, if it proves useful:** bringing back `bench` to measure what a rule claims, such as whether plain descriptions route as well as emphatic ones.

**Not planned:** a web UI, a hosted service, Windows support, fixes applied without confirmation.

## Honest limits

- groma checks against what Claude Code's documentation says today. A field, tool or event added by a newer release shows up as unknown until groma learns it.
- Prompt injection detection is heuristic. A determined attacker can phrase instructions no rule recognises.
- A clean report means "no known mistake found", not "correct" or "safe".
- Every rule can misfire. Its known false positives are documented next to it.
