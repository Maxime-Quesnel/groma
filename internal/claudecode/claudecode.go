// Package claudecode holds what Claude Code accepts in skills, subagents,
// commands and hooks, as its documentation states it. Rules check components
// against it; when Claude Code adds a field, a tool or an event, this is the
// one file to update.
package claudecode

import (
	"regexp"
	"slices"
	"strings"
)

// Docs are the pages each list below comes from.
const (
	SkillsDocs            = "https://code.claude.com/docs/en/skills"
	SubagentsDocs         = "https://code.claude.com/docs/en/sub-agents"
	HooksDocs             = "https://code.claude.com/docs/en/hooks"
	HooksGuide            = "https://code.claude.com/docs/en/hooks-guide"
	ToolsDocs             = "https://code.claude.com/docs/en/tools-reference"
	PermissionsDocs       = "https://code.claude.com/docs/en/permissions"
	PluginComponentsDocs  = "https://code.claude.com/docs/en/plugins/components"
	PluginManifestDocs    = "https://code.claude.com/docs/en/plugins/manifest-reference"
	PluginTroubleshooting = "https://code.claude.com/docs/en/plugins/troubleshooting"
	BestPractices         = "https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices"
	PromptingDocs         = "https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices"
	PluginDevAgents       = "https://github.com/anthropics/claude-code/tree/main/plugins/plugin-dev/skills/agent-development"
	PluginDevCommands     = "https://github.com/anthropics/claude-code/tree/main/plugins/plugin-dev/skills/command-development"
	SkillCreator          = "https://github.com/anthropics/skills/tree/main/skills/skill-creator"
)

var SkillFields = []string{
	"name", "description", "when_to_use", "argument-hint", "arguments",
	"disable-model-invocation", "user-invocable", "allowed-tools", "disallowed-tools",
	"model", "effort", "context", "agent", "background", "hooks", "paths", "shell",
	"metadata", "license", "compatibility",
}

// CommandFields are a skill's, less name and paths: a command takes its name
// from its file.
var CommandFields = slices.DeleteFunc(slices.Clone(SkillFields), func(f string) bool {
	return f == "name" || f == "paths"
})

var AgentFields = []string{
	"name", "description", "tools", "disallowedTools", "model", "permissionMode",
	"maxTurns", "skills", "mcpServers", "hooks", "memory", "background",
	"omitClaudeMd", "effort", "isolation", "color", "initialPrompt", "experimental",
}

// PluginIgnoredAgentFields are agent fields Claude Code ignores when the agent
// ships in a plugin.
var PluginIgnoredAgentFields = []string{"permissionMode", "mcpServers", "hooks", "initialPrompt"}

var Tools = []string{
	"Agent", "Artifact", "AskUserQuestion", "Bash", "CronCreate", "CronDelete", "CronList",
	"Edit", "EndConversation", "EnterPlanMode", "EnterWorktree", "ExitPlanMode", "ExitWorktree",
	"Glob", "Grep", "ListAgents", "ListMcpResourcesTool", "LSP", "Monitor", "NotebookEdit",
	"PowerShell", "PushNotification", "Read", "ReadMcpResourceTool", "RemoteTrigger",
	"ReportFindings", "ScheduleWakeup", "SendFeedback", "SendMessage", "SendUserFile",
	"ShareOnboardingGuide", "Skill", "SubagentHandback", "TaskCreate", "TaskGet", "TaskList",
	"TaskOutput", "TaskStop", "TaskUpdate", "TodoWrite", "ToolSearch", "WaitForMcpServers",
	"WebFetch", "WebSearch", "Workflow", "Write",
	// Task is the Agent tool's name before Claude Code 2.1.63, still accepted.
	"Task",
}

// FormerTools are tool names Claude Code no longer has, with what replaced
// them.
var FormerTools = map[string]string{
	"MultiEdit":    "Edit",
	"BashOutput":   "Read",
	"KillShell":    "TaskStop",
	"KillBash":     "TaskStop",
	"NotebookRead": "Read",
	"LS":           "Glob",
	"SlashCommand": "Skill",
}

