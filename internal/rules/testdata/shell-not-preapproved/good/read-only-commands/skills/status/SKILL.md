---
name: status
description: Shows the repository status. Use when the user asks where things stand.
---

- Branch: !`git branch --show-current`
- Log: !`git log --oneline -5`
- Files: !`ls -la`
