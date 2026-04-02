More documentation to come.
PRE-BETA - EARLY DEVELOPMENT

Derek Pauley
dpdatadev@gmail.com

MIT License

Copyright (c) 2026 DP Digital Solutions, LLC

##Full Example handling Asynchoronous Job runners fed from shell commands in a text file.
In this example, persistence is handled by the HubStore (SQLITE default, could be Gorm or Redis)

##API compliant execution is handled by the executor which applies secruity scrubbers to handle off limit command text.

##The HubCommandService orchestrates the redirection of Stdout and Stderr to a database, handles security and lineage (batch id tracking and command history replay)
```go
package filerunner

import (
	hub "dpdigital/cmdhub/api-beta"
	"dpdigital/cmdhub/examples/jobs"
)

func ExecuteHub() {

	// Debug
	willDebug := true

	// Ingest commands using shell syntax from our proc.txt file
	commandsToRun := (&hub.CmdIOHelper{}).ParseHubCommands("proc.txt")

	// Get username for auditing purposes
	rootUser := commandsToRun[0].GetUserName()

	hub.PrintIdentity("Parsing Command List to execute as user: %s", rootUser)

	// Acquire Database
	db, err := hub.GetSQLITEDB("testcmd4")

	if err != nil {
		hub.PrintFailure("CANT CONNECT TO DB, %s", err.Error())
	}

	// Bootstrap Command Hub
	executor := hub.NewLocalExecutor()                                // Default local Executor
	store := hub.NewSqliteHubCommandStore(db)                         // Sqlite DB for persistence
	service := hub.NewHubCommandService(store, executor)              // Orchestrator
	ctx, _ := hub.LongRunningCtx()                                    // 30 Second Timeout Context
	results := jobs.MultiExec(service, ctx, commandsToRun, willDebug) // Run each command from text file as Go routine in Wait Group

	// Display results
	for _, cmd := range results {
		hub.PrintIdentity("Executing Command Results for : %s", cmd.Name)
		hub.PrintIdentity("Status: %v", cmd.Status)
		hub.PrintIdentity("Exit Code: %d", cmd.ExitCode)
		hub.PrintIdentity("OUTPUT:: => %s\n", cmd.Stdout)
	}
}
```

Using third party Execx to run a system command to dump Hackernews - with Security policy and Persistance Layer manually configured instead of default Service layer abstraction (NewHubCommandService)
```
import (
	hub "dpdigital/cmdhub/api-beta"
	"log"
	"os"
	"time"

	"github.com/goforj/execx"
)

// This could be loaded via JSON, RPC, RDBMS, or YAML.
var (
	CMD  = "lynx"
	ARGS = []string{
		"-dump",
		"-nolist",
		"https://news.ycombinator.com/",
	}
)

func Dump() {
	//Open database
	db, err := hub.GetSQLITEDB("testcmd4")

	if err != nil {
		hub.PrintFailure("DB NOT OPEN: %s", err.Error())
	}
	//Create repo (crud store)
	//This is a default DAL provided by the framework to get up quickly.
	//You're free to use absolutely any store you want. Any store.
	//You just have to get the HubCommand schema and write more code. :)
	//For quickly testing your own Service implementations, I provided an "InMemoryHubStore".
	//see (store.go)
	store := hub.NewSqliteHubCommandStore(db)

	//User will provide their own context that fits their needs
	ctx, cancel := hub.DefaultCtx() //(runutils.go) Background ctx w/ 10 second timeout
	defer cancel()                  //dont forget

	//BYOC (Bring your own Command)
	//The user is choosing to use a third party command
	//instead of the default implementation or raw os.Exec.

	//Execx fluent api for pipelining commands
	//and flexibly redirecting output
	res := execx.
		Command(CMD, ARGS...).
		WithContext(ctx).
		//Callback functions to redirect outputs
		OnStdout(func(line string) {
			log.Println(line) //Do something really fancy here.
		}).
		OnStderr(func(line string) {
			hub.PrintFailure(line) //err output red and bold
		}) //Multiple quick output helpers for debugging/testing found in (ioutils.go)

	//HubCommand is the wrapper DTO that will store our command metadata
	//to be saved in the database and log files, assigns UUID and Timestamps

	//Since we are using a third party command, we just copy the fields and put a note.
	dbCmd := hub.NewHubCommand(res.String(), []string{}, "execx command")

	//You could bypass all the code below by just using the Command service and attach dbCmd to it like this:
	//dbCmd.XCmd = res //

	//Start the clock
	dbCmd.StartedAt = time.Now()

	//Default security policy (no chown, no kernel stuff, no "rm -rf.")
	if hub.NewDefaultScrubber().Scrub(dbCmd) != nil {

		violation := "SECURITY POLICY TRIGGERED"

		// Mark rejected
		dbCmd.Status = hub.StatusRejected
		dbCmd.StartedAt = time.Now()
		dbCmd.Stdout = violation
		dbCmd.Stderr = violation
		dbCmd.EndedAt = time.Now()
		hub.PrintFailure("VIOLATION: %s", dbCmd.Name)
		os.Exit(-1) // In production, return err.
	}
	//No violations, now we start work
	dbCmd.Status = hub.StatusRunning

	// Run the command
	// execx.Run executes the command (os.StartProcess under the hood)
	// and returns the result along with any error(s).
	result, runErr := res.Run()

	//There was a problem with the Execution itself (invalid arguments, OS Errors, etc.,)
	if runErr != nil {
		hub.PrintFailure("RUNTIME ERROR ON hub COMMAND %s", runErr.Error())
		os.Exit(-1)
	}

	//Command executed but had errors and/ non 0 Exit Code.
	if !result.OK() {
		hub.PrintStdErr("Command executed with ERROR(s)::Exit code:%d", result.ExitCode)
	}

	//Execution Result - set fields on DTO (fields saved to database and exposed via REST/RPC, etc.,)
	//This is handled by default in (service.go)
	dbCmd.Category = hub.CommandType_WEB
	dbCmd.Stdout = result.Stdout
	dbCmd.Stderr = result.Stderr
	if result.ExitCode != 0 {
		dbCmd.Status = hub.StatusFailed
	} else {
		dbCmd.Status = hub.StatusSuccess
	}
	dbCmd.ExitCode = result.ExitCode
	dbCmd.EndedAt = time.Now()

	//Save execution to file
	(&hub.CmdIOHelper{}).FileDump(dbCmd, "executions.log")

	//Debug output
	hub.PrintDebug("ExitCode: %d\n", dbCmd.ExitCode)
	hub.PrintDebug("Error: %v\n", dbCmd.Stderr)
	hub.PrintDebug("Duration: %v\n", result.Duration)

	//The repository will make sure the command is structurally valid
	//then insert to DB
	createError := store.Create(ctx, dbCmd)
	if createError != nil { //We will search logs for unique ID (uuid)
		hub.PrintFailure("COMMAND NOT SAVED %s\n", dbCmd.ID.String())
	} else { //Hub commands also include the user who ran the process
		hub.PrintSuccess("COMMAND RAN and SAVED by user %s\n", dbCmd.GetUserName())
	}
}

```
