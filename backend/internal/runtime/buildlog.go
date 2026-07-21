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

// isHousekeeping reports whether a RUN belongs to Applad rather than the
// project — tidying the generated image is not something to narrate.
func isHousekeeping(cmd string) bool {
	return strings.HasPrefix(cmd, "rm -f /usr/share/nginx/html/") ||
		strings.HasPrefix(cmd, "apk add --no-cache socat")
}
