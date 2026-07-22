package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// cloneURL matches the only repository shape this system builds from: an
// https URL, optionally with userinfo (the x-access-token form of a GitHub
// App installation token). Everything else git accepts — ext:: (runs a
// shell), file://, ssh://, a leading -flag — is rejected outright rather
// than escaped: the builds worker holds the Docker socket and the App key,
// so a URL that executes is a full compromise.
var cloneURL = regexp.MustCompile(`^https://(?:[A-Za-z0-9._%+-]+(?::[^@/\s]*)?@)?[A-Za-z0-9.-]+(?::\d+)?/`)

// CloneRepo clones the given git repository URL at the specified branch into destDir.
// It performs a shallow clone (depth=1) for speed.
//
// authURL, when given, is the same repository with credentials in it — an
// installation token from the GitHub App. It is passed separately so that only
// the clean URL is ever quoted in an error: git echoes the URL it was given,
// and a failed private clone would otherwise print the token into a build log.
func CloneRepo(ctx context.Context, repoURL, branch, destDir string) error {
	return CloneRepoAs(ctx, repoURL, "", branch, destDir)
}

// CloneRepoAs clones using authURL for the fetch while reporting repoURL.
func CloneRepoAs(ctx context.Context, repoURL, authURL, branch, destDir string) error {
	if !cloneURL.MatchString(repoURL) {
		return fmt.Errorf("git clone: repository must be an https:// URL, got %q", repoURL)
	}
	// Never echo authURL: it carries the token.
	if authURL != "" && !cloneURL.MatchString(authURL) {
		return fmt.Errorf("git clone %s: credentialled URL is not https://", repoURL)
	}
	if strings.HasPrefix(branch, "-") {
		return fmt.Errorf("git clone %s: invalid branch %q", repoURL, branch)
	}

	fetchURL := repoURL
	if authURL != "" {
		fetchURL = authURL
	}

	args := []string{"clone", "--depth=1", "--single-branch"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	// "--" so neither URL nor dest can ever be parsed as a flag.
	args = append(args, "--", fetchURL, destDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	// Workers have no TTY. Without this git tries to prompt for credentials on a
	// private repo and dies with "could not read Username ... No such device",
	// which hides the real cause. Fail fast with a clear message instead.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "SSH_ASKPASS=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		// Never let the token reach a log, whatever git chose to echo.
		if authURL != "" {
			msg = strings.ReplaceAll(msg, authURL, repoURL)
			msg = redactCredentials(msg)
		}
		if strings.Contains(msg, "could not read Username") || strings.Contains(msg, "Authentication failed") ||
			strings.Contains(msg, "terminal prompts disabled") {
			return fmt.Errorf("git clone %s: repository is private or does not exist — connect credentials for this repo", repoURL)
		}
		return fmt.Errorf("git clone %s: %w\n%s", repoURL, err, msg)
	}
	return nil
}

// redactCredentials removes anything of the form scheme://user:secret@host
// from text on its way to a build log.
func redactCredentials(msg string) string {
	for {
		at := strings.Index(msg, "@")
		if at < 0 {
			return msg
		}
		start := strings.LastIndex(msg[:at], "https://")
		if start < 0 {
			return msg
		}
		msg = msg[:start+len("https://")] + msg[at+1:]
	}
}
