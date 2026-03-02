package main

import (
	"context"
	hub "dpdigital/cmdhub/api-alpha"
	"sync"
	"time"
)

func getDefaultCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// Helper/Testing function to run multiple Commands
func CommandRunner(
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
