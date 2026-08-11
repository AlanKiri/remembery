package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/alankiri/remembery/internal/paths"
	"github.com/alankiri/remembery/internal/vault"
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
	db    *sql.DB
	vault *vault.Vault
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
		_ = sqldb.Close()
		return nil, err
	}
	return st, nil
}

func (st *DB) VaultExists() (bool, error) {
	var n int
	if err := st.db.QueryRow("SELECT COUNT(*) FROM vault").Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (st *DB) SetVault(v *vault.Vault) {
	st.vault = v
}

func (st *DB) HasVault() bool {
	return st.vault != nil
}

func (st *DB) ChangeVault(newPassword string) (*vault.Vault, error) {
	if st.vault == nil {
		return nil, errors.New("no active vault")
	}
	tx, err := st.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	oldV := st.vault
	rows, err := tx.Query("SELECT id, password FROM trainers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type pair struct {
		id     int64
		cipher string
		plain  string
	}
	var reencrypt []pair
	for rows.Next() {
		var id int64
		var cipher string
		if err := rows.Scan(&id, &cipher); err != nil {
			return nil, err
		}
		plain, err := oldV.Decrypt(cipher)
		if err != nil {
			return nil, err
		}
		reencrypt = append(reencrypt, pair{id: id, plain: plain})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if _, err := tx.Exec("DELETE FROM vault"); err != nil {
		return nil, err
	}
	newV, err := createVaultForTx(tx, newPassword)
	if err != nil {
		return nil, err
	}
	for _, p := range reencrypt {
		cipher, err := newV.Encrypt(p.plain)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec("UPDATE trainers SET password = ? WHERE id = ?", cipher, p.id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	st.vault = newV
	return newV, nil
}

func createVaultForTx(tx *sql.Tx, password string) (*vault.Vault, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key, err := vault.DeriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	v := vault.New(key)
	canary, err := v.Encrypt(vault.Canary)
	if err != nil {
		return nil, err
	}
	saltB64 := base64.StdEncoding.EncodeToString(salt)
	if _, err := tx.Exec("INSERT INTO vault (salt, canary) VALUES (?, ?)", saltB64, canary); err != nil {
		return nil, err
	}
	return v, nil
}

func (st *DB) DecryptVault() error {
	if st.vault == nil {
		return errors.New("no active vault")
	}
	tx, err := st.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query("SELECT id, password FROM trainers")
	if err != nil {
		return err
	}
	defer rows.Close()

	type pair struct {
		id    int64
		plain string
	}
	var updates []pair
	for rows.Next() {
		var id int64
		var cipher string
		if err := rows.Scan(&id, &cipher); err != nil {
			return err
		}
		plain, err := st.vault.Decrypt(cipher)
		if err != nil {
			return err
		}
		updates = append(updates, pair{id, plain})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, u := range updates {
		if _, err := tx.Exec("UPDATE trainers SET password = ? WHERE id = ?", u.plain, u.id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("DELETE FROM vault"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	st.vault = nil
	return nil
}

func (st *DB) CreateVaultAndEncrypt(password string) (*vault.Vault, error) {
	tx, err := st.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	v, err := createVaultForTx(tx, password)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query("SELECT id, password FROM trainers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type pair struct {
		id     int64
		cipher string
	}
	var updates []pair
	for rows.Next() {
		var id int64
		var plain string
		if err := rows.Scan(&id, &plain); err != nil {
			return nil, err
		}
		cipher, err := v.Encrypt(plain)
		if err != nil {
			return nil, err
		}
		updates = append(updates, pair{id, cipher})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, u := range updates {
		if _, err := tx.Exec("UPDATE trainers SET password = ? WHERE id = ?", u.cipher, u.id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	st.vault = v
	return v, nil
}

func (st *DB) CreateVault(password string) (*vault.Vault, error) {
	tx, err := st.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	v, err := createVaultForTx(tx, password)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	st.vault = v
	return v, nil
}

func (st *DB) LoadVault(password string) (*vault.Vault, error) {
	var saltB64, canary string
	if err := st.db.QueryRow("SELECT salt, canary FROM vault LIMIT 1").Scan(&saltB64, &canary); err != nil {
		return nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, err
	}
	key, err := vault.DeriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	v := vault.New(key)
	if !v.Verify(canary) {
		return nil, errors.New("incorrect master password")
	}
	st.vault = v
	return v, nil
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

    CREATE TABLE IF NOT EXISTS vault (
        salt TEXT NOT NULL,
        canary TEXT NOT NULL
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

func (st *DB) Reset() error {
	if err := st.Close(); err != nil {
		return err
	}
	return os.Remove(paths.DBFile())
}

func (st *DB) CreateTrainer(label, password string, level int) (int64, error) {
	if st.vault != nil {
		cipher, err := st.vault.Encrypt(password)
		if err != nil {
			return 0, err
		}
		password = cipher
	}
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
	return st.scanTrainer(row)
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
		t, err := st.scanTrainer(rows)
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
	password := t.Password
	if st.vault != nil {
		cipher, err := st.vault.Encrypt(t.Password)
		if err != nil {
			return err
		}
		password = cipher
	}
	var lastCounted sql.NullInt64
	if t.LastCountedSession != nil {
		lastCounted = sql.NullInt64{Int64: t.LastCountedSession.Unix(), Valid: true}
	}
	_, err := st.db.Exec(`
		UPDATE trainers SET label = ?, password = ?, level = ?, sessions_at_level = ?,
		total_sessions = ?, last_counted_session = ?, last_reset_date = ? WHERE id = ?`,
		t.Label, password, t.Level, t.SessionsAtLevel, t.TotalSessions, lastCounted, t.LastResetDate.Unix(), t.ID)
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

func (st *DB) scanTrainer(s scanner) (Trainer, error) {
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
	if st.vault != nil {
		plain, err := st.vault.Decrypt(t.Password)
		if err != nil {
			return t, err
		}
		t.Password = plain
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
