package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/justincampbell/gh-watch/internal/commit"
	"github.com/justincampbell/gh-watch/internal/events"
	"github.com/justincampbell/gh-watch/internal/output"
	"github.com/justincampbell/gh-watch/internal/poller"
	"github.com/justincampbell/gh-watch/internal/pr"
	"github.com/spf13/cobra"
)

func main() {
	var (
		interval time.Duration
		exit     bool
		exitOn   []string
	)

	rootCmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch GitHub resources for state changes",
	}

	rootCmd.PersistentFlags().DurationVar(&interval, "interval", 60*time.Second, "Polling interval")
	rootCmd.PersistentFlags().BoolVar(&exit, "exit", false, "Exit after any state change (shorthand for --exit-on any)")
	rootCmd.PersistentFlags().StringSliceVar(&exitOn, "exit-on", nil, `Exit after specific event types (e.g. --exit-on ci-passed,ci-failed)`)
	rootCmd.SilenceUsage = true

	prCmd := &cobra.Command{
		Use:   "pr [<number>]",
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

			writer := &output.Writer{JSON: os.Stdout}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			fetcher := pr.NewFetcher()

			return poller.Run(ctx, poller.Config[pr.State]{
				Fetch: func() (*pr.State, error) {
					return fetcher.Fetch(owner, repoName, number)
				},
				Diff: func(old, new *pr.State) []events.Event {
					return events.Diff(old, new)
				},
				IsTerminal: func(e events.Event) bool {
					return e.Event == events.PRMerged || e.Event == events.PRClosed
				},
				Strategy: &poller.FixedStrategy{
					Duration: interval,
				},
				OnEvents: writer.WriteEvents,
				ExitOn:   buildExitOnEvents(exit, exitOn),
			})
		},
	}

	commitCmd := &cobra.Command{
		Use:   "commit <sha-or-url>",
		Short: "Watch a commit for CI status changes",
		Long:  "Monitors a commit's CI checks, polling for changes and printing updates as they occur.\nAccepts a SHA or a GitHub commit URL (https://github.com/owner/repo/commit/sha).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner, repoName, sha string

			parsed, err := parseCommitURL(args[0])
			if err == nil {
				owner = parsed.owner
				repoName = parsed.repo
				sha = parsed.sha
			} else {
				sha = args[0]
				repo, err := repository.Current()
				if err != nil {
					return fmt.Errorf("detecting repository: %w", err)
				}
				owner = repo.Owner
				repoName = repo.Name
			}

			writer := &output.Writer{JSON: os.Stdout}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			fetcher := commit.NewFetcher()

			return poller.Run(ctx, poller.Config[commit.State]{
				Fetch: func() (*commit.State, error) {
					return fetcher.Fetch(owner, repoName, sha)
				},
				Diff: func(old, new *commit.State) []events.Event {
					return events.DiffCommit(old, new)
				},
				IsTerminal: func(e events.Event) bool {
					return e.Event == events.CIAllPassed || e.Event == events.CIFailed
				},
				Strategy: &poller.FixedStrategy{
					Duration: interval,
				},
				OnEvents: writer.WriteEvents,
				ExitOn:   buildExitOnEvents(exit, exitOn),
			})
		},
	}

	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(commitCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildExitOnEvents(exit bool, exitOn []string) []events.EventType {
	var exitOnEvents []events.EventType
	if exit && len(exitOn) == 0 {
		exitOnEvents = []events.EventType{events.AnyEvent}
	}
	for _, e := range exitOn {
		exitOnEvents = append(exitOnEvents, events.EventType(e))
	}
	return exitOnEvents
}

type commitRef struct {
	owner string
	repo  string
	sha   string
}

func parseCommitURL(s string) (*commitRef, error) {
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("not a URL")
	}

	// Expected: /owner/repo/commit/sha
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "commit" {
		return nil, fmt.Errorf("not a commit URL")
	}

	return &commitRef{
		owner: parts[0],
		repo:  parts[1],
		sha:   parts[3],
	}, nil
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
