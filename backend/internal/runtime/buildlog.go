package runtime

import (
	"fmt"
	"regexp"
	"strings"
)

/*
 * Turning Docker's build chatter into a deploy log somebody wants to read.
 *
 * The raw stream narrates the Dockerfile — "Step 2/5 : COPY applad-log.conf",
 * layer hashes, "Using cache". For a site Applad generated that Dockerfile
 * itself, so none of it is the user's: it describes our plumbing, in our
 * words, about a file they have never seen.
 *
 * What is theirs is the output of their own commands, which sits in the same
 * stream. So the bookkeeping is dropped and their output kept, with the
 * commands announced as phases.
 */

var (
	stepLine      = regexp.MustCompile(`^Step \d+/\d+ : (\w+) ?(.*)$`)
	layerLine     = regexp.MustCompile(`^ *---> `)
	containerLine = regexp.MustCompile(`^ *(Removed intermediate container|Successfully built|Successfully tagged) `)
)

// RenderDeployLog rewrites a raw Docker build log as a deploy log.
//
// Lines that describe the image being assembled are dropped; lines produced by
// the project's own commands are kept, under a heading naming the command that
// produced them.
func RenderDeployLog(raw string) string {
	var b strings.Builder
	inUserCommand := false

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}

		if m := stepLine.FindStringSubmatch(strings.TrimSpace(trimmed)); m != nil {
			instruction, arg := m[1], strings.TrimSpace(m[2])
			// A RUN is the project's own command, and its output is the part
			// worth reading. Everything else is how the image is put together.
			if instruction == "RUN" && !isHousekeeping(arg) {
				fmt.Fprintf(&b, "$ %s\n", arg)
				inUserCommand = true
			} else {
				inUserCommand = false
			}
			continue
		}

		if layerLine.MatchString(trimmed) || containerLine.MatchString(trimmed) {
			continue
		}

		if inUserCommand {
			b.WriteString(trimmed)
			b.WriteByte('\n')
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

var (
	exitLine   = regexp.MustCompile(`^The command '(?:/bin/sh -c )?(.+)' returned a non-zero code: (\d+)$`)
	shellNoise = []string{
		"A complete log of this run can be found in",
		"This is related to npm not being able to find a file",
	}
)

// SummariseBuildFailure reduces a failed build log to the part that explains it.
//
// The whole stream was previously stored as the error, so the console showed a
// paragraph of layer hashes with the one line that mattered buried inside it.
// The stream is kept — it is the build log, shown directly underneath — and
// what is returned here is the headline: which command failed, and the last
// thing it said before it did.
//
// The first line is the headline; anything after it is the failing output.
func SummariseBuildFailure(raw string) string {
	command, code := "", ""
	for _, line := range strings.Split(raw, "\n") {
		if m := exitLine.FindStringSubmatch(strings.TrimSpace(strings.TrimRight(line, "\r"))); m != nil {
			command, code = m[1], m[2]
		}
	}

	// The project's own output, with Docker's narration already stripped.
	var output []string
	for _, line := range strings.Split(RenderDeployLog(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "$ ") || isNoise(line) || exitLine.MatchString(line) {
			continue
		}
		output = append(output, line)
	}
	output = dropPrefixesOfOthers(output)
	// The tail: a compiler or package manager says what went wrong last, after
	// however many lines of progress nobody needs to re-read.
	if len(output) > 6 {
		output = output[len(output)-6:]
	}

	if command == "" {
		// No command failed, so the image itself could not be assembled —
		// a missing COPY source, a bad base image, a daemon that said no.
		if len(output) == 0 {
			output = tailLines(raw, 3)
		}
		return strings.TrimSpace("Build failed\n" + strings.Join(output, "\n"))
	}

	headline := fmt.Sprintf("%s exited with code %s", command, code)
	if len(output) == 0 {
		return headline
	}
	return headline + "\n" + strings.Join(output, "\n")
}

// isNoise reports whether a line is the tool telling you where to read more,
// rather than telling you what happened.
func isNoise(line string) bool {
	for _, n := range shellNoise {
		if strings.Contains(line, n) {
			return true
		}
	}
	return false
}

// dropPrefixesOfOthers removes lines that another line already says more of.
//
// npm ends a failure with a bare "npm error enoent" after the line that
// spells the same thing out; keeping both spends one of six slots saying
// nothing.
func dropPrefixesOfOthers(lines []string) []string {
	kept := make([]string, 0, len(lines))
	for i, line := range lines {
		redundant := false
		for j, other := range lines {
			if i != j && len(other) > len(line) && strings.HasPrefix(other, line) {
				redundant = true
				break
			}
		}
		if !redundant {
			kept = append(kept, line)
		}
	}
	return kept
}

func tailLines(raw string, n int) []string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// isHousekeeping reports whether a RUN belongs to Applad rather than the
// project — tidying the generated image is not something to narrate.
func isHousekeeping(cmd string) bool {
	return strings.HasPrefix(cmd, "rm -f /usr/share/nginx/html/") ||
		strings.HasPrefix(cmd, "apk add --no-cache socat")
}
