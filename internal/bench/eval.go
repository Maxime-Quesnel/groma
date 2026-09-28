package bench

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type Options struct {
	Runs        int
	Model       string
	JudgeModel  string
	MaxCostUSD  float64
	Concurrency int
	// Scaffold runs each case's scaffold script, as the user.
	Scaffold   bool
	AllowTools []string
}

// Eval runs claude plugin eval on the prepared plugin copy, through the
// user's Claude Code login and plan, and writes its JSON result to
// resultPath. The report is never published, and no no-plugin baseline
// runs: routing needs the plugin loaded.
func Eval(ctx context.Context, pluginCopy, resultPath string, o Options, stdout, stderr io.Writer) error {
	args := evalArgs(pluginCopy, resultPath, o)
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	// Exit 2 means the cost ceiling or an interruption cut the run short;
	// the partial result is still written and still worth reading.
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 2 {
		return nil
	}
	return err
}

// CheckClaude makes sure the claude found on PATH can run plugin eval with
// the flags groma passes. Older Claude Code releases lack it, and an older
// second install earlier on PATH, from Homebrew for instance, is an easy way
// to end up running one.
func CheckClaude(ctx context.Context) error {
	path, err := exec.LookPath("claude")
	if err != nil {
		return errors.New("claude not found on PATH: install Claude Code first")
	}
	help, _ := exec.CommandContext(ctx, path, "plugin", "eval", "--help").CombinedOutput()
	if bytes.Contains(help, []byte("--ablation")) {
		return nil
	}
	version, _ := exec.CommandContext(ctx, path, "--version").Output()
	release := "an older Claude Code"
	if fields := strings.Fields(string(version)); len(fields) > 0 {
		release = "Claude Code " + fields[0]
	}
	return fmt.Errorf("%s is %s, which has no plugin eval: update it, or put a newer claude first on PATH", path, release)
}

func evalArgs(pluginCopy, resultPath string, o Options) []string {
	args := []string{"plugin", "eval", pluginCopy,
		"--ablation", "none", "--no-publish", "--trust-plugin", "--threshold", "0",
		"--runs", strconv.Itoa(o.Runs),
		"--concurrency", strconv.Itoa(max(o.Concurrency, 1)),
		"--json", resultPath,
	}
	// Without a model, runs use the one the user chose in Claude Code.
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	if o.JudgeModel != "" {
		args = append(args, "--judge-model", o.JudgeModel)
	}
	if o.MaxCostUSD > 0 {
		args = append(args, "--max-cost-usd", strconv.FormatFloat(o.MaxCostUSD, 'f', -1, 64))
	}
	if o.Scaffold {
		args = append(args, "--scaffold")
	}
	// --allow-tools takes a list, so it goes last.
	if len(o.AllowTools) > 0 {
		args = append(append(args, "--allow-tools"), o.AllowTools...)
	}
	return args
}
