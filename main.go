package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/justincampbell/gh-watch-pr/internal/events"
	"github.com/justincampbell/gh-watch-pr/internal/output"
	"github.com/justincampbell/gh-watch-pr/internal/poller"
	"github.com/justincampbell/gh-watch-pr/internal/pr"
	"github.com/spf13/cobra"
)

func main() {
	var (
		interval time.Duration
		jsonOnly bool
		exitOn   string
	)

	cmd := &cobra.Command{
		Use:   "watch-pr [<number>]",
		Short: "Watch a pull request for state changes",
		Long:  "Monitors a pull request, polling for changes and printing updates as they occur.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := repository.Current()
			if err != nil {
				return fmt.Errorf("detecting repository: %w", err)
			}

			owner := repo.Owner
			repoName := repo.Name

			var number int
			if len(args) == 1 {
				number, err = strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid PR number: %s", args[0])
				}
			} else {
				number, err = detectPRForCurrentBranch()
				if err != nil {
					return err
				}
			}

			writer := &output.Writer{
				JSON:   os.Stdout,
				Stderr: os.Stderr,
				Quiet:  jsonOnly,
			}

			if !jsonOnly {
				fmt.Fprintf(os.Stderr, "Watching PR #%d on %s/%s (polling every %s)\n", number, owner, repoName, interval)
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			var exitOnEvent events.EventType
			if exitOn != "" {
				exitOnEvent = events.EventType(exitOn)
			}

			return poller.Run(ctx, poller.Config{
				Owner:   owner,
				Repo:    repoName,
				Number:  number,
				Fetcher: pr.NewFetcher(),
				Strategy: &poller.FixedStrategy{
					Duration: interval,
				},
				OnEvents: writer.WriteEvents,
				ExitOn:   exitOnEvent,
			})
		},
	}

	cmd.Flags().DurationVar(&interval, "interval", 60*time.Second, "Polling interval")
	cmd.Flags().BoolVar(&jsonOnly, "json", false, "JSON-only output (suppress stderr human-friendly lines)")
	cmd.Flags().StringVar(&exitOn, "exit-on", "", "Exit after a specific event type (e.g. ci-passed)")

	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func detectPRForCurrentBranch() (int, error) {
	stdout, _, err := gh.Exec("pr", "view", "--json", "number", "--jq", ".number")
	if err != nil {
		return 0, fmt.Errorf("detecting PR for current branch: %w\nTip: pass a PR number as an argument, or run from a branch with an open PR", err)
	}
	num, err := strconv.Atoi(strings.TrimSpace(stdout.String()))
	if err != nil {
		return 0, fmt.Errorf("parsing PR number from gh output: %w", err)
	}
	return num, nil
}
