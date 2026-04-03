package dbquerywithbarebackground

import (
	"context"
	"database/sql"
)

// BadQueryContextWithBackground passes a bare context.Background().
func BadQueryContextWithBackground(db *sql.DB) {
	db.QueryContext(context.Background(), "SELECT 1") // want `QueryContext called with context\.Background\(\); pass a request-scoped context instead`
}

// BadExecContextWithBackground passes a bare context.Background().
func BadExecContextWithBackground(db *sql.DB) {
	db.ExecContext(context.Background(), "DELETE FROM old_rows") // want `ExecContext called with context\.Background\(\); pass a request-scoped context instead`
}

// BadQueryRowContextWithBackground passes a bare context.Background().
func BadQueryRowContextWithBackground(db *sql.DB) {
	db.QueryRowContext(context.Background(), "SELECT count(*) FROM users") // want `QueryRowContext called with context\.Background\(\); pass a request-scoped context instead`
}

// GoodQueryContextWithRealCtx passes a request-scoped context.
func GoodQueryContextWithRealCtx(ctx context.Context, db *sql.DB) {
	db.QueryContext(ctx, "SELECT 1")
}

// GoodExecContextWithRealCtx passes a request-scoped context.
func GoodExecContextWithRealCtx(ctx context.Context, db *sql.DB) {
	db.ExecContext(ctx, "DELETE FROM old_rows")
}

// GoodPlainQuery uses the non-context variant, which this analyzer ignores.
func GoodPlainQuery(db *sql.DB) {
	db.Query("SELECT 1")
}
