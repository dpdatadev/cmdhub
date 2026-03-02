package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

//BETA:
//I have made a struct field datatype/API call dependent on a third party library.
//Only string should be used to represent the ID - the UUID implementation abstracted away.
//Now if the UUID library gets changed, I'll have to update a crap load of code.
//In the database, the UUID's are stored as string representations of UUID's, so string
//should be all the caller needs to work with.

//TODO, add username to lineage log and database

func GetSQLITEDB(databaseName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("./%s.db", databaseName)) //TODO, hanlde paths
	if err != nil {
		PrintFailure("Error opening database: %v", err)
		return nil, err
	}
	return db, nil
}

type HubCommandStore interface {
	//Store a single HubCommand (InMemory, SQLITE, FlatFile)
	Create(ctx context.Context, cmd *HubCommand) error
	//Retrieve a single HubCommand Record
	GetByID(ctx context.Context, id uuid.UUID) (*HubCommand, error)
	//Retrieve all previous HubCommand Records (return any depends on implementation)
	GetAll(ctx context.Context) ([]*HubCommand, error)
	//Update a single HubCommand record, usually called internally for updating active HubCommands
	Update(ctx context.Context, cmd *HubCommand) error
	//Delete - Stores don't delete. No compromised Audit trail. Every execution captured.
	//Store size can be managed separately
}

// Sharing behavior for all HubCommandStore implementations (InMemory, SQLITE, FlatFile)
type BaseHubCommandStore struct{}

func (b *BaseHubCommandStore) CanStore(cmd *HubCommand) bool {
	var canBeSavedToStore bool
	if cmd == nil || cmd.ID == uuid.Nil || strings.TrimSpace(cmd.ID.String()) == "" {
		canBeSavedToStore = false
	} else {
		canBeSavedToStore = true
	}

	return canBeSavedToStore
}

type InMemoryHubCommandStore struct {
	BaseHubCommandStore
	mu   sync.RWMutex
	data map[uuid.UUID]*HubCommand
}

/* SQLITE impl */ // TODO, testing
// 2/16, flesh out, test, conform to HubCommandStore interface
type SQLiteHubCommandStore struct {
	BaseHubCommandStore
	db *sql.DB
}

func NewSqliteHubCommandStore(db *sql.DB) *SQLiteHubCommandStore {
	return &SQLiteHubCommandStore{db: db}
}

