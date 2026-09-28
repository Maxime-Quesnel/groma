# Changelog

## v0.1.0 (2026-09-28)

First release: a linter for Claude Code plugins, in the spirit of RuboCop.

- `groma check <path>` checks the plugins, skills, agents, commands and hooks of a plugin, a marketplace, a `.claude` directory or a single file against Claude Code's documentation, with 77 rules. It reports red flags, security risks and what Claude Code won't load, won't run or ignores, and warnings, what works but goes against the documentation's advice. It exits with 1 on a red flag.
- `groma fix [--unsafe] <path>` corrects what 12 of those rules can correct on their own. It shows the diff and asks before writing; safe fixes change nothing where a component works today, and `--unsafe` adds the ones that change what runs.
- `.groma.yml` and `# groma:disable <rule>` comments turn rules off, everywhere, for some paths or for one component.
- Static binaries for Linux and macOS, on amd64 and arm64, with SHA-256 checksums and build provenance attestations.
