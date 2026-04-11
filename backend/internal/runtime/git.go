package runtime

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CloneRepo clones the given git repository URL at the specified branch into destDir.
// It performs a shallow clone (depth=1) for speed.
func CloneRepo(ctx context.Context, repoURL, branch, destDir string) error {
	args := []string{"clone", "--depth=1", "--single-branch"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, repoURL, destDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %w\n%s", repoURL, err, strings.TrimSpace(string(out)))
	}
	return nil
}
