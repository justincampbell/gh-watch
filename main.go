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
	"github.com/justincampbell/gh-watch/internal/branch"
	"github.com/justincampbell/gh-watch/internal/checks"
	"github.com/justincampbell/gh-watch/internal/commit"
	"github.com/justincampbell/gh-watch/internal/events"
	"github.com/justincampbell/gh-watch/internal/output"
	"github.com/justincampbell/gh-watch/internal/poller"
	"github.com/justincampbell/gh-watch/internal/pr"
	"github.com/justincampbell/gh-watch/internal/tag"
	"github.com/spf13/cobra"
)

func main() {
	var (
		interval    time.Duration
		exit        bool
		exitOn      []string
		tagMatch    string
		contains    string
		ignoreCheck []string
		search      string
		eventNames  []string
		eventTypes  []events.EventType
	)

	rootCmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch GitHub resources for state changes",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			eventTypes, err = events.ParseSelection(eventNames)
			return err
		},
	}

	rootCmd.PersistentFlags().DurationVar(&interval, "interval", 60*time.Second, "Polling interval")
	rootCmd.PersistentFlags().BoolVar(&exit, "exit", false, "Exit after any state change (shorthand for --exit-on any)")
	rootCmd.PersistentFlags().StringSliceVar(&exitOn, "exit-on", nil, `Exit after specific event types (e.g. --exit-on ci-passed,ci-failed)`)
	rootCmd.PersistentFlags().StringSliceVar(&eventNames, "events", nil, "Only emit these event types (initial-state and terminal events always emit)")
	rootCmd.SilenceUsage = true

	prCmd := &cobra.Command{
		Use:   "pr [<number> | --search <query>]",
		Short: "Watch a pull request, or every PR matching a search, for state changes",
		Long: "Monitors a pull request, polling for changes and printing updates as they occur.\n" +
			"With --search, watches every PR in the current repository matching the query\n" +
			"(any `gh pr list --search` query), re-running it each interval so PRs join and\n" +
			"leave the set.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := repository.Current()
			if err != nil {
				return fmt.Errorf("detecting repository: %w", err)
			}

			owner := repo.Owner
			repoName := repo.Name

			ignore, err := checks.ParseIgnore(ignoreCheck)
			if err != nil {
				return err
			}

			writer := &output.Writer{JSON: os.Stdout}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			fetcher := pr.NewFetcher()

			if search != "" {
				if len(args) > 0 {
					return fmt.Errorf("pass a PR number or --search, not both")
				}
				if err := pr.ValidateSearch(search); err != nil {
					return err
				}
				searcher := &pr.SearchFetcher{Owner: owner, Repo: repoName, Query: search, Single: fetcher}
				var prev *pr.SetState
				return poller.Run(ctx, poller.Config[pr.SetState]{
					Fetch: func() (*pr.SetState, error) {
						set, err := searcher.Fetch(prev)
						if err != nil {
							return nil, err
						}
						for _, state := range set.PRs {
							state.CheckRuns = ignore.Filter(state.CheckRuns)
						}
						return set, nil
					},
					Diff: func(old, new *pr.SetState) []events.Event {
						prev = new
						return events.DiffSet(old, new)
					},
					Strategy: &poller.FixedStrategy{
						Duration: interval,
					},
					OnEvents: writer.WriteEvents,
					ExitOn:   buildExitOnEvents(exit, exitOn),
					Events:   eventTypes,
				})
			}

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

			return poller.Run(ctx, poller.Config[pr.State]{
				Fetch: func() (*pr.State, error) {
					state, err := fetcher.Fetch(owner, repoName, number)
					if err != nil {
						return nil, err
					}
					state.CheckRuns = ignore.Filter(state.CheckRuns)
					return state, nil
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
				Events:   eventTypes,
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

			ignore, err := checks.ParseIgnore(ignoreCheck)
			if err != nil {
				return err
			}

			writer := &output.Writer{JSON: os.Stdout}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			fetcher := commit.NewFetcher()

			return poller.Run(ctx, poller.Config[commit.State]{
				Fetch: func() (*commit.State, error) {
					state, err := fetcher.Fetch(owner, repoName, sha)
					if err != nil {
						return nil, err
					}
					state.CheckRuns = ignore.Filter(state.CheckRuns)
					return state, nil
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
				Events:   eventTypes,
			})
		},
	}

	branchCmd := &cobra.Command{
		Use:   "branch <name>",
		Short: "Watch a branch for new commits",
		Long:  "Monitors a branch, polling for new commits pushed to it.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := repository.Current()
			if err != nil {
				return fmt.Errorf("detecting repository: %w", err)
			}

			owner := repo.Owner
			repoName := repo.Name
			branchName := args[0]

			writer := &output.Writer{JSON: os.Stdout}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			fetcher := branch.NewFetcher()

			return poller.Run(ctx, poller.Config[branch.State]{
				Fetch: func() (*branch.State, error) {
					return fetcher.Fetch(owner, repoName, branchName)
				},
				Diff: func(old, new *branch.State) []events.Event {
					return events.DiffBranch(old, new)
				},
				Strategy: &poller.FixedStrategy{
					Duration: interval,
				},
				OnEvents: writer.WriteEvents,
				ExitOn:   buildExitOnEvents(exit, exitOn),
				Events:   eventTypes,
			})
		},
	}

	tagCmd := &cobra.Command{
		Use:   "tag",
		Short: "Watch for new tags",
		Long: "Monitors a repository's tags, polling for new tags as they are created.\n" +
			"Use --match to filter tag names by glob, and --contains to report only\n" +
			"tags whose commit includes a given commit (reporting immediately if such\n" +
			"a tag already exists).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := repository.Current()
			if err != nil {
				return fmt.Errorf("detecting repository: %w", err)
			}

			owner := repo.Owner
			repoName := repo.Name

			writer := &output.Writer{JSON: os.Stdout}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			fetcher := tag.NewFetcher()

			return poller.Run(ctx, poller.Config[tag.State]{
				Fetch: func() (*tag.State, error) {
					return fetcher.Fetch(owner, repoName, tagMatch, contains)
				},
				Diff: func(old, new *tag.State) []events.Event {
					return events.DiffTag(old, new)
				},
				Strategy: &poller.FixedStrategy{
					Duration: interval,
				},
				OnEvents: writer.WriteEvents,
				ExitOn:   buildExitOnEvents(exit, exitOn),
				Events:   eventTypes,
			})
		},
	}
	prCmd.Flags().StringVar(&search, "search", "", `Watch every PR in this repo matching a search query (e.g. "author:@me is:open")`)
	for _, cmd := range []*cobra.Command{prCmd, commitCmd} {
		cmd.Flags().StringArrayVar(&ignoreCheck, "ignore-check", nil, "Ignore checks whose name matches this glob (repeatable; * matches across /)")
	}
	tagCmd.Flags().StringVar(&tagMatch, "match", "", "Glob to filter tag names (default: all tags)")
	tagCmd.Flags().StringVar(&contains, "contains", "", "Only report tags whose commit contains this commit SHA")

	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(branchCmd)
	rootCmd.AddCommand(tagCmd)

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
