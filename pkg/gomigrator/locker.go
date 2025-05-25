package gomigrator

import "database/sql"

type Locker struct {
	db *sql.DB
}

func NewLocker(db *sql.DB) *Locker {
	return &Locker{
		db: db,
	}
}

func (l *Locker) Lock() error {
	_, err := l.db.Exec("SELECT pg_advisory_lock($1)", 1234)
	if err != nil {
		return err
	}

	return nil
}

func (l *Locker) Unlock() {
	l.db.Exec("SELECT pg_advisory_unlock($1)", 1234)
}
