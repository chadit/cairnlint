package queryinloop

import (
	"context"
	"database/sql"
	"fmt"
)

// BadQueryInLoop demonstrates the N+1 query pattern.
func BadQueryInLoop(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.Query("SELECT * FROM users WHERE id = ?", id) // #nosec G104 -- test fixture // want `Query called inside loop; this is an N\+1 query pattern, use a batch query instead`
	}
}

// BadQueryRowInLoop uses QueryRow inside a loop.
func BadQueryRowInLoop(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.QueryRow("SELECT name FROM users WHERE id = ?", id) // #nosec G104 -- test fixture // want `QueryRow called inside loop; this is an N\+1 query pattern, use a batch query instead`
	}
}

// BadExecInLoop uses Exec inside a loop.
func BadExecInLoop(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.Exec("DELETE FROM users WHERE id = ?", id) // #nosec G104 -- test fixture // want `Exec called inside loop; this is an N\+1 query pattern, use a batch query instead`
	}
}

// GoodQueryOutsideLoop is fine because the query is not in a loop.
func GoodQueryOutsideLoop(db *sql.DB) {
	db.Query("SELECT * FROM users") // #nosec G104 -- test fixture
}

// GoodBatchQuery is fine because it uses a single query.
func GoodBatchQuery(db *sql.DB) {
	db.Query("SELECT * FROM users WHERE id IN (?, ?, ?)", 1, 2, 3) // #nosec G104 -- test fixture
}

// GoodDynamicExecInLoop is fine because each iteration builds a structurally
// different query via fmt.Sprintf (e.g., TRUNCATE on different tables).
func GoodDynamicExecInLoop(db *sql.DB, tables []string) {
	for _, table := range tables {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)) // #nosec G104 -- test fixture
	}
}

// GoodDynamicExecContextInLoop is fine because the Context variant also uses
// fmt.Sprintf to build a different query per iteration.
func GoodDynamicExecContextInLoop(db *sql.DB, tables []string) {
	for _, table := range tables {
		db.ExecContext(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)) // #nosec G104 -- test fixture
	}
}

// BadQueryContextInLoop flags because the query is a static parameterized
// string repeated with different bind values per iteration.
func BadQueryContextInLoop(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.QueryContext(context.Background(), "SELECT * FROM users WHERE id = ?", id) // #nosec G104 -- test fixture // want `QueryContext called inside loop; this is an N\+1 query pattern, use a batch query instead`
	}
}