// UnavailableToSubagents are removed from a subagent's tools even when its
// tools field lists them. Agent is only removed at the nesting limit, and
// ExitPlanMode is kept in plan mode, so neither is listed.
var UnavailableToSubagents = []string{
	"AskUserQuestion", "EndConversation", "EnterPlanMode", "ScheduleWakeup", "WaitForMcpServers", "Workflow",
}

// Tool returns the tool a permission entry such as Bash(git add *) or
// mcp__github__create_issue names.
func Tool(entry string) string {
	name, _, _ := strings.Cut(strings.TrimSpace(entry), "(")
	return strings.TrimSpace(name)
}

// KnownTool reports whether Claude Code has a tool of that name. MCP tools,
// named mcp__<server>__<tool>, can't be checked without the server.
func KnownTool(name string) bool {
	return slices.Contains(Tools, name) || strings.HasPrefix(name, "mcp__")
}

var HookEvents = []string{
	"SessionStart", "Setup", "UserPromptSubmit", "UserPromptExpansion", "PreToolUse",
	"PermissionRequest", "PermissionDenied", "PostToolUse", "PostToolUseFailure", "PostToolBatch",
	"Notification", "MessageDisplay", "SubagentStart", "SubagentStop", "TaskCreated",
	"TaskCompleted", "Stop", "StopFailure", "TeammateIdle", "InstructionsLoaded", "ConfigChange",
	"CwdChanged", "DirectoryAdded", "FileChanged", "WorktreeCreate", "WorktreeRemove",
	"PreCompact", "PostCompact", "PreModelSwitch", "PostModelSwitch", "Elicitation",
	"ElicitationResult", "SessionEnd",
}

// EventsWithoutMatcher fire on every occurrence; Claude Code ignores a
// matcher on them.
var EventsWithoutMatcher = []string{
	"CwdChanged", "UserPromptSubmit", "PostToolBatch", "Stop", "TeammateIdle",
	"TaskCreated", "TaskCompleted", "WorktreeCreate", "WorktreeRemove", "MessageDisplay",
}

// ToolEvents match tool names, and are the only events where a hook's if
// condition is evaluated.
var ToolEvents = []string{"PreToolUse", "PostToolUse", "PostToolUseFailure", "PermissionRequest", "PermissionDenied"}

// HandlerFields maps each hook type to the fields it requires and the ones
// it accepts.
var HandlerFields = map[string]struct{ Required, Optional []string }{
	"command":  {[]string{"command"}, []string{"args", "async", "asyncRewake", "shell"}},
	"http":     {[]string{"url"}, []string{"headers", "allowedEnvVars"}},
	"mcp_tool": {[]string{"server", "tool"}, []string{"input"}},
	"prompt":   {[]string{"prompt"}, []string{"model", "continueOnBlock"}},
	"agent":    {[]string{"prompt"}, []string{"model"}},
}

// hookTypes lists the hook types each event runs; an event missing from it
// runs all five.
var hookTypes = map[string][]string{
	"PermissionRequest": {"command", "http", "mcp_tool", "prompt"},
	"SessionStart":      {"command", "mcp_tool"},
	"Setup":             {"command", "mcp_tool"},
}

func init() {
	for _, event := range []string{
		"ConfigChange", "CwdChanged", "DirectoryAdded", "Elicitation", "ElicitationResult", "FileChanged",
		"InstructionsLoaded", "MessageDisplay", "Notification", "PostCompact", "PostModelSwitch", "PreCompact",
		"PreModelSwitch", "SessionEnd", "StopFailure", "SubagentStart", "WorktreeCreate", "WorktreeRemove",
	} {
		hookTypes[event] = []string{"command", "http", "mcp_tool"}
	}
}

// HookTypes returns the hook types an event runs.
func HookTypes(event string) []string {
	if types, ok := hookTypes[event]; ok {
		return types
	}
	return []string{"command", "http", "mcp_tool", "prompt", "agent"}
}

