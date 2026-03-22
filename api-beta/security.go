package api

import (
	"errors"
	"slices"
	"strings"
)

// TODO, add a way for the user to add more Deny HubCommands
var DefaultDenyHubCommands = []string{
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
	"echo", // users do not directly ask for what they want to see, valid HubCommands/programs control output
}

// TODO, add a way for the user to add more Deny HubCommands
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

type HubCommandScrubber interface {
	Scrub(cmd HubCommand) error
}

type ScrubPolicy struct {
	DenyHubCommands []string
	DenyPatterns    []string
	ProtectedPaths  []string

	AllowHubCommands []string // optional allowlist mode
	AllowMode        bool
}

type DefaultScrubber struct {
	Policy ScrubPolicy
}

func NewDefaultScrubber() *DefaultScrubber {
	return &DefaultScrubber{
		Policy: ScrubPolicy{
			DenyHubCommands: DefaultDenyHubCommands,
			DenyPatterns:    DefaultDenyPatterns,
			ProtectedPaths:  DefaultProtectedPaths,
			AllowMode:       false,
		},
	}
}

func (s *DefaultScrubber) Scrub(
	cmd *HubCommand,
) error {

	name := strings.ToLower(cmd.Name)

	// ---- Allowlist Mode ----
	if s.Policy.AllowMode {
		if !slices.Contains(s.Policy.AllowHubCommands, name) {
			return errors.New("HubCommand not in allowlist")
		}
	}

	// ---- Deny HubCommand ----
	if slices.Contains(s.Policy.DenyHubCommands, name) {
		return errors.New("HubCommand denied by policy: " + name)
	}

	// ---- Argument String ----
	full := name + " " + strings.Join(cmd.Args, " ")
	full = strings.ToLower(full)

	// ---- Pattern Checks ----
	for _, pattern := range s.Policy.DenyPatterns {
		if strings.Contains(full, pattern) {
			return errors.New(
				"HubCommand contains dangerous pattern: " + pattern,
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
