---
name: check
description: Checks a Claude Code plugin, marketplace, skill, agent, command or hooks file with groma, and fixes what it reports. Use when the user asks to check, lint or audit a plugin or one of its components, or before publishing one.
argument-hint: "[path]"
allowed-tools: Bash(groma check *) Bash(groma fix *) Bash(groma --version)
---

Check the path in $ARGUMENTS, or the current directory when it's empty, with groma, a linter that holds Claude Code components against Claude Code's documentation.

1. Run `groma --version`. If groma isn't installed, stop and point the user to https://github.com/Maxime-Quesnel/groma#install.
2. Run `groma check <path>`. It lists every component with its warnings, then the red flags: security risks, and what Claude Code won't load, won't run or ignores. Exit status 1 means there is at least one red flag.
3. Run `groma fix --unsafe <path>` to see the corrections groma can make. From here it can't ask for confirmation, so it only prints the diff and writes nothing. The diff says which fixes are safe and which change what runs.
4. Apply the safe fixes with Edit. Before an unsafe one, such as a rewritten description or a tool granted to an agent, say what it changes and let the user decide.
5. Fix the other red flags by hand, following each finding's Fix line. Leave a warning alone when it contradicts a deliberate convention of the plugin, and tell the user they can turn the rule off in `.groma.yml` or with a `# groma:disable <rule>` comment in the component's frontmatter.
6. Run `groma check <path>` again and report what's left, red flags first.
