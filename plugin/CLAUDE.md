# gh-watch plugin

Watches GitHub resources for state changes — pull requests (CI, reviews, comments, merge conflicts), commits (CI status), and branches (new commits).

## Requires

The `gh-watch` extension must be installed:

```
gh extension install justincampbell/gh-watch
```

## Usage

Use the `/watch-pr` skill to monitor a PR after pushing code or when waiting for CI/reviews.

Use the `/watch-commit` skill to monitor a commit's CI checks — useful when waiting for a build to finish on a specific SHA.

Use the `/watch-branch` skill to monitor a branch for new commits — useful when waiting for merges to main or tracking branch activity.
