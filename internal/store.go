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
)

//BETA:
//I have made a struct field datatype/API call dependent on a third party library.
//Only string should be used to represent the ID - the UUID implementation abstracted away.
//Now if the UUID library gets changed, I'll have to update a crap load of code.
//In the database, the UUID's are stored as string representations of UUID's, so string
//should be all the caller needs to work with.

func GetSQLITEDB(databaseName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("./%s.db", databaseName)) //TODO, hanlde paths
	if err != nil {
		PrintFailure("Error opening database: %v", err)
		return nil, err
	}
	return db, nil
}

type CommandStore interface {
	//Store a single Command (InMemory, SQLITE, FlatFile)
	Create(ctx context.Context, cmd *Command) error
	//Retrieve a single Command Record
	GetByID(ctx context.Context, id uuid.UUID) (*Command, error)
	//Retrieve all previous Command Records (return any depends on implementation)
	GetAll(ctx context.Context) ([]*Command, error)
	//Update a single Command record, usually called internally for updating active Commands
	Update(ctx context.Context, cmd *Command) error
	//Delete - Stores don't delete. No compromised Audit trail. Every execution captured.
	//Store size can be managed separately
}

// Sharing behavior for all CommandStore implementations (InMemory, SQLITE, FlatFile)
type BaseCommandStore struct{}

func (b *BaseCommandStore) CanStore(cmd *Command) bool {
	var canBeSavedToStore bool
	if cmd == nil || cmd.ID == uuid.Nil || strings.TrimSpace(cmd.ID.String()) == "" {
		canBeSavedToStore = false
	} else {
		canBeSavedToStore = true
	}

	return canBeSavedToStore
}

type InMemoryCommandStore struct {
	BaseCommandStore
	mu   sync.RWMutex
	data map[uuid.UUID]*Command
}

/* SQLITE impl */ // TODO, testing
// 2/16, flesh out, test, conform to CommandStore interface
type SQLiteCommandStore struct {
	BaseCommandStore
	db *sql.DB
}

func NewSqliteCommandStore(db *sql.DB) *SQLiteCommandStore {
	return &SQLiteCommandStore{db: db}
}

func (s *SQLiteCommandStore) GetAll(
	ctx context.Context,
) ([]*Command, error) {

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
        FROM commands
        ORDER BY created_at DESC
    `

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []*Command

	for rows.Next() {
		cmd := new(Command)

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

		commands = append(commands, cmd)
	}

	return commands, nil
}

func (s *SQLiteCommandStore) GetByID(
	ctx context.Context,
	uuid uuid.UUID,
) (*Command, error) {

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
        FROM commands
        WHERE uuid = ?
        LIMIT 1
    `
	uuidString := uuid.String()
	row := s.db.QueryRowContext(ctx, query, uuidString)

	cmd := new(Command)

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
				"command not found: %s",
				uuidString,
			)
		}
		return nil, err
	}

	return cmd, nil
}

func (s *SQLiteCommandStore) Update(
	ctx context.Context,
	cmd *Command,
) error {

	query := `
        UPDATE commands
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
			"no command updated (id=%s)",
			cmd.ID,
		)
	}

	return nil
}

func (s *SQLiteCommandStore) SaveBatch(
	ctx context.Context,
	cmds []*Command,
) error {

	if len(cmds) == 0 {
		return errors.New("No Commands to save")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO commands (
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
				cmd.ExecString(),
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

func (s *SQLiteCommandStore) MarkStarted(
	ctx context.Context,
	id string,
	startedAt time.Time,
) error {

	query := `
        UPDATE commands
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

func (s *SQLiteCommandStore) MarkFinished(
	ctx context.Context,
	id string,
	finishedAt time.Time,
	exitCode int,
	stdout string,
	stderr string,
) error {

	query := `
        UPDATE commands
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

func (s *SQLiteCommandStore) GetRecent(
	ctx context.Context,
	limit uint,
) ([]*Command, error) {

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
			FROM commands
        ORDER BY created_at DESC
        LIMIT ?
    `

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []*Command

	for rows.Next() {
		cmd := new(Command)

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

		commands = append(commands, cmd)
	}

	return commands, nil
}

func (s *SQLiteCommandStore) Create(
	ctx context.Context,
	cmd *Command,
) error {

	if !s.CanStore(cmd) {
		return errors.New("Command cannot be nil or have blank UUID")
	}

	query := `
        INSERT INTO commands (
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
		cmd.ExecString(),
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

func NewInMemoryStore() *InMemoryCommandStore {
	return &InMemoryCommandStore{
		data: make(map[uuid.UUID]*Command),
	}
}

// Depending on how large the map gets, this will help recycle memory.
// See here --> https://medium.com/@caring_smitten_gerbil_914/go-maps-and-hidden-memory-leaks-what-every-developer-should-know-17b322b177eb
// This may not be neccessary since we are storing single value pointers as opposed to 128 byte buckets
// Though, we know now that some of the command data will grow fairly large, UTF-8 text.
func (s *InMemoryCommandStore) shrinkMap(old map[uuid.UUID]*Command) map[uuid.UUID]*Command {
	newMap := make(map[uuid.UUID]*Command, len(old))
	maps.Copy(newMap, old)
	return newMap
}

func (s *InMemoryCommandStore) Create(
	ctx context.Context,
	cmd *Command,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[cmd.ID] = cmd
	return nil
}

func (s *InMemoryCommandStore) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Command, error) {

	if id == uuid.Nil {
		return nil, errors.New("invalid UUID")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	cmd, ok := s.data[id]
	if !ok {
		return nil, errors.New("command not found")
	}

	return cmd, nil
}

// InMemoryStore puts data in Map (for various reasons), but API always works with an Array/Slice
func (s *InMemoryCommandStore) GetAll(ctx context.Context) ([]*Command, error) {

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
		return []*Command{}, nil
	}

	mapToList := make([]*Command, 0, len(s.data))

	for _, cmd := range s.data {
		mapToList = append(mapToList, cmd)
	}

	return mapToList, nil
}

// internal function for InMemoryCommandStore to return the in memory map as is (not a List)
func (s *InMemoryCommandStore) memoryMap() (map[uuid.UUID]*Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data, nil
}

func (s *InMemoryCommandStore) Update(
	ctx context.Context,
	cmd *Command,
) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[cmd.ID] = cmd
	return nil
}

//SQL Store Implementation (SQLITE default)
/*
CREATE TABLE IF NOT EXISTS commands (
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

CREATE INDEX idx_commands_created_at
ON commands(created_at DESC);
*/

//Additional feature (for lineage and testing)

func (s *SQLiteCommandStore) SaveBatchHistory(
	ctx context.Context,
	cmds []*Command,
) error {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO commands (
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

	return tx.Commit()
}
