# Changelog

## v0.2.0 (2026-09-28)

- A Claude Code plugin, installed with `/plugin marketplace add Maxime-Quesnel/groma` then `/plugin install groma@groma`. After every Edit or Write, its hook checks the component the edited file belongs to and tells Claude what to fix: red flags as something to fix before moving on, warnings as context. `/groma:check [path]` checks a whole plugin on demand.
- `groma hook` reads Claude Code's PostToolUse input and writes the answer the plugin's hook returns. It says nothing about files that aren't part of a component, and never fails an edit.
- A file checked on its own, outside a plugin with a manifest, is read with the directory above its `agents/`, `skills/`, `commands/` or `hooks/` directory, so it's recognized for what it is.
- `unknown-component-reference` no longer reads groma's own `groma:disable` directive as a reference to a plugin named groma.

## v0.1.0 (2026-09-28)

First release: a linter for Claude Code plugins, in the spirit of RuboCop.

- `groma check <path>` checks the plugins, skills, agents, commands and hooks of a plugin, a marketplace, a `.claude` directory or a single file against Claude Code's documentation, with 77 rules. It reports red flags, security risks and what Claude Code won't load, won't run or ignores, and warnings, what works but goes against the documentation's advice. It exits with 1 on a red flag.
- `groma fix [--unsafe] <path>` corrects what 12 of those rules can correct on their own. It shows the diff and asks before writing; safe fixes change nothing where a component works today, and `--unsafe` adds the ones that change what runs.
- `.groma.yml` and `# groma:disable <rule>` comments turn rules off, everywhere, for some paths or for one component.
- Static binaries for Linux and macOS, on amd64 and arm64, with SHA-256 checksums and build provenance attestations.
