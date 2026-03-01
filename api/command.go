package internal

import (
	"os/user"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Reporting the status of the HubCommand
const (
	StatusPending  = "PENDING"
	StatusRunning  = "RUNNING"
	StatusSuccess  = "SUCCESS"
	StatusFailed   = "FAILED"
	StatusRejected = "REJECTED (SECURITY)"
)

const (
	_ = iota
	HubCommandType_NIL
	HubCommandType_TEXT
	HubCommandType_WEB
	HubCommandType_DATA
	HubCommandType_OTHER
)

type HubCommand struct {
	ID       uuid.UUID
	Name     string
	Category int
	Args     []string
	Notes    string

	Stdout   string
	Stderr   string
	ExitCode int
	Error    string

	Status    string
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
}

func (c *HubCommand) ExecString() string {
	return c.Name + " " + strings.Join(c.Args, " ")
}

func (c *HubCommand) GetUserName() string {
	current_user, err := user.Current()
	if err != nil {
		PrintStdErr("USER LOOKUP err OCCURRED: ", err)
	}

	username := current_user.Username

	return username
}

func NewHubCommand(name string, args []string, notes string) *HubCommand {
	return &HubCommand{
		ID:        uuid.New(),
		Name:      name,
		Args:      args,
		Notes:     notes,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
}
