package runtime

import (
	"context"
	"fmt"
	"os"
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
	// Workers have no TTY. Without this git tries to prompt for credentials on a
	// private repo and dies with "could not read Username ... No such device",
	// which hides the real cause. Fail fast with a clear message instead.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "SSH_ASKPASS=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "could not read Username") || strings.Contains(msg, "Authentication failed") ||
			strings.Contains(msg, "terminal prompts disabled") {
			return fmt.Errorf("git clone %s: repository is private or does not exist — connect credentials for this repo", repoURL)
		}
		return fmt.Errorf("git clone %s: %w\n%s", repoURL, err, msg)
	}
	return nil
}
