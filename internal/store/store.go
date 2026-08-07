package store

import (
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"

	"github.com/alankiri/password-memorizer-tui/internal/paths"
)

type Trainer struct {
	ID                 int64
	Label              string
	Password           string
	Level              int
	SessionsAtLevel    int
	TotalSessions      int
	LastCountedSession *time.Time
	LastResetDate      time.Time
	CreatedAt          time.Time
}

type Session struct {
	ID          int64
	TrainerID   int64
	StartedAt   time.Time
	CompletedAt *time.Time
	Repetitions int
	Errors      int
	Successful  bool
}

type DB struct {
	db *sql.DB
}

func New() (*DB, error) {
	if err := paths.EnsureDirs(); err != nil {
		return nil, err
	}
	dsn := paths.DBFile() + "?_pragma=foreign_keys(1)"
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	st := &DB{db: sqldb}
	if err := st.migrate(); err != nil {
		return nil, err
	}
	return st, nil
}

func (st *DB) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS trainers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    label TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 1,
    sessions_at_level INTEGER NOT NULL DEFAULT 0,
    total_sessions INTEGER NOT NULL DEFAULT 0,
    last_counted_session INTEGER,
    last_reset_date INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trainer_id INTEGER NOT NULL,
    started_at INTEGER NOT NULL,
    completed_at INTEGER,
    repetitions INTEGER NOT NULL DEFAULT 0,
    errors INTEGER NOT NULL DEFAULT 0,
    successful INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (trainer_id) REFERENCES trainers(id) ON DELETE CASCADE
);
`
	if _, err := st.db.Exec(schema); err != nil {
		return err
	}
	return st.migrateTrainerColumns()
}

func (st *DB) migrateTrainerColumns() error {
	rows, err := st.db.Query("PRAGMA table_info(trainers)")
	if err != nil {
		return err
	}
	defer rows.Close()

	var name string
	var hasLastCounted, hasLastReset bool
	for rows.Next() {
		var cid, notnull, pk int
		var tp string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &tp, &notnull, &dflt, &pk); err != nil {
			return err
		}
		switch name {
		case "last_counted_session":
			hasLastCounted = true
		case "last_reset_date":
			hasLastReset = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasLastCounted {
		if _, err := st.db.Exec("ALTER TABLE trainers ADD COLUMN last_counted_session INTEGER"); err != nil {
			return err
		}
	}
	if !hasLastReset {
		if _, err := st.db.Exec("ALTER TABLE trainers ADD COLUMN last_reset_date INTEGER NOT NULL DEFAULT (strftime('%s','now'))"); err != nil {
			return err
		}
		if _, err := st.db.Exec("UPDATE trainers SET last_reset_date = created_at WHERE last_reset_date IS NULL OR last_reset_date = 0"); err != nil {
			return err
		}
	}
	return nil
}

func (st *DB) Close() error {
	return st.db.Close()
}

func (st *DB) CreateTrainer(label, password string, level int) (int64, error) {
	now := time.Now().Unix()
	res, err := st.db.Exec(
		"INSERT INTO trainers (label, password, level, last_reset_date, created_at) VALUES (?, ?, ?, ?, ?)",
		label, password, level, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (st *DB) GetTrainer(id int64) (Trainer, error) {
	row := st.db.QueryRow(`
		SELECT id, label, password, level, sessions_at_level, total_sessions, last_counted_session, last_reset_date, created_at
		FROM trainers WHERE id = ?`, id)
	return scanTrainer(row)
}

func (st *DB) ListTrainers() ([]Trainer, error) {
	rows, err := st.db.Query(`
		SELECT id, label, password, level, sessions_at_level, total_sessions, last_counted_session, last_reset_date, created_at
		FROM trainers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trainers []Trainer
	for rows.Next() {
		t, err := scanTrainer(rows)
		if err != nil {
			return nil, err
		}
		trainers = append(trainers, t)
	}
	return trainers, rows.Err()
}

func (st *DB) DeleteTrainer(id int64) error {
	_, err := st.db.Exec("DELETE FROM trainers WHERE id = ?", id)
	return err
}

func (st *DB) UpdateTrainer(t Trainer) error {
	var lastCounted sql.NullInt64
	if t.LastCountedSession != nil {
		lastCounted = sql.NullInt64{Int64: t.LastCountedSession.Unix(), Valid: true}
	}
	_, err := st.db.Exec(`
		UPDATE trainers SET label = ?, password = ?, level = ?, sessions_at_level = ?,
		total_sessions = ?, last_counted_session = ?, last_reset_date = ? WHERE id = ?`,
		t.Label, t.Password, t.Level, t.SessionsAtLevel, t.TotalSessions, lastCounted, t.LastResetDate.Unix(), t.ID)
	return err
}

func (st *DB) CreateSession(s Session) (int64, error) {
	var completed sql.NullInt64
	if s.CompletedAt != nil {
		completed = sql.NullInt64{Int64: s.CompletedAt.Unix(), Valid: true}
	}
	success := 0
	if s.Successful {
		success = 1
	}
	res, err := st.db.Exec(
		"INSERT INTO sessions (trainer_id, started_at, completed_at, repetitions, errors, successful) VALUES (?, ?, ?, ?, ?, ?)",
		s.TrainerID, s.StartedAt.Unix(), completed, s.Repetitions, s.Errors, success,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (st *DB) ListSessionsForTrainer(trainerID int64) ([]Session, error) {
	rows, err := st.db.Query(`
		SELECT id, trainer_id, started_at, completed_at, repetitions, errors, successful
		FROM sessions WHERE trainer_id = ? ORDER BY started_at DESC`, trainerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTrainer(s scanner) (Trainer, error) {
	var t Trainer
	var lastCounted sql.NullInt64
	var lastReset, createdAt int64
	err := s.Scan(&t.ID, &t.Label, &t.Password, &t.Level, &t.SessionsAtLevel, &t.TotalSessions, &lastCounted, &lastReset, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, err
		}
		return t, err
	}
	if lastCounted.Valid {
		d := time.Unix(lastCounted.Int64, 0)
		t.LastCountedSession = &d
	}
	t.LastResetDate = time.Unix(lastReset, 0)
	t.CreatedAt = time.Unix(createdAt, 0)
	return t, nil
}

func scanSession(s scanner) (Session, error) {
	var sess Session
	var startedAt int64
	var completed sql.NullInt64
	var successful int
	err := s.Scan(&sess.ID, &sess.TrainerID, &startedAt, &completed, &sess.Repetitions, &sess.Errors, &successful)
	if err != nil {
		return sess, err
	}
	sess.StartedAt = time.Unix(startedAt, 0)
	if completed.Valid {
		d := time.Unix(completed.Int64, 0)
		sess.CompletedAt = &d
	}
	sess.Successful = successful != 0
	return sess, nil
}
