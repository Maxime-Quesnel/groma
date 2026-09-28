package rules

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Maxime-Quesnel/groma/internal/claudecode"
	"github.com/Maxime-Quesnel/groma/internal/component"
	"github.com/Maxime-Quesnel/groma/internal/rule"
)

var readsCredentials = Rule{
	Meta: rule.Meta{
		ID:    "reads-credentials",
		Level: rule.RedFlag,
		Title: "Code reaches for the user's credentials",
		Description: "A hook, an inline shell command, or a script one of them or a skill ships, names a file or store that holds the user's secrets: SSH private keys, cloud and cluster credentials, registry and package tokens, the GitHub CLI token, GPG keys, the macOS keychain, browser logins or Claude's own credentials. " +
			"Hooks and scripts run as the user, so nothing stops them from reading those secrets; one network call away, they leave the machine.",
		Remediation: "Remove the access, or explain in the plugin's README why it needs that file and read only what it needs. Don't install someone else's plugin that does this until they explain it.",
		FalsePositives: []string{
			"Code that only checks a credential file's permissions or existence, such as a security audit.",
			"A deny list of sensitive paths inside a guard hook that blocks access to them.",
			"A matching path in a string or docstring the code never uses. Comment lines are skipped.",
			"A project .npmrc or .pypirc that holds registry settings rather than tokens.",
		},
		References: []string{claudecode.HooksDocs},
	},
	Kinds: everything,
	Check: func(c *component.Component, t *component.Tree) []string {
		var evidence []string
		for _, f := range attached(c, t) {
			if !isCode(f) {
				continue
			}
			for _, cl := range findCredentials(f.Content) {
				evidence = append(evidence, in(c, f, fmt.Sprintf("line %d reaches %s: %s", cl.line, cl.text, clip(cl.code))))
			}
		}
		for _, cmd := range commands(c) {
			for _, cl := range findCredentials([]byte(cmd.run)) {
				evidence = append(evidence, fmt.Sprintf("%s reaches %s: %s", cmd.where, cl.text, clip(cl.code)))
			}
		}
		return evidence
	},
}

var credentialStores = []struct {
	what string
	re   *regexp.Regexp
}{
	{"SSH private keys", regexp.MustCompile(`(?m)\.ssh/(?:id_[\w.-]+|identity\b|\*)|\.ssh/?(?:["'\s)\]]|$)`)},
	{"AWS credentials", regexp.MustCompile(`(?m)\.aws/credentials\b|\.aws/?(?:["'\s)\]]|$)`)},
	{"Google Cloud credentials", regexp.MustCompile(`gcloud/(?:credentials\.db|application_default_credentials\.json|access_tokens\.db|legacy_credentials)`)},
	{"Azure tokens", regexp.MustCompile(`\.azure/(?:accessTokens\.json|msal_token_cache)`)},
	{"the Kubernetes config", regexp.MustCompile(`\.kube/config\b`)},
	{"Docker registry logins", regexp.MustCompile(`\.docker/config\.json`)},
	{"stored Git and HTTP passwords", regexp.MustCompile(`\.netrc\b|\.git-credentials\b`)},
	{"package registry tokens", regexp.MustCompile(`\.npmrc\b|\.pypirc\b|\.gem/credentials\b|\.cargo/credentials`)},
	{"the GitHub CLI token", regexp.MustCompile(`gh/hosts\.yml`)},
	{"GPG keys", regexp.MustCompile(`\.gnupg\b`)},
	{"a password store", regexp.MustCompile(`\.password-store\b|Library/Keychains|\bsecurity\s+(?:find|dump)-(?:generic-password|internet-password|keychain)`)},
	{"Claude's credentials", regexp.MustCompile(`\.claude/\.credentials\.json`)},
	{"browser logins and cookies", regexp.MustCompile(`Login Data\b|logins\.json|key4\.db|Cookies\.binarycookies|cookies\.sqlite`)},
}

type credentialLine struct {
	line       int
	text, code string
}

func findCredentials(content []byte) []credentialLine {
	var found []credentialLine
	for i, line := range strings.Split(string(content), "\n") {
		if comment(line) {
			continue
		}
		for _, store := range credentialStores {
			if slices.ContainsFunc(store.re.FindAllString(line, -1), secret) {
				found = append(found, credentialLine{line: i + 1, text: store.what, code: strings.TrimSpace(line)})
				break
			}
		}
	}
	return found
}

// A public key is meant to be shared.
func secret(match string) bool {
	return !strings.HasSuffix(strings.TrimRight(match, "\"') ]\t"), ".pub")
}

func comment(line string) bool {
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"#", "//", "/*", "* ", "--"} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

var codeExtensions = map[string]bool{
	".sh": true, ".bash": true, ".zsh": true, ".fish": true, ".ps1": true,
	".py": true, ".rb": true, ".pl": true, ".php": true, ".lua": true,
	".js": true, ".mjs": true, ".cjs": true, ".ts": true, ".mts": true,
}

// isCode reports whether the file is something the plugin can run: a script
// by extension or shebang, or anything in a bin/ directory.
func isCode(f component.File) bool {
	return codeExtensions[path.Ext(f.Path)] || bytes.HasPrefix(f.Content, []byte("#!")) ||
		strings.HasPrefix(f.Path, "bin/") || strings.Contains(f.Path, "/bin/")
}
