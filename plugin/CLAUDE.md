# gh-watch plugin

Watches GitHub pull requests for CI status, reviews, comments, merge conflicts, and terminal states.

## Requires

The `gh-watch` extension must be installed:

```
gh extension install justincampbell/gh-watch
```

## Hook

After `git push`, automatically starts `gh watch pr` in the background if the branch has an open PR.
