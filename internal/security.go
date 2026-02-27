package internal

import (
	"errors"
	"slices"
	"strings"
)

// TODO, add a way for the user to add more Deny Commands
var DefaultDenyCommands = []string{
	"sudo",
	"rm",
	"dd",
	"mkfs",
	"shutdown",
	"reboot",
	"halt",
	"poweroff",
	"init",
	"kill",
	"killall",
	"pkill",
	"chmod",
	"chown",
	"mount",
	"umount",
	"iptables",
	"cat",
	"echo", // users do not directly ask for what they want to see, valid commands/programs control output
}

// TODO, add a way for the user to add more Deny Commands
var DefaultDenyPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	":(){ :|:& };:", // fork bomb
	"dd if=",
	"mkfs.",
	"> /dev/",
	"/etc/passwd",
}

var DefaultProtectedPaths = []string{
	"/",
	"/boot",
	"/etc",
	"/bin",
	"/usr",
	"/lib",
	"/sys",
	//"/proc",
	"/dev",
}

type CommandScrubber interface {
	Scrub(cmd Command) error
}

type ScrubPolicy struct {
	DenyCommands   []string
	DenyPatterns   []string
	ProtectedPaths []string

	AllowCommands []string // optional allowlist mode
	AllowMode     bool
}

type DefaultScrubber struct {
	Policy ScrubPolicy
}

func NewDefaultScrubber() *DefaultScrubber {
	return &DefaultScrubber{
		Policy: ScrubPolicy{
			DenyCommands:   DefaultDenyCommands,
			DenyPatterns:   DefaultDenyPatterns,
			ProtectedPaths: DefaultProtectedPaths,
			AllowMode:      false,
		},
	}
}

func (s *DefaultScrubber) Scrub(
	cmd *Command,
) error {

	name := strings.ToLower(cmd.Name)

	// ---- Allowlist Mode ----
	if s.Policy.AllowMode {
		if !slices.Contains(s.Policy.AllowCommands, name) {
			return errors.New("command not in allowlist")
		}
	}

	// ---- Deny Command ----
	if slices.Contains(s.Policy.DenyCommands, name) {
		return errors.New("command denied by policy: " + name)
	}

	// ---- Argument String ----
	full := name + " " + strings.Join(cmd.Args, " ")
	full = strings.ToLower(full)

	// ---- Pattern Checks ----
	for _, pattern := range s.Policy.DenyPatterns {
		if strings.Contains(full, pattern) {
			return errors.New(
				"command contains dangerous pattern: " + pattern,
			)
		}
	}

	// ---- Protected Paths ----
	for _, arg := range cmd.Args {
		for _, path := range s.Policy.ProtectedPaths {
			if strings.HasPrefix(arg, path) {
				return errors.New(
					"operation on protected path: " + path,
				)
			}
		}
	}

	return nil
}
