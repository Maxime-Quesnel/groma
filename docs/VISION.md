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

## `groma check`

Given a plugin, a marketplace, a `.claude` directory or a single file, it finds each component where Claude Code looks for it and checks it. The report lists every component, with its warnings under it, and ends with the red flags:

- **Red flags**: a security risk (hidden Unicode, download-and-run, credential reads, unrestricted shell, arguments in shell, prompt injection, bypassed permissions), or something Claude Code won't load, won't run or ignores.
- **Warnings**: what works but goes against the documentation's advice (descriptions that don't say when to use the component, bodies over 500 lines, broken links, agents with every tool, names that aren't kebab-case).

It exits with 1 on a red flag, so CI can block a release on them while warnings stay advice.

## Design choices

- **One binary, no runtime to install.** Written in Go with the standard library only, distributed as static binaries with checksums.
- **No configuration needed.** It works with none.
- **No account, no cloud, no telemetry.** Nothing leaves the machine.
- **Rules are code and fixtures.** Each rule is an isolated file with an example it flags, a near-miss it passes, its known false positives and the documentation it enforces, so anyone can read why a finding fired.

## Roadmap

**v1: `groma check` for Claude Code plugins**

1. A plugin, marketplace, `.claude` directory or file given as a path, with exit codes a CI job can use.
2. Machine-readable output: JSON, then SARIF for GitHub code scanning.
3. The frontmatter hooks of skills and agents, and `.mcp.json` servers.

**Next, if it proves useful**

4. Everything installed on the machine, from `~/.claude`.
5. Bringing back `groma scan`'s broader security checks and `groma bench`'s routing benchmark, set aside on the branch `scan-and-bench`.

**Not planned:** a web UI, a hosted service, Windows support, automatic fixes without confirmation.

## Honest limits

- groma checks against what Claude Code's documentation says today. A field, tool or event added by a newer release shows up as unknown until groma learns it.
- Prompt injection detection is heuristic. A determined attacker can phrase instructions no rule recognises.
- A clean report means "no known mistake found", not "correct" or "safe".
- Every rule can misfire. Its known false positives are documented next to it.
