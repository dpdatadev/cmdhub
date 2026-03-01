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
	Duration  time.Duration // TODO, we should handle this somewhere, important info
}

// Define the Executor contract for the Service layer. All HubCommands are Executed by Executors.
type HubCommandExecutor interface {
	Execute(
		ctx context.Context,
		cmd *HubCommand, debug bool,
	) (*ExecutionResult, error)
}

type BaseExecutor struct{}

// Local (not Remote) HubCommand Executor (Default)
type LocalExecutor struct {
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

func (e *LocalExecutor) Execute(
	ctx context.Context,
	cmd *HubCommand, debug bool,
) (*ExecutionResult, error) {

	//execx api detected - treat differently
	/*
		if cmd.xCmd != nil {
			return e.xExecute(ctx, cmd.xCmd, true)
		}
	*/

	//https://gobyexample.com/execing-processes
	//exclude binary, just checking to see if the program is installed on host
	_, notInstalled := exec.LookPath(cmd.Name)

	if notInstalled != nil {
		return nil, fmt.Errorf("ERR:: %s PROGRAM NOT INSTALLED", cmd.Name)
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

//To avoid breaking changes on pre-alpha 1 we are going to just add
//the execx functionality as a separate call so I can use new execx or built-in command structure
//Learn more about how I should redirect execx standard output (3-1)
/*
func (le *LocalExecutor) xExecute(ctx context.Context,
	cmd *execx.Cmd, debug bool) (*ExecutionResult, error) {
	start := time.Now()

	res, err := execx.
		Command("printf", "hello\nworld\n").
		Pipe("tr", "a-z", "A-Z").
		Env("MODE=demo").
		WithContext(ctx).
		OnStdout(func(line string) {
			fmt.Println("OUT:", line)
		}).
		OnStderr(func(line string) {
			fmt.Println("ERR:", line)
		}).
		Run()

	if !res.OK() {
		log.Fatalf("HubCommand failed: %v", err)
	}

	fmt.Printf("Stdout: %q\n", res.Stdout)
	fmt.Printf("Stderr: %q\n", res.Stderr)
	fmt.Printf("ExitCode: %d\n", res.ExitCode)
	fmt.Printf("Error: %v\n", res.Err)
	fmt.Printf("Duration: %v\n", res.Duration)
}
*/

func execxTest() {
	// Run executes the HubCommand and returns the result and any error.

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := execx.
		Command("printf", "hello\nworld\n").
		Pipe("tr", "a-z", "A-Z").
		Env("MODE=demo").
		WithContext(ctx).
		OnStdout(func(line string) {
			fmt.Println("OUT:", line)
		}).
		OnStderr(func(line string) {
			fmt.Println("ERR:", line)
		}).
		Run()

	if !res.OK() {
		log.Fatalf("HubCommand failed: %v", err)
	}

	fmt.Printf("Stdout: %q\n", res.Stdout)
	fmt.Printf("Stderr: %q\n", res.Stderr)
	fmt.Printf("ExitCode: %d\n", res.ExitCode)
	fmt.Printf("Error: %v\n", res.Err)
	fmt.Printf("Duration: %v\n", res.Duration)
	// OUT: HELLO
	// OUT: WORLD
	// Stdout: "HELLO\nWORLD\n"
	// Stderr: ""
	// ExitCode: 0
	// Error: <nil>
	// Duration: 10.123456ms
}
