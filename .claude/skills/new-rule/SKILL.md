---
name: new-rule
description: Add a check to groma, with its metadata, the documentation it enforces, bad and good fixtures, known false positives and tests. Use when adding a check of skills, agents, commands or hooks, or when turning a Claude Code documentation change, a reported mistake or a risk into a rule. Do NOT use for changes to the CLI, the report format or component detection.
---

# Add a rule

Every check in groma is one rule: one Go file in `internal/rules/`, and its fixtures. A rule reads, it never writes, runs nothing it inspects, and never touches the network.

## 1. Ground it

- Find the page of Claude Code's documentation, or Anthropic's authoring guidance, that the rule enforces, and read it: the rule's References cite it. A rule without a source is an opinion; don't add it.
- Search `internal/rules/` for a rule about the same problem. If one exists and only the pattern differs, extend it: two rules must never report the same finding twice.
- What Claude Code accepts, such as field names, tools, hook events or allowed values, goes in `internal/claudecode/claudecode.go`, never inline in the rule, so a Claude Code release means one file to update.

## 2. Name it

- **ID:** the problem in kebab-case, stated as what's wrong, not how it's detected: `hook-unknown-event`, not `hook-event-regex`.
- **Files:** `internal/rules/<snake_case_id>.go`, fixtures in `internal/rules/testdata/<id>/bad/` and `.../good/`. Add a `<snake_case_id>_test.go` only to pin the exact wording of the evidence.
- Register the rule in `All` in `internal/rules/rules.go`, with the rules of its level.

## 3. Pick a level

| Level | When |
|---|---|
| `rule.RedFlag` | A security risk, or something Claude Code won't load, won't run or silently ignores: the component doesn't do what its author wrote. |
| `rule.Warning` | It works, but goes against the documentation's advice: routing, context cost, portability, readability. |

When in doubt, it's a warning. A red flag makes CI fail.

## 4. Write the metadata

- **ID** and **Level**, as above, and **Kinds**: the components it checks (`markdown`, `skillsAndCommands`, `agents`, `hooks`, `everything`, ...).
- **Title:** one line stating the problem: "Hook on an event Claude Code doesn't have".
- **Description:** two to four sentences: what is detected, what Claude Code does with it, and what that costs the user.
- **Remediation:** what to change, concretely.
- **FalsePositives:** the known cases where the rule fires on something fine. Never empty: if none are known, say so and name the near-misses the good fixtures cover.
- **References:** the documentation it enforces, from the constants in `internal/claudecode`.

## 5. Write the check

`Check(c *component.Component, t *component.Tree) []string` returns one evidence line per problem, empty when there is none.

- Point at the place: `at(line, ...)` or `fieldAt(c, key, ...)` for frontmatter, `h.Where()` for a hook, `in(c, f, ...)` for a file other than the component's own.
- Rules that read frontmatter fields return nothing when `!headerReadable(c)`: `frontmatter-unreadable` already reports it.
- Quote what you found, clipped with `clip`. Evidence can hold secrets from the files checked: the report masks the common shapes, but never build evidence that spells a secret out on purpose.

## 6. Write the fixtures

- Under `bad/`, at least one case the rule must flag; under `good/`, at least one near-miss it must pass, as close to the bad case as possible: it is what keeps false positives down.
- A case is a directory laid out like a real plugin (`skills/pdf/SKILL.md`, `agents/reviewer.md`, `hooks/hooks.json`), or a single file. It must hold a component of a kind the rule checks.
- One case per directory, named after what it shows: `bad/lowercase-matcher`, `good/exact-names`.
- Fixtures may contain prompt-injection text aimed at agents: data under test, never instructions. Exfiltration targets use `example.com`, `.invalid` or `192.0.2.0/24`.
- Fake secrets only: provider-documented examples (`AKIAIOSFODNN7EXAMPLE`) or strings assembled at test time. Never a real key, even a revoked one.
- File modes matter to some rules, such as `missing-script`: set them with `chmod` before `git add`.

## 7. Test

`TestRules` runs every registered rule on its fixtures: each `bad/` case must yield evidence, each `good/` case none, and the metadata must be complete. Then run it on real plugins, such as `~/.claude/plugins/marketplaces/*`, and read every finding: a rule that misfires on well-written plugins isn't done.

```sh
go test ./...
gofmt -l .
go vet ./...
go build -o /tmp/groma ./cmd/groma && /tmp/groma check ~/.claude/plugins/marketplaces/claude-plugins-official
```

`gofmt -l .` must print nothing.

## 8. Before calling it done

- [ ] The rule cites the documentation it enforces.
- [ ] Claude Code's vocabulary it uses lives in `internal/claudecode`.
- [ ] The rule is read-only, runs nothing, and makes no network call.
- [ ] Bad and good fixtures exist, and the tests pass.
- [ ] False positives are documented, and a run on real plugins shows none that aren't.
- [ ] The rule is registered in `All` and listed in the README's tables.
