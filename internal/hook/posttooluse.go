package hook

import (
	"encoding/json"
	"fmt"
	"io"
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
// and outputs a message telling Claude to run gh watch pr as a background task.
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

	// Check if there's already a watcher running
	if isWatcherRunning() {
		return nil
	}

	// Detect PR for current branch
	prNumber, err := detectPR()
	if err != nil || prNumber == "" {
		return nil
	}

	fmt.Printf("Run `gh watch pr %s --exit --json` as a background task (Bash with run_in_background) to monitor this PR for CI status, reviews, and other changes.\n", prNumber)
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
