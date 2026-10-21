package git

// DiffLinesBase returns the commit diff-lines should be computed against. The
// PR merge base is preferred so that changes landed on the target branch after
// the PR branched are not reported as added by the PR. It falls back to the
// target branch tip when no merge base was resolved or when the merge base
// commit cannot be made available locally.
func DiffLinesBase(gitClient *Client, repoPath, mergeBase, targetTip string) string {
	if mergeBase == "" {
		gitClient.logger.Info("no merge base resolved; building diff-lines against the target branch tip", "target_sha", targetTip)
		return targetTip
	}
	if err := EnsureCommitPresent(gitClient, repoPath, mergeBase); err != nil {
		gitClient.logger.Warn("merge base commit unavailable; building diff-lines against the target branch tip",
			"merge_base_sha", mergeBase, "target_sha", targetTip, "error", err)
		return targetTip
	}
	return mergeBase
}
