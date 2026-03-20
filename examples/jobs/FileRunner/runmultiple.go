package filerunner

import (
	hub "dpdigital/cmdhub/api-alpha"
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
	ctx, _ := jobs.DefaultCtx()                                       // 30 Second Timeout Context
	results := jobs.MultiExec(service, ctx, commandsToRun, willDebug) // Run each command from text file as Go routine in Wait Group

	// Display results
	for _, cmd := range results {
		hub.PrintIdentity("Executing Command Results for : %s", cmd.Name)
		hub.PrintIdentity("Status: %v", cmd.Status)
		hub.PrintIdentity("Exit Code: %d", cmd.ExitCode)
		hub.PrintIdentity("OUTPUT:: => %s\n", cmd.Stdout)
	}
}
