package internal

//BETA FEATURE

// Lineage/tracker impl
import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Lineage interface {
	BeginChain() []*CommandLineage          //Step 1 - (Hydrate) - create CommandLineage objects from Command objects (copying relevant fields and adding lineage metadata)
	LinkChain(cmds []*CommandLineage) error //Step 2 - (Chain together) - assign PrevID, NextID, RootID to CommandLineage objects to create a linked tracking chain
	LogLineage(lineage []*CommandLineage, lineageFileName string) error
}

type DBHistoryService struct {
	AuditCommands []*HubCommand
	Store         HubCommandStore
}

// TODO, beta thoughts (think on this) -- I think the second struct needs to be removed and keep PrevID and NextID on Command
// Then any Command can easily be checked for lineage then move forward or backward instead of
// checking a different table/output.

// 3-15-2026
// TODO, beta - we are def going to remove this Lineage struct. Right now the lineage objects are being stored in the same Command table anyway.
// as I stated above - we will just put the PrevID and NextID fields on Command.
type CommandLineage struct {
	ID      string
	Name    string
	BatchID string

	// Execution lineage
	PrevID string //* want to see the actual string value stored, not the address/reference of the previous object in memory (which is what a pointer would give us)
	NextID string //*

	// Optional richer lineage
	//ParentID string  // spawned from (copied from Command object in Lineage creation via HydrateLineage())
	RootID string //* workflow root (copied from first CommandLineage in ChainLineage())

	Status    string
	Stdout    string
	CreatedAt time.Time
}

// ///////////////////////////////////////////////////////////
// TODO - improve https://chatgpt.com/c/698c0190-d8ec-832d-8aee-537b6c64320d
func (hs *DBHistoryService) BeginChain() []*CommandLineage {

	if len(hs.AuditCommands) == 0 {
		return []*CommandLineage{}
	}

	lineageObjects := make([]*CommandLineage, 0, len(hs.AuditCommands))

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

	for _, cmd := range hs.AuditCommands {

		lineageObject := &CommandLineage{
			ID:        cmd.ID.String(),
			Name:      cmd.Name,
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
	cmds []*CommandLineage, //todo add history struct to keep separate table of tracking and we can join on uuid
) error {

	if len(cmds) == 0 {
		return errors.New("Chain Empty! No Commands to Link!")
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
func (hs *DBHistoryService) LogLineage(lineage []*CommandLineage, lineageFileName string, persist bool) error {

	if len(lineage) == 0 {
		return errors.New("cannot persist lineage: slice empty\n")
	}

	f := (&CmdIOHelper{}).GetFileWrite(lineageFileName)
	if f == nil {
		err := errors.New("LINEAGE FILE ERROR")
		PrintFailure("errors.New(\"\"): %v\n", err)
		return err
	}
	defer f.Close()

	var lineageBuilder strings.Builder
	var persistString string

	for _, cmd := range lineage {
		line := fmt.Sprintf("ID: %s, Name: %s, BatchID: %s, PrevID: %v, NextID: %v, Status: %s, RootID: %s\n",
			cmd.ID, cmd.Name, cmd.BatchID, cmd.PrevID, cmd.NextID, cmd.Status, cmd.RootID)
		lineageBuilder.WriteString(line)
	}
	persistString = lineageBuilder.String()
	_, err := f.WriteString(persistString)
	if err != nil {
		return err
	}

	PrintDebug("LINEAGE COUNT: %d\n", len(lineage))

	//currently a way to store lineage in same table as commands (temporary)
	if persist {
		PrintIdentity("Saving / Tracking %d items in DB\n", len(lineage))
		rootID := lineage[0].RootID
		PrintDebug(persistString)
		return hs.persistLineage(rootID, persistString)
	}

	return nil
}

func (hs *DBHistoryService) persistLineage(rootID string, lineageLog string) error {
	PrintDebug("[+]Begin DB LINLOG[+]\n")
	ctx, _ := DefaultCtx()
	cmd := &HubCommand{ID: uuid.New(), Name: fmt.Sprintf("lineage_execution_%s", rootID), Stdout: lineageLog, Status: StatusTracked, CreatedAt: time.Now().Local()}

	if hs.Store == nil {
		err := errors.New("[!!]STORE IS NIL[!!]")
		PrintFailure(err.Error())
		return errors.New(err.Error())
	}

	err := hs.Store.Create(ctx, cmd)

	PrintDebug("[-]END DB LINLOG[-]\n")

	if err != nil {
		PrintFailure("PERSIST LINEAGE FAILED: %s\n", err.Error())
		return err
	}

	PrintSuccess("[=]LINLOG COMPLETE[=]\n")
	PrintSuccess("\nRAN AS USER:: %s\n", cmd.GetUserName())

	return nil
}
