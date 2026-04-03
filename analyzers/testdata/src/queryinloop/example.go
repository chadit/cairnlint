package queryinloop

import "database/sql"

// BadQueryInLoop demonstrates the N+1 query pattern.
func BadQueryInLoop(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.Query("SELECT * FROM users WHERE id = ?", id) // want `Query called inside loop; this is an N\+1 query pattern, use a batch query instead`
	}
}

// BadQueryRowInLoop uses QueryRow inside a loop.
func BadQueryRowInLoop(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.QueryRow("SELECT name FROM users WHERE id = ?", id) // want `QueryRow called inside loop; this is an N\+1 query pattern, use a batch query instead`
	}
}

// BadExecInLoop uses Exec inside a loop.
func BadExecInLoop(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.Exec("DELETE FROM users WHERE id = ?", id) // want `Exec called inside loop; this is an N\+1 query pattern, use a batch query instead`
	}
}

// GoodQueryOutsideLoop is fine because the query is not in a loop.
func GoodQueryOutsideLoop(db *sql.DB) {
	db.Query("SELECT * FROM users")
}

// GoodBatchQuery is fine because it uses a single query.
func GoodBatchQuery(db *sql.DB) {
	db.Query("SELECT * FROM users WHERE id IN (?, ?, ?)", 1, 2, 3)
}
