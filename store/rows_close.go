package store

import "database/sql"

// closeRows closes a read-only result set. Iteration failures are surfaced
// through rows.Err() at the call site; a Close error on a query result carries
// no recovery action, so it is deliberately not propagated.
func closeRows(rows *sql.Rows) { // #nosec G104 -- read-only result set, see doc comment
	if rows == nil {
		return
	}
	_ = rows.Close()
}
