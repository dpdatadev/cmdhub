package main

//BETA development. (3-1)
//Running some tests without the service/executor layer to see
//what can be simplified and what is unneccessary.

//Focus on usage and different test cases.

import (
	hub "dpdigital/cmdhub/api-alpha"
	"log"
	"os"
	"time"

	"github.com/goforj/execx"
)

func GetHackerNews() {
	//Open database
	db, err := hub.GetSQLITEDB("testcmd4")

	if err != nil {
		hub.PrintFailure("DB NOT OPEN: %s", err.Error())
	}
	//Create repo (crud store)
	store := hub.NewSqliteHubCommandStore(db)

	// Run executes the command (os.StartProcess under the hood)
	// and returns the result along with any error(s).
	ctx, cancel := getDefaultCtx() //Background ctx w/ 10 second timeout
	defer cancel()                 //dont forget

	//Execx fluent api for pipelining commands
	//and flexibly redirecting output
	res := execx.
		Command("lynx", "-dump", "-nolist", "https://news.ycombinator.com/").
		WithContext(ctx).
		//Callback functions to redirect outputs
		OnStdout(func(line string) {
			log.Println(line) //cmd output not formatted
		}).
		OnStderr(func(line string) {
			hub.PrintFailure(line) //err output red and bold
		})

	//HubCommand is the wrapper DTO that will store our command metadata
	//to be saved in the database and log files, assigns UUID and Timestamps
	dbCmd := hub.NewHubCommand(res.String(), []string{}, "execx command")

	//Start the clock
	dbCmd.StartedAt = time.Now()

	//Default security policy (no root, no kernel stuff, no rm -rf. :) )
	if hub.NewDefaultScrubber().Scrub(dbCmd) != nil {

		violation := "SECURITY POLICY TRIGGERED"

		// Mark rejected
		dbCmd.Status = hub.StatusRejected
		dbCmd.StartedAt = time.Now()
		dbCmd.Stdout = violation
		dbCmd.Stderr = violation
		dbCmd.EndedAt = time.Now()
		hub.PrintFailure("VIOLATION: %s", dbCmd.Name)
		os.Exit(-1)
	}

	dbCmd.Status = hub.StatusRunning

	//Run the command
	result, runErr := res.Run()

	if runErr != nil {
		hub.PrintFailure("RUNTIME ERROR ON hub COMMAND %s", runErr.Error())
	}

	//Execution Result - set fields on DTO
	dbCmd.EndedAt = time.Now()
	dbCmd.Category = hub.HubCommandType_WEB
	dbCmd.Stdout = result.Stdout
	dbCmd.Stderr = result.Stderr
	if result.ExitCode != 0 {
		dbCmd.Status = hub.StatusFailed
	} else {
		dbCmd.Status = hub.StatusSuccess
	}
	dbCmd.ExitCode = result.ExitCode

	//IO ops
	//(&hub.CmdIOHelper{}).ConsoleDump(dbCmd)
	(&hub.CmdIOHelper{}).FileDump(dbCmd, "executions.log")

	hub.PrintDebug("ExitCode: %d\n", dbCmd.ExitCode)
	hub.PrintDebug("Error: %v\n", result.Err)
	hub.PrintDebug("Duration: %v\n", result.Duration)

	//The repository will make sure the command is structurally valid
	//then insert to DB
	createError := store.Create(ctx, dbCmd)
	if createError != nil {
		hub.PrintFailure("COMMAND NOT SAVED %s\n", dbCmd.ID.String())
	} else {
		hub.PrintSuccess("COMMAND RAN and SAVED by user %s", dbCmd.GetUserName())
	}
}
