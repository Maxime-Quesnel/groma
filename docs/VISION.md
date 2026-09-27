# Vision

Before a Roman legion set up camp, a surveyor planted a groma and traced the perimeter. Nothing was built before the boundaries were drawn. groma does the same for AI agents: it traces what can get in, what gets out, and what the agent can touch.

## The problem

AI agents don't just answer questions. They run shell commands, read and write files, and call APIs with their owner's keys, and many of them run around the clock.

They are deployed casually. A typical setup is a cheap VPS, the agent's gateway bound to `0.0.0.0` so a phone app can reach it, or a tunnel opened without an access policy. The agent often runs as root, sometimes in a container with the Docker socket mounted.

They are extended with community code. Skills, plugins, hooks and MCP servers are installed from a marketplace or a GitHub link and rarely read. A skill is instructions plus scripts running with the agent's privileges: one hidden line, sometimes invisible to the human eye, is enough to tell the agent to send `~/.ssh` somewhere.

The real attack surface is the combination of three things: what the agent is allowed to do, what the host exposes, and what every installed extension intends. Nobody looks at the three together.

## Who it's for

- **Authors of Claude Code plugins** who want to check their skills, agents, hooks and MCP servers before publishing them, on their machine and in CI. This is the first audience.
- **People installing plugins** from a marketplace or a GitHub link, who want to know what they are about to trust.
- **Individual developers and small teams** who self-host an always-on agent on a VPS, a homelab or their own laptop.

Not a first target: enterprises that need fleet management, compliance reports or SOC integration.

## Positioning

Agent platforms are adding their own protections: permission systems, sandboxes, marketplaces that screen submissions. They are necessary, and groma doesn't replace them. It covers what they leave out.

- **Each platform guards its own walls.** A Claude Code permission rule says nothing about an OpenClaw skill on the same machine. groma audits every agent with the same rules and the same report.
- **They look at the agent, not the deployment.** An agent can't tell that its dashboard is bound to a public interface, that the tunnel in front of it has no authentication, or that it runs as root next to the Docker socket. `groma expose` can.
- **groma is neutral.** It has no marketplace to protect and no agent to promote. Its rules are public, reviewed in the open, and give everyone the same findings.

### Next to `claude plugin eval`

`claude plugin eval` runs a plugin against test cases in a sandboxed session and grades its behaviour: did the skill trigger, call the right tools, produce the right file. It answers "does the plugin do its job?". It doesn't read the plugin's files, doesn't look for secrets or dangerous patterns, and doesn't inspect hooks or MCP servers, which run outside its sandbox. Nor does it question the tools a skill pre-approves.

groma is the static counterpart. It reads everything the plugin ships and runs none of it, and answers "what else could this plugin do?". A plugin author runs both in CI.

groma is not a sandbox and not a runtime monitor: it inspects a state, it doesn't watch execution. It is not an antivirus with signatures pulled from a cloud. And it is not a guarantee.

## The three commands

### `groma scan`: what gets in

Inspects a plugin, a marketplace or a skill directory, or everything installed for an agent. For Claude Code that means every place where a plugin can run code or gain permissions without a per-action approval:

- hooks (`hooks/hooks.json`, `plugin.json`, skill and agent frontmatter) and the scripts they call, some of which run on every session start or tool call;
- MCP and LSP servers (`.mcp.json`, `.lsp.json`), which start when the plugin is enabled and run as the user;
- executables in `bin/`, added to `PATH`;
- skills, agents and commands: their instructions, the tools they pre-approve (`allowed-tools`, `tools`, `permissionMode`) and the shell commands they inline;
- the manifests (`plugin.json`, `marketplace.json`) and every script the plugin ships.

It looks for:

- hidden instructions and prompt injection aimed at the agent;
- invisible or deceptive Unicode: zero-width characters, tag characters, bidirectional overrides;
- `curl | sh` and other download-and-execute patterns;
- encoded payloads (base64, hex) and what they decode to;
- reads of `~/.ssh`, `.env` files and cloud credentials;
- suspicious network calls;
- hardcoded secrets;
- permissions far broader than the extension needs.

`scan` is static analysis. It never runs the code it inspects.

### `groma expose`: what gets out

Checks what a machine exposes, locally or on a remote host over SSH (`--host user@ip`). It uses the system `ssh` client and runs read-only commands only; nothing is uploaded to the host. It looks for:

- agent ports and web interfaces reachable without authentication;
- tunnels without an access policy;
- secrets readable by every user;
- an agent running as root;
- a Docker socket mounted into an agent's container;
- SSH and firewall hygiene: password login, root login, no default-deny policy.

Each finding comes with its evidence (the listening socket, the file mode, the config line), with any secret masked.

### `groma fix`: tightening what it can touch

Proposes safe settings for the findings of `scan` and `expose`: bind to localhost, tighten permissions on secret files, disable flagged skills. It shows a diff for every change and asks before applying it. `scan` and `expose` never call it.

## Design choices

- **One binary, no runtime to install.** Written in Go, distributed as static binaries with checksums.
- **Plain-text config, strong defaults.** It works with no configuration at all.
- **No account, no cloud, no telemetry.** Nothing leaves the machine, except commands sent to the host being audited.
- **Rules are code and fixtures.** Each rule is an isolated file with a dangerous example, a safe example and its known false positives, so anyone can read why a finding fired.

## Roadmap

**v1: `groma scan` for Claude Code plugins**

1. A plugin, marketplace or skill directory given as a path, with exit codes a CI job can use.
2. Everything installed: `~/.claude/plugins`, user and project skills, agents and hooks.
3. Machine-readable output (JSON, then SARIF for GitHub code scanning).

**Next**

4. `groma expose` for always-on agents on a Linux VPS, starting with OpenClaw.
5. `groma fix` for the findings of both commands.
6. More adapters: skills of OpenClaw and Hermes Agent for `scan`, Hermes Agent for `expose`, then Codex CLI and OpenCode.

**Ideas, not commitments:** host checks on macOS.

**Not planned:** a web UI, a hosted service, Windows support, fixes applied without confirmation.

## Honest limits

- Prompt injection detection is heuristic. A determined attacker can phrase instructions no rule recognises. groma raises the cost of the easy attacks, and says so.
- A clean report means "no known pattern found", not "safe".
- `scan` sees what is on disk. Code an extension downloads at runtime is invisible to it.
- Every rule can misfire. Its known false positives are documented next to it.
