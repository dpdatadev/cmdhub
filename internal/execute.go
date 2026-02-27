package internal

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os/exec"
	"time"
)

// Stores the results/metadata of a Command Operation
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    string

	StartedAt time.Time
	EndedAt   time.Time
	Duration  time.Duration // TODO, we should handle this somewhere, important info
}

// Define the Executor contract for the Service layer. All Commands are Executed by Executors.
type CommandExecutor interface {
	Execute(
		ctx context.Context,
		cmd *Command, debug bool,
	) (*ExecutionResult, error)
}

type BaseExecutor struct{}

// Local (not Remote) Command Executor (Default)
type LocalExecutor struct {
	BaseExecutor
}

func (le *BaseExecutor) debugDump(cmd *Command, er *ExecutionResult, logFileName string) {

	file := (&CmdIOHelper{}).GetFileWrite(logFileName)

	if file == nil {
		PrintFailure("errors.New(\"\"): %v\n", errors.New("FILE ERROR"))
		return
	}

	// Ensure the file is closed when the main function exits.
	defer file.Close()

	// Set the standard logger's output to the file.
	log.SetOutput(file)

	// Log messages will now be written to "application.log" instead of stderr.
	log.Println("===========================================================================================")
	log.Println("::BEGIN EXECUTION::")
	log.Println("Time: ", time.Now())
	log.Println("Name: ", cmd.Name)
	log.Println("Args: ", cmd.Args)
	log.Println("Notes: ", cmd.Notes)
	log.Println("Status: ", cmd.Status)
	log.Println("Command Started: ", cmd.StartedAt)
	log.Println("Execution Ended: ", er.EndedAt)
	log.Println("Duration: ", er.Duration)
	log.Println("ExitCode: ", er.ExitCode)
	log.Println("::END EXECUTION::")
	log.Println("===========================================================================================")
}

func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

func (e *LocalExecutor) Execute(
	ctx context.Context,
	cmd *Command, debug bool,
) (*ExecutionResult, error) {

	start := time.Now()

	c := exec.CommandContext(ctx, cmd.Name, cmd.Args...)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	//Security Audit/Scub check happens in Service, only valid commands make it to the Executor
	err := c.Run()

	end := time.Now()

	result := &ExecutionResult{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		StartedAt: start,
		EndedAt:   end,
		Duration:  end.Sub(start),
	}

	if c.ProcessState != nil {
		result.ExitCode = c.ProcessState.ExitCode()
	}

	if err != nil {
		result.Error = err.Error()
	}

	if debug {
		go e.debugDump(cmd, result, "executions.log")
	}

	return result, err
}
