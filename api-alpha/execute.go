package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/goforj/execx"
)

// Stores the results/metadata of a HubCommand Operation
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    string

	StartedAt time.Time
	EndedAt   time.Time
	Duration  time.Duration
}

// Define the Executor contract for the Service layer.
type HubCommandExecutor interface {
	Execute(
		ctx context.Context,
		cmd *HubCommand, debug bool,
	) (*ExecutionResult, error)
}

type BaseExecutor struct{}

// Local (not Remote) HubCommand Executor (Default)
// os.StartProcess
type LocalExecutor struct {
	BaseExecutor
}

// TODO, BETA
type SSHExecutor struct {
	BaseExecutor
}

func (le *BaseExecutor) debugDump(cmd *HubCommand, er *ExecutionResult, logFileName string) {

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
	log.Println("HubCommand Started: ", cmd.StartedAt)
	log.Println("Execution Ended: ", er.EndedAt)
	log.Println("Duration: ", er.Duration)
	log.Println("ExitCode: ", er.ExitCode)
	log.Println("::END EXECUTION::")
	log.Println("===========================================================================================")
}

func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

// Execx Fluent API - for advanced piping, complex pipelines,
// and cross platform support.
func (e *LocalExecutor) xExecute(
	ctx context.Context,
	//Pass custom configured execx.Cmd
	cmd *execx.Cmd, debug bool,
) (*ExecutionResult, error) {

	start := time.Now()

	result := &ExecutionResult{
		StartedAt: start,
	}

	res, err := cmd.WithContext(ctx).Run()

	if !res.OK() {
		PrintFailure("HubCommand failed: %v", err)
	}

	if err != nil {
		result.Error = err.Error()
	}

	end := time.Now()

	result.Stdout = res.Stdout
	result.Stderr = res.Stderr
	result.ExitCode = res.ExitCode
	result.EndedAt = end
	result.Duration = end.Sub(start)

	if debug {
		PrintDebug("Stdout: %q\n", res.Stdout)
		PrintDebug("Stderr: %q\n", res.Stderr)
		PrintDebug("ExitCode: %d\n", res.ExitCode)
		PrintDebug("Error: %v\n", res.Err)
		PrintDebug("Duration: %v\n", res.Duration)
		go e.debugDump(NewHubCommand(fmt.Sprintf("execx Cmd: %s", cmd.String()), []string{cmd.Args()[0]}, result.Stdout), result, "executions.log")
	}

	return result, err
}

// Default Hub Command API
func (e *LocalExecutor) Execute(
	ctx context.Context,
	cmd *HubCommand, debug bool,
) (*ExecutionResult, error) {

	//temporary, kinda hacky
	if cmd.XCmd != nil {
		return e.xExecute(ctx, cmd.XCmd, true)
	}

	start := time.Now()

	c := exec.CommandContext(ctx, cmd.Name, cmd.Args...)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	//Security Audit/Scub check happens in Service, only valid HubCommands make it to the Executor
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