// MatcherValues are the values an event with a fixed set of matcher values
// matches against.
var MatcherValues = map[string][]string{
	"SessionStart":   {"startup", "resume", "clear", "compact", "fork"},
	"Setup":          {"init", "maintenance"},
	"SessionEnd":     {"clear", "resume", "logout", "prompt_input_exit", "other"},
	"PreCompact":     {"manual", "auto"},
	"PostCompact":    {"manual", "auto"},
	"ConfigChange":   {"user_settings", "project_settings", "local_settings", "policy_settings", "skills"},
	"DirectoryAdded": {"slash_command", "register_repo_root"},
	"Notification": {"permission_prompt", "idle_prompt", "auth_success", "elicitation_dialog", "elicitation_url_dialog",
		"elicitation_complete", "elicitation_response", "agent_needs_input", "agent_completed",
		"quota_auto_resume_fired", "quota_auto_resume_stale", "quota_auto_resume_disabled"},
	"InstructionsLoaded": {"session_start", "nested_traversal", "path_glob_match", "include", "compact"},
	"StopFailure": {"rate_limit", "overloaded", "authentication_failed", "oauth_org_not_allowed", "account_on_hold",
		"billing_error", "invalid_request", "model_not_found", "server_error", "max_output_tokens",
		"cloud_credential_error", "unknown"},
}

var exactMatcher = regexp.MustCompile(`^[A-Za-z0-9_, |-]+$`)

// MatcherAlternatives returns the exact values a matcher lists, separated by
// | or commas. A matcher with any other character is a regular expression,
// and "" or * matches everything; neither has alternatives to check.
func MatcherAlternatives(matcher string) ([]string, bool) {
	if !exactMatcher.MatchString(matcher) {
		return nil, false
	}
	var alternatives []string
	for _, a := range strings.FieldsFunc(matcher, func(r rune) bool { return r == '|' || r == ',' }) {
		if a = strings.TrimSpace(a); a != "" {
			alternatives = append(alternatives, a)
		}
	}
	return alternatives, len(alternatives) > 0
}

// EnvFileEvents are the only events whose hooks get CLAUDE_ENV_FILE.
var EnvFileEvents = []string{"SessionStart", "Setup", "CwdChanged", "FileChanged"}

// BlockingEvents are events a hook can block with exit 2.
var BlockingEvents = []string{
	"PreToolUse", "UserPromptSubmit", "UserPromptExpansion", "Stop", "SubagentStop",
	"TeammateIdle", "TaskCreated", "TaskCompleted", "PreModelSwitch",
}

// CommonHandlerFields apply to every hook type.
var CommonHandlerFields = []string{"type", "if", "timeout", "statusMessage", "once"}

var (
	Effort          = []string{"low", "medium", "high", "xhigh", "max"}
	Colors          = []string{"red", "blue", "green", "yellow", "purple", "orange", "pink", "cyan"}
	PermissionModes = []string{"default", "acceptEdits", "auto", "dontAsk", "bypassPermissions", "plan", "manual"}
	Memory          = []string{"user", "project", "local"}
	Isolation       = []string{"worktree"}
	Context         = []string{"fork"}
	Shells          = []string{"bash", "powershell"}
)

var (
	modelAliases = []string{"default", "best", "sonnet", "opus", "haiku", "fable", "opusplan", "inherit"}
	modelID      = regexp.MustCompile(`^claude-[a-z0-9.-]+$`)
)

// Model reports whether Claude Code accepts s as a model: an alias, as /model
// takes it, or a full model ID, either with the [1m] suffix for the
// million-token context.
func Model(s string) bool {
	s = strings.TrimSuffix(strings.ToLower(s), "[1m]")
	return slices.Contains(modelAliases, s) || modelID.MatchString(s)
}

// Bool reports whether s is one of the boolean spellings Claude Code accepts.
func Bool(s string) bool {
	return slices.Contains([]string{"true", "false", "yes", "no", "on", "off", "1", "0"}, strings.ToLower(s))
}
