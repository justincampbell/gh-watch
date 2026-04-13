package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type toolInput struct {
	Command string `json:"command"`
}

type hookInput struct {
	ToolInput toolInput `json:"tool_input"`
}

var gitPushRe = regexp.MustCompile(`git\s+push`)

// PostToolUse handles the PostToolUse hook event.
// It checks if the Bash command was a git push, detects the PR,
// and starts gh watch pr in the background.
func PostToolUse(stdin io.Reader) error {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil // fail silently
	}

	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil
	}

	if !gitPushRe.MatchString(input.ToolInput.Command) {
		return nil
	}

	// Check if gh watch is available
	if _, err := exec.LookPath("gh"); err != nil {
		return nil
	}

	// Check if there's already a watcher running
	if isWatcherRunning() {
		return nil
	}

	// Detect PR for current branch
	prNumber, err := detectPR()
	if err != nil || prNumber == "" {
		return nil
	}

	// Start watcher in background
	cmd := exec.Command("gh", "watch", "pr", prNumber, "--json")
	logFile := fmt.Sprintf("/tmp/gh-watch-pr-%s.log", prNumber)
	f, err := os.Create(logFile)
	if err != nil {
		return nil
	}
	cmd.Stdout = f
	cmd.Stderr = f

	if err := cmd.Start(); err != nil {
		f.Close()
		return nil
	}

	fmt.Fprintf(os.Stdout, "Started watching PR #%s in the background (pid %d)\n", prNumber, cmd.Process.Pid)
	return nil
}

func isWatcherRunning() bool {
	out, err := exec.Command("pgrep", "-f", "gh-watch.*watch.*pr").Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func detectPR() (string, error) {
	out, err := exec.Command("gh", "pr", "view", "--json", "number", "--jq", ".number").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
