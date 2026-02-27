package internal

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Reporting the status of the command
const (
	StatusPending  = "PENDING"
	StatusRunning  = "RUNNING"
	StatusSuccess  = "SUCCESS"
	StatusFailed   = "FAILED"
	StatusRejected = "REJECTED (SECURITY)"
)

const (
	_ = iota
	CommandType_NIL
	CommandType_TEXT
	CommandType_WEB
	CommandType_DATA
	CommandType_OTHER
)

type Command struct {
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

func (c *Command) ExecString() string {
	return c.Name + " " + strings.Join(c.Args, " ")
}

func NewCommand(name string, args []string, notes string) *Command {
	return &Command{
		ID:        uuid.New(),
		Name:      name,
		Args:      args,
		Notes:     notes,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
}
