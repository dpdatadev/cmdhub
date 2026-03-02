package internal

import (
	"context"
	"errors"
	"time"
)

// A HubCommand Service must have access to an Executor for managing HubCommands and a Store for persisting results.
// The service handles implementation specific details and communication.
type HubCommandService struct {
	//Service
	Store    HubCommandStore //SQLITE, Redis, Postgres, DuckDB, who cares!
	Executor HubCommandExecutor
}

func NewHubCommandService(
	store HubCommandStore,
	exec HubCommandExecutor,
) *HubCommandService {
	return &HubCommandService{
		Store:    store,
		Executor: exec,
	}
}

func (s *HubCommandService) RunHubCommand(
	ctx context.Context,
	cmd *HubCommand, debug bool,
) error {

	if NewDefaultScrubber().Scrub(cmd) != nil {
		//var ioHelper CmdIOHelper
		violation := "SECURITY POLICY TRIGGERED"

		// Mark rejected
		cmd.Status = StatusRejected
		cmd.StartedAt = time.Now()
		cmd.Stdout = violation
		cmd.Stderr = violation
		cmd.EndedAt = time.Now()
		// Keep track of our rejections (Audit everything. Track everything.)
		// We may also keep security violations in a separate text log
		s.Store.Update(ctx, cmd)
		PrintFailure(violation)
		//ioHelper.FileDump(cmd, "security.log")
		//get weird results when trying to write to console and log file at same time
		//probably some stdout concurrency thing I'm not aware of (2/17, TODO)
		return errors.New(string(StatusRejected))
	}

	// Persist initial record
	if err := s.Store.Create(ctx, cmd); err != nil {
		return err
	}

	// Mark running
	cmd.Status = StatusRunning
	cmd.StartedAt = time.Now()
	s.Store.Update(ctx, cmd)

	// Execute
	result, err := s.Executor.Execute(ctx, cmd, debug)

	// Apply results
	cmd.Stdout = result.Stdout
	cmd.Stderr = result.Stderr
	cmd.ExitCode = result.ExitCode
	cmd.Error = result.Error
	cmd.EndedAt = result.EndedAt

	if err != nil {
		cmd.Status = StatusFailed
	} else {
		cmd.Status = StatusSuccess
	}

	return s.Store.Update(ctx, cmd)
}
