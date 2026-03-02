package main

//BETA development. (3-1)
//Running some tests without the service/executor layer to see
//what can be simplified and what is unneccessary.

//Focus on usage and different test cases.

import (
	"context"
	hub "dpdigital/cmdhub/api-alpha"
	"fmt"
	"time"

	"github.com/goforj/execx"
)

func main() {
	db, err := hub.GetSQLITEDB("testcmd4")

	if err != nil {
		hub.PrintFailure("DB NOT OPEN: %s", err.Error())
	}

	store := hub.NewSqliteHubCommandStore(db)

	scrubber := hub.NewDefaultScrubber()

	// Run executes the command and returns the result and any error.

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res := execx.
		Command("printf", "hello\nworld\n").
		WithContext(ctx).
		OnStdout(func(line string) {
			fmt.Println("OUT:", line)
		}).
		OnStderr(func(line string) {
			fmt.Println("ERR:", line)
		})

	dbCmd := hub.NewHubCommand(res.String(), res.Args(), "execx command")

	dbCmd.StartedAt = time.Now()

	if scrubber.Scrub(dbCmd) != nil {
		hub.PrintFailure("SECURITY VIOLATION %s", dbCmd.Name)
	}

	result, runErr := res.Run()

	if runErr != nil {
		hub.PrintFailure("RUNTIME ERROR ON hub COMMAND %s", runErr.Error())
	}

	dbCmd.EndedAt = time.Now()
	dbCmd.Category = hub.HubCommandType_TEXT
	dbCmd.Stdout = result.Stdout
	dbCmd.Stderr = result.Stderr
	dbCmd.Status = hub.StatusSuccess
	dbCmd.ExitCode = result.ExitCode

	(&hub.CmdIOHelper{}).ConsoleDump(dbCmd)
	(&hub.CmdIOHelper{}).FileDump(dbCmd, "executions.log")

	fmt.Printf("Stdout: %q\n", result.Stdout)
	fmt.Printf("Stderr: %q\n", result.Stderr)
	fmt.Printf("ExitCode: %d\n", result.ExitCode)
	fmt.Printf("Error: %v\n", result.Err)
	fmt.Printf("Duration: %v\n", result.Duration)

	//ensure command is not only secure, but structurally valid

	insertErr := store.Create(ctx, dbCmd)
	if insertErr != nil {
		hub.PrintSuccess("COMMAND RAN and SAVED by user %s", dbCmd.GetUserName())
	} else {
		hub.PrintFailure("COMMAND NOT SAVED %s", dbCmd.ExecString())
	}

}

//TODO
//issue with insert code
//COMMAND NOT SAVED (3-1) 18:02
