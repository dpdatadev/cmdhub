package internal

import (
	"os/user"
	"strings"
	"time"

	"github.com/goforj/execx"
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
	CommandType_NIL
	CommandType_TEXT
	CommandType_WEB
	CommandType_DATA
	CommandType_OTHER
)

/*
Command that wraps default implementation for persistance and logging.
Default is os.Exec (Command)
Optional support for Fluent API "execx"
See: https://github.com/goforj/execx
*/
type HubCommand struct {
	ID       uuid.UUID
	Name     string
	Category int
	Args     []string
	Notes    string
	CmdFunc  func(...any) any //Treat regular Go functions as "Commands"

	Stdout   string
	Stderr   string
	ExitCode int
	Error    string

	Status    string
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time

	/*
		Allows us to either run standard os.Exec commands
		or, if xCmd is present, run fluent go4J Execx Cmd
		See: https://github.com/goforj/execx
	*/
	XCmd *execx.Cmd
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

func NewxHubCommand(name string, args []string, cmd *execx.Cmd, notes string) *HubCommand {
	return &HubCommand{
		ID:        uuid.New(),
		Name:      name,
		Args:      args,
		Notes:     notes,
		XCmd:      cmd,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
}

// TODO, add tests
func NewHubFuncCommand(name string, args []string, f func(...any) any, notes string) *HubCommand {
	return &HubCommand{
		ID:        uuid.New(),
		Name:      name,
		Args:      args,
		Notes:     notes,
		CmdFunc:   f,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
}
