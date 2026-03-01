package internal

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

type LineageService interface {
	BeginChain() []*HubCommandLineage          //Step 1 - (Hydrate) - create HubCommandLineage objects from HubCommand objects (copying relevant fields and adding lineage metadata)
	LinkChain(cmds []*HubCommandLineage) error //Step 2 - (Chain together) - assign PrevID, NextID, RootID to HubCommandLineage objects to create a linked tracking chain
	LogLineage(lineage []*HubCommandLineage, lineageFileName string) error
}

type DBHistoryService struct {
	AuditHubCommands []*HubCommand
	Store            HubCommandStore
}

// TODO, beta thoughts (think on this) -- I think the second struct needs to be removed and keep PrevID and NextID on HubCommand
// Then any HubCommand can easily be checked for lineage then move forward or backward instead of
// checking a different table/output.
type HubCommandLineage struct {
	ID      string
	BatchID string

	// Execution lineage
	PrevID string //* want to see the actual string value stored, not the address/reference of the previous object in memory (which is what a pointer would give us)
	NextID string //*

	// Optional richer lineage
	//ParentID string  // spawned from (copied from HubCommand object in Lineage creation via HydrateLineage())
	RootID string //* workflow root (copied from first HubCommandLineage in ChainLineage())

	Status    string
	Stdout    string
	CreatedAt time.Time
}

// ///////////////////////////////////////////////////////////
// TODO - improve https://chatgpt.com/c/698c0190-d8ec-832d-8aee-537b6c64320d
func (hs *DBHistoryService) BeginChain() []*HubCommandLineage {

	if len(hs.AuditHubCommands) == 0 {
		return []*HubCommandLineage{}
	}

	lineageObjects := make([]*HubCommandLineage, 0, len(hs.AuditHubCommands))

	shortUUID, err := (&CmdIOHelper{}).NewShortUUID()

	var batchSuffix string

	if err != nil {
		PrintStdErr("UUID function fail: %v", err)
		batchSuffix = strconv.FormatInt(time.Now().UnixNano(), 10)
	} else {
		batchSuffix = shortUUID
	}

	batchID := fmt.Sprintf("batch__%s", batchSuffix)
	now := time.Now()

	for _, cmd := range hs.AuditHubCommands {

		lineageObject := &HubCommandLineage{
			ID:        cmd.ID.String(),
			BatchID:   batchID,
			Status:    cmd.Status,
			Stdout:    cmd.Stdout,
			CreatedAt: now, // or cmd.CreatedAt
		}

		lineageObjects = append(lineageObjects, lineageObject)
	}

	return lineageObjects
}

func (hs *DBHistoryService) LinkChain(
	cmds []*HubCommandLineage, //todo add history struct to keep separate table of tracking and we can join on uuid
) error {

	if len(cmds) == 0 {
		return errors.New("Chain Empty! No HubCommands to Link!")
	}

	rootID := cmds[0].ID

	for i := range cmds {
		// Root assignment
		cmds[i].RootID = rootID //&

		if i > 0 {
			//copy of UUID value (string)
			prev := cmds[i-1].ID
			cmds[i].PrevID = prev //&
		}

		if i < len(cmds)-1 {
			next := cmds[i+1].ID
			cmds[i].NextID = next //&
		}
	}

	return nil
}

// Write lineage graph to file
func (hs *DBHistoryService) LogLineage(lineage []*HubCommandLineage, lineageFileName string) error {

	f := (&CmdIOHelper{}).GetFileWrite(lineageFileName)
	if f == nil {
		err := errors.New("LINEAGE FILE ERROR")
		PrintFailure("errors.New(\"\"): %v\n", err)
		return err
	}
	defer f.Close()

	for _, cmd := range lineage {
		line := fmt.Sprintf("ID: %s, BatchID: %s, PrevID: %v, NextID: %v, Status: %s, RootID: %v\n",
			cmd.ID, cmd.BatchID, cmd.PrevID, cmd.NextID, cmd.Status, cmd.RootID)
		_, err := f.WriteString(line)
		if err != nil {
			return err
		}
	}
	return nil
}
