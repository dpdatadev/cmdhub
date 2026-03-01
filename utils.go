package main

import (
	"context"
	"database/sql"
	hub "dpdigital/cmdhub/api-alpha"
	"fmt"
	"sync"
)

func getDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "runner.db")

	if err != nil {
		return nil, fmt.Errorf("DB NOT OPEN: %v", err)
	}

	return db, nil
}

func RunCommands(
	svc *hub.HubCommandService,
	ctx context.Context,
	cmds []*hub.HubCommand, debug bool,
) []*hub.HubCommand {

	var wg sync.WaitGroup

	finished := make([]*hub.HubCommand, len(cmds))

	for i, cmd := range cmds {
		wg.Add(1)

		go func(i int, cmd *hub.HubCommand) {
			defer wg.Done()

			if err := svc.RunHubCommand(ctx, cmd, debug); err != nil {
				hub.PrintFailure("\nERR --> See Logs::<<%v>>::\n", err)
			}

			finished[i] = cmd
		}(i, cmd)
	}

	wg.Wait()
	return finished
}
