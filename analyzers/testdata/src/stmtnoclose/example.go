package stmtnoclose

import (
	"context"
	"database/sql"
)

// BadPrepareNoClose leaks a prepared statement.
func BadPrepareNoClose(db *sql.DB) {
	stmt, err := db.Prepare("SELECT 1") // want `db\.Prepare without defer Close leaks prepared statements`
	if err != nil {
		return
	}
	_ = stmt
}

// BadPrepareContextNoClose leaks with context variant.
func BadPrepareContextNoClose(ctx context.Context, db *sql.DB) {
	stmt, err := db.PrepareContext(ctx, "SELECT 1") // want `db\.Prepare without defer Close leaks prepared statements`
	if err != nil {
		return
	}
	_ = stmt
}

// BadTxPrepareNoClose leaks a prepared statement on a transaction.
func BadTxPrepareNoClose(tx *sql.Tx) {
	stmt, err := tx.Prepare("SELECT 1") // want `db\.Prepare without defer Close leaks prepared statements`
	if err != nil {
		return
	}
	_ = stmt
}

// GoodPrepareWithClose has proper cleanup.
func GoodPrepareWithClose(db *sql.DB) {
	stmt, err := db.Prepare("SELECT 1")
	if err != nil {
		return
	}
	defer stmt.Close()
	_ = stmt
}

// GoodPrepareContextWithClose has proper cleanup with context variant.
func GoodPrepareContextWithClose(ctx context.Context, db *sql.DB) {
	stmt, err := db.PrepareContext(ctx, "SELECT 1")
	if err != nil {
		return
	}
	defer stmt.Close()
	_ = stmt
}

// GoodPrepareReturned returns stmt to caller.
func GoodPrepareReturned(db *sql.DB) (*sql.Stmt, error) {
	stmt, err := db.Prepare("SELECT 1")
	if err != nil {
		return nil, err
	}

	return stmt, nil
}

// GoodPrepareClosureDefer uses a closure wrapper for the defer.
func GoodPrepareClosureDefer(db *sql.DB) {
	stmt, err := db.Prepare("SELECT 1")
	if err != nil {
		return
	}
	defer func() {
		stmt.Close() // #nosec G104 -- test fixture, error intentionally ignored
	}()
	_ = stmt
}

type repo struct{ stmt *sql.Stmt }

// GoodPrepareAssignedToField stores the stmt in a struct for later cleanup.
func (r *repo) GoodPrepareAssignedToField(db *sql.DB) {
	stmt, err := db.Prepare("SELECT 1")
	if err != nil {
		return
	}
	r.stmt = stmt
}