func (s *SQLiteHubCommandStore) GetAll(
	ctx context.Context,
) ([]*HubCommand, error) {

	query := `
        SELECT
            uuid,
            name,
            status,
            created_at,
            started_at,
            finished_at,
            stdout,
            stderr,
            exit_code
        FROM Commands
        ORDER BY created_at DESC
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var HubCommands []*HubCommand

	for rows.Next() {
		cmd := new(HubCommand)

		err := rows.Scan(
			&cmd.ID,
			&cmd.Name,
			&cmd.Status,
			&cmd.CreatedAt,
			&cmd.StartedAt,
			&cmd.EndedAt,
			&cmd.Stdout,
			&cmd.Stderr,
			&cmd.ExitCode,
		)
		if err != nil {
			return nil, err
		}

		HubCommands = append(HubCommands, cmd)
	}

	return HubCommands, nil
}

func (s *SQLiteHubCommandStore) GetByID(
	ctx context.Context,
	uuid uuid.UUID,
) (*HubCommand, error) {

	query := `
        SELECT
			uuid,
            name,
            status,
            created_at,
            started_at,
            finished_at,
            stdout,
            stderr,
            exit_code
        FROM Commands
        WHERE uuid = ?
        LIMIT 1
    `
	uuidString := uuid.String()
	row := s.db.QueryRowContext(ctx, query, uuidString)

	cmd := new(HubCommand)

	err := row.Scan(
		&cmd.ID,
		&cmd.Name,
		&cmd.Status,
		&cmd.CreatedAt,
		&cmd.StartedAt,
		&cmd.EndedAt,
		&cmd.Stdout,
		&cmd.Stderr,
		&cmd.ExitCode,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(
				"HubCommand not found: %s",
				uuidString,
			)
		}
		return nil, err
	}

	return cmd, nil
}

func (s *SQLiteHubCommandStore) Update(
	ctx context.Context,
	cmd *HubCommand,
) error {

	query := `
        UPDATE Commands
        SET
            name = ?,
            status = ?,
            started_at = ?,
            finished_at = ?,
            stdout = ?,
            stderr = ?,
            exit_code = ?
        WHERE uuid = ?
    `

	result, err := s.db.ExecContext(
		ctx,
		query,
		cmd.Name,
		cmd.Status,
		cmd.StartedAt,
		cmd.EndedAt,
		cmd.Stdout,
		cmd.Stderr,
		cmd.ExitCode,
		cmd.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf(
			"no HubCommand updated (id=%s)",
			cmd.ID,
		)
	}

	return nil
}

func (s *SQLiteHubCommandStore) SaveBatch(
	ctx context.Context,
	cmds []*HubCommand,
) error {

	if len(cmds) == 0 {
		return errors.New("No HubCommands to save")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO Commands (d
            id, name, status, created_at
        )
        VALUES (?, ?, ?, ?)
    `)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, cmd := range cmds {

		if s.CanStore(cmd) {
			_, err := stmt.ExecContext(
				ctx,
				cmd.ID,
				cmd.Name,
				cmd.Status,
				cmd.CreatedAt,
			)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *SQLiteHubCommandStore) MarkStarted(
	ctx context.Context,
	id string,
	startedAt time.Time,
) error {

	query := `
        UPDATE Commands
        SET
            status = 'RUNNING',
            started_at = ?
        WHERE uuid = ?
    `

	_, err := s.db.ExecContext(
		ctx,
		query,
		startedAt,
		id,
	)

	return err
}

func (s *SQLiteHubCommandStore) MarkFinished(
	ctx context.Context,
	id string,
	finishedAt time.Time,
	exitCode int,
	stdout string,
	stderr string,
) error {

	query := `
        UPDATE Commands
        SET
            status = 'COMPLETED',
            finished_at = ?,
            exit_code = ?,
            stdout = ?,
            stderr = ?
        WHERE uuid = ?
    `

	_, err := s.db.ExecContext(
		ctx,
		query,
		finishedAt,
		exitCode,
		stdout,
		stderr,
		id,
	)

	return err
}

func (s *SQLiteHubCommandStore) GetRecent(
	ctx context.Context,
	limit uint,
) ([]*HubCommand, error) {

	query := `
        SELECT
			uuid,
            name,
            status,
            created_at,
            started_at,
            finished_at,
            stdout,
            stderr,
            exit_code        
			FROM Commands
        ORDER BY created_at DESC
        LIMIT ?
    `

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var HubCommands []*HubCommand

	for rows.Next() {
		cmd := new(HubCommand)

		err := rows.Scan(
			&cmd.ID,
			&cmd.Name,
			&cmd.Status,
			&cmd.CreatedAt,
			&cmd.StartedAt,
			&cmd.EndedAt,
			&cmd.Stdout,
			&cmd.Stderr,
			&cmd.ExitCode,
		)
		if err != nil {
			return nil, err
		}

		HubCommands = append(HubCommands, cmd)
	}

	return HubCommands, nil
}

func (s *SQLiteHubCommandStore) Create(
	ctx context.Context,
	cmd *HubCommand,
) error {

	if !s.CanStore(cmd) {
		return errors.New("HubCommand cannot be nil or have blank UUID")
	}

	query := `
        INSERT INTO Commands (
            uuid,
            name,
            status,
            created_at,
            started_at,
            finished_at,
            stdout,
            stderr,
            exit_code
        )
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	_, err := s.db.ExecContext(
		ctx,
		query,
		cmd.ID,
		cmd.Name,
		cmd.Status,
		cmd.CreatedAt,
		cmd.StartedAt,
		cmd.EndedAt,
		cmd.Stdout,
		cmd.Stderr,
		cmd.ExitCode,
	)

	return err
}

//TODO - Postgres store

func NewInMemoryStore() *InMemoryHubCommandStore {
	return &InMemoryHubCommandStore{
		data: make(map[uuid.UUID]*HubCommand),
	}
}

// Depending on how large the map gets, this will help recycle memory.
// See here --> https://medium.com/@caring_smitten_gerbil_914/go-maps-and-hidden-memory-leaks-what-every-developer-should-know-17b322b177eb
// This may not be neccessary since we are storing single value pointers as opposed to 128 byte buckets
// Though, we know now that some of the HubCommand data will grow fairly large, UTF-8 text.
func (s *InMemoryHubCommandStore) shrinkMap(old map[uuid.UUID]*HubCommand) map[uuid.UUID]*HubCommand {
	newMap := make(map[uuid.UUID]*HubCommand, len(old))
	maps.Copy(newMap, old)
	return newMap
}

func (s *InMemoryHubCommandStore) Create(
	ctx context.Context,
	cmd *HubCommand,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[cmd.ID] = cmd
	return nil
}

func (s *InMemoryHubCommandStore) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*HubCommand, error) {

	if id == uuid.Nil {
		return nil, errors.New("invalid UUID")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	cmd, ok := s.data[id]
	if !ok {
		return nil, errors.New("HubCommand not found")
	}

	return cmd, nil
}

// InMemoryStore puts data in Map (for various reasons), but API always works with an Array/Slice
func (s *InMemoryHubCommandStore) GetAll(ctx context.Context) ([]*HubCommand, error) {

	// Honor context cancellation
	//Overkill here, add in SQL and Network Stores (usually handled for us in library i.e. SQL Drivers sql.DB*)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.data) == 0 {
		return []*HubCommand{}, nil
	}

	mapToList := make([]*HubCommand, 0, len(s.data))

	for _, cmd := range s.data {
		mapToList = append(mapToList, cmd)
	}

	return mapToList, nil
}

// internal function for InMemoryHubCommandStore to return the in memory map as is (not a List)
func (s *InMemoryHubCommandStore) memoryMap() (map[uuid.UUID]*HubCommand, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data, nil
}

func (s *InMemoryHubCommandStore) Update(
	ctx context.Context,
	cmd *HubCommand,
) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[cmd.ID] = cmd
	return nil
}

//SQL Store Implementation (SQLITE default)
/*
CREATE TABLE IF NOT EXISTS HubCommands (
    id	 		  PRIMARY KEY AUTOINCREMENT
	uuid          TEXT,
    name          TEXT NOT NULL,
    status        TEXT NOT NULL,
    created_at    DATETIME NOT NULL,
    started_at    DATETIME,
    finished_at   DATETIME,
    stdout        TEXT,
    stderr        TEXT,
    exit_code     INTEGER
);

CREATE INDEX idx_HubCommands_created_at
ON HubCommands(created_at DESC);
*/
