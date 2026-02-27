package internal

import (
	"context"
	"log"
	"sync"
)

const (
	_ = iota
	CONSOLE_RUN
	FLATFILE_RUN
	HTTP_RUN
	UDP_RUN
)

// Types built for Automation and Testing purposes
type CommandRunner interface {
	RunCommands(svc *CommandService, ctx context.Context, cmds []*Command, debug bool) []*Command
}

type ConsoleCommandRunner struct{}
type HTTPCommandRunner struct{}
type UDPCommandRunner struct{}
type FlatFileCommandRunner struct{}

// TODO!
func NewCommandRunner(runnerType uint) CommandRunner {
	// Not yet implemented
	/*
		switch runnerType {
		case RunnerType_Console:
			return &ConsoleCommandRunner{}
		case RunnerType_FlatFile:
			return &FlatFileCommandRunner{}
		case RunnerType_HTTP:
			return &HTTPCommandRunner{}
		case RunnerType_UDP:
			return &UDPCommandRunner{}
		}
	*/
	log.Printf("\n::Console Runner Selected (default): %v", runnerType)
	return &ConsoleCommandRunner{}
}

func (ccr *ConsoleCommandRunner) RunCommands(
	svc *CommandService,
	ctx context.Context,
	cmds []*Command, debug bool,
) []*Command {

	var wg sync.WaitGroup

	finished := make([]*Command, len(cmds))

	for i, cmd := range cmds {
		wg.Add(1)

		go func(i int, cmd *Command) {
			defer wg.Done()

			if err := svc.RunCommand(ctx, cmd, debug); err != nil {
				PrintFailure("\nERR --> See Logs::<<%v>>::\n", err)
			}

			finished[i] = cmd
		}(i, cmd)
	}

	wg.Wait()
	return finished
}

//End FRAMEWORK
