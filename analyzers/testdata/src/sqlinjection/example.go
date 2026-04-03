package sqlinjection

import "fmt"

// Bad: SQL keywords in fmt.Sprintf format string.
func buildSelectQuery(table string) string {
	return fmt.Sprintf("SELECT * FROM %s WHERE id = 1", table) // want `SQL string formatting detected`
}

// Bad: INSERT query.
func buildInsertQuery(name string) string {
	return fmt.Sprintf("INSERT INTO users (name) VALUES ('%s')", name) // want `SQL string formatting detected`
}

// Bad: DELETE query.
func buildDeleteQuery(userID string) string {
	return fmt.Sprintf("DELETE FROM users WHERE id = %s", userID) // want `SQL string formatting detected`
}

// Good: no SQL keywords.
func buildGreeting(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

// Good: not fmt.Sprintf.
func notSprintf() string {
	return "SELECT * FROM users"
}
