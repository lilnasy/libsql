package libsql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
)

func runMemoryAndFileTests(t *testing.T, test func(*testing.T, *sql.DB)) {
	t.Parallel()
	t.Run("Memory", func(t *testing.T) {
		t.Parallel()
		db, err := sql.Open("libsql", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
		}()
		test(t, db)
	})
	t.Run("File", func(t *testing.T) {
		t.Parallel()
		dir, err := os.MkdirTemp("", "libsql-*")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)
		db, err := sql.Open("libsql", dir+"/test.db")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
		}()
		test(t, db)
	})
}

func TestErrorNonUtf8URL(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("libsql", "a\xc5z")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	conn, err := db.Conn(context.Background())
	if err == nil {
		defer func() {
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}
		}()
		t.Fatal("expected error")
	}
	if err.Error() != "failed to open database a\xc5z\nerror code = 1: Wrong URL: invalid utf-8 sequence of 1 bytes from index 1" {
		t.Fatal("unexpected error:", err)
	}
}

func TestErrorWrongURL(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("libsql", "http://example.com/test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	conn, err := db.Conn(context.Background())
	if err == nil {
		defer func() {
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}
		}()
		t.Fatal("expected error")
	}
	if err.Error() != "failed to open database http://example.com/test\nerror code = 1: Error opening URL http://example.com/test: Failed to connect to database: `Unable to open remote database http://example.com/test with Database::open()`" {
		t.Fatal("unexpected error:", err)
	}
}

func TestExec(t *testing.T) {
	runMemoryAndFileTests(t, func(t *testing.T, db *sql.DB) {
		if _, err := db.ExecContext(context.Background(), "CREATE TABLE test (id INTEGER, name TEXT)"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestExecWithQuery(t *testing.T) {
	runMemoryAndFileTests(t, func(t *testing.T, db *sql.DB) {
		if _, err := db.ExecContext(context.Background(), "SELECT 1"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestQuery(t *testing.T) {
	runMemoryAndFileTests(t, func(t *testing.T, db *sql.DB) {
		if _, err := db.ExecContext(context.Background(), "CREATE TABLE test (id INTEGER, name TEXT, gpa REAL, cv BLOB)"); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 10; i++ {
			if _, err := db.ExecContext(context.Background(), "INSERT INTO test VALUES("+fmt.Sprint(i)+", '"+fmt.Sprint(i)+"', "+fmt.Sprint(i)+".5, randomblob(10))"); err != nil {
				t.Fatal(err)
			}
		}
		rows, err := db.QueryContext(context.Background(), "SELECT NULL, id, name, gpa, cv FROM test")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		idx := 0
		for rows.Next() {
			var null any
			var id int
			var name string
			var gpa float64
			var cv []byte
			if err := rows.Scan(&null, &id, &name, &gpa, &cv); err != nil {
				t.Fatal(err)
			}
			if null != nil {
				t.Fatal("null should be nil")
			}
			if id != int(idx) {
				t.Fatal("id should be", idx)
			}
			if name != fmt.Sprint(idx) {
				t.Fatal("name should be", idx)
			}
			if gpa != float64(idx)+0.5 {
				t.Fatal("gpa should be", float64(idx)+0.5)
			}
			if len(cv) != 10 {
				t.Fatal("cv should be 10 bytes")
			}
			idx++
		}
	})
}
