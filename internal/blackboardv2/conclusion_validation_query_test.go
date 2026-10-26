package blackboardv2

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"modernc.org/sqlite"
	"pentest/internal/store"
)

var conclusionQueryDriverSequence atomic.Uint64

type conclusionQuery struct {
	sql  string
	args []any
}
type conclusionQueryRecorder struct {
	mu      sync.Mutex
	queries []conclusionQuery
}
type conclusionRecordingDriver struct {
	inner    driver.Driver
	recorder *conclusionQueryRecorder
}
type conclusionRecordingConn struct {
	driver.Conn
	recorder *conclusionQueryRecorder
}

func (d conclusionRecordingDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.inner.Open(name)
	if err != nil {
		return nil, err
	}
	return conclusionRecordingConn{Conn: conn, recorder: d.recorder}, nil
}

func (c conclusionRecordingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	copied := make([]any, len(args))
	for i, arg := range args {
		copied[i] = arg.Value
	}
	c.recorder.mu.Lock()
	c.recorder.queries = append(c.recorder.queries, conclusionQuery{sql: query, args: copied})
	c.recorder.mu.Unlock()
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
}

// Observe SQL sent by the actual validators. Explaining a separate test query
// would not detect a future validator that reads and filters a whole Project.
func TestActualConclusionValidationQueriesProbeAffectedPrimaryKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queries.db")
	baseline, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := baseline.Close(); err != nil {
		t.Fatal(err)
	}
	recorder := &conclusionQueryRecorder{}
	name := fmt.Sprintf("conclusion-query-%d", conclusionQueryDriverSequence.Add(1))
	sql.Register(name, conclusionRecordingDriver{inner: &sqlite.Driver{}, recorder: recorder})
	db, err := sql.Open(name, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, validate := range []func(context.Context, *sql.Tx, string, []string) error{validateConfirmedFindings, validateVerifiedSolutions} {
		if err := validate(context.Background(), tx, "project:selected", []string{"finding:selected", "solution:selected"}); err != nil {
			t.Fatal(err)
		}
	}
	recorder.mu.Lock()
	observed := append([]conclusionQuery(nil), recorder.queries...)
	recorder.mu.Unlock()
	checked := 0
	for _, query := range observed {
		if !strings.Contains(query.sql, "blackboard_v2_records") || !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query.sql)), "SELECT") {
			continue
		}
		checked++
		rows, err := tx.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+query.sql, query.args...)
		if err != nil {
			t.Fatal(err)
		}
		indexed := false
		details := []string{}
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			details = append(details, detail)
			indexed = indexed || strings.Contains(detail, "project_id=? AND key=?")
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		rows.Close()
		if !indexed {
			t.Fatalf("actual conclusion query scans unrelated records: %s; plan=%v", query.sql, details)
		}
	}
	if checked != 2 {
		t.Fatalf("observed %d real conclusion queries, want 2", checked)
	}
}
