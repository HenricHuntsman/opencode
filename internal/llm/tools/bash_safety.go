package tools

import (
	"path/filepath"
	"strings"
)

// Best-effort classification only; the approval prompt is the real control, not a sandbox.

var bannedCommands = []string{
	"alias", "curl", "curlie", "wget", "axel", "aria2c",
	"nc", "telnet", "lynx", "w3m", "links", "httpie", "xh",
	"http-prompt", "chrome", "firefox", "safari",
}

// Commands that only inspect state. Anything that can run another program,
// modify files, or change git refs must not appear here.
var safeReadOnlyCommands = []string{
	"ls", "echo", "pwd", "date", "cal", "uptime", "whoami", "id", "groups", "printenv", "which", "type", "whereis",
	"whatis", "uname", "hostname", "df", "du", "free", "ps",

	"git status", "git log", "git diff", "git show", "git ls-files", "git ls-remote",
	"git rev-parse", "git config --get", "git config --list", "git describe", "git blame", "git shortlog",

	"go version", "go help", "go list", "go env", "go doc", "go vet",
}

// Wrappers that execute their arguments as a command.
var commandWrappers = []string{
	"sudo", "doas", "xargs", "exec", "command", "builtin", "env", "nohup", "time", "timeout", "nice", "ionice", "setsid", "stdbuf", "watch",
}

// Arguments that make an otherwise read-only command write files or run programs.
var unsafeArgs = []string{"--output", "--ext-diff", "--textconv", "--open-files-in-pager", "-o"}

func isBanned(name string) bool {
	for _, b := range bannedCommands {
		if strings.EqualFold(name, b) {
			return true
		}
	}
	return false
}

func isWrapper(name string) bool {
	for _, w := range commandWrappers {
		if strings.EqualFold(name, w) {
			return true
		}
	}
	return false
}

// segments splits a command line on control operators and newlines.
func segments(command string) []string {
	r := strings.NewReplacer(";", "\n", "&", "\n", "|", "\n", "(", "\n", ")", "\n", "`", "\n", "{", "\n", "}", "\n")
	var out []string
	for _, s := range strings.Split(r.Replace(command), "\n") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// bannedCommandIn returns the first banned command found in any segment of the
// command line, looking through wrappers and leading VAR=value assignments.
func bannedCommandIn(command string) (string, bool) {
	for _, seg := range segments(command) {
		for _, tok := range strings.Fields(seg) {
			if strings.Contains(tok, "=") && !strings.HasPrefix(tok, "=") && !strings.Contains(tok, "/") {
				continue
			}
			name := filepath.Base(strings.Trim(tok, `"'`))
			if isBanned(name) {
				return name, true
			}
			if !isWrapper(name) {
				break
			}
		}
	}
	return "", false
}

// isSafeReadOnly reports whether a command can skip the approval prompt. Only
// an exact allowlisted command word sequence followed by plain arguments
// qualifies; anything involving shell syntax requires approval.
func isSafeReadOnly(command string) bool {
	if strings.ContainsAny(command, ";&|<>`\n\r$\\\"'") {
		return false
	}
	fields := strings.Fields(strings.ToLower(command))
	if len(fields) == 0 {
		return false
	}

	for _, field := range fields {
		for _, bad := range unsafeArgs {
			if field == bad || strings.HasPrefix(field, bad+"=") {
				return false
			}
		}
	}

	for _, safe := range safeReadOnlyCommands {
		words := strings.Fields(safe)
		if len(fields) < len(words) {
			continue
		}
		match := true
		for i, w := range words {
			if fields[i] != w {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
