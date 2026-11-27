package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Initialize the single household row once. A previous JSON installation is read
// only during creation. Restarting after an import never overwrites database state.
func postgresHandler(ctx context.Context, db *sql.DB, path string, secure bool, scanner *LibraryScanner) (http.Handler, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(68317421)"); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS familytv_household (
 singleton boolean PRIMARY KEY CHECK (singleton), version integer NOT NULL, document jsonb NOT NULL
 )`); err != nil {
		return nil, err
	}
	var version int
	var document []byte
	err = tx.QueryRowContext(ctx, "SELECT version, document FROM familytv_household WHERE singleton FOR UPDATE").Scan(&version, &document)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		legacy, err := New(path, secure)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(legacy.data)
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO familytv_household VALUES (true,2,$1)", string(b)); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	case version == 1:
		migrated, err := retireChannels(document)
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE familytv_household SET version=2,document=$1 WHERE singleton", string(migrated)); err != nil {
			return nil, err
		}
	case version != 2:
		return nil, errors.New("unsupported household schema version")
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	public := (&Server{}).Routes()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/health" {
			public.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		readOnly := r.Method == http.MethodGet || r.Method == http.MethodHead
		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
		if err != nil {
			write(w, 503, map[string]string{"error": "storage unavailable"})
			return
		}
		defer tx.Rollback()
		query := "SELECT document FROM familytv_household WHERE singleton AND version=2"
		if !readOnly {
			query += " FOR UPDATE"
		}
		var b []byte
		s := &Server{secureCookies: secure, scanner: scanner}
		s.currentState = func(ctx context.Context) (state, error) {
			var current state
			var snapshot []byte
			e := db.QueryRowContext(ctx, "SELECT document FROM familytv_household WHERE singleton AND version=2").Scan(&snapshot)
			if e == nil {
				e = json.Unmarshal(snapshot, &current)
			}
			return current, e
		}
		if err = tx.QueryRowContext(ctx, query).Scan(&b); err != nil || json.Unmarshal(b, &s.data) != nil {
			write(w, 503, map[string]string{"error": "storage unavailable"})
			return
		}
		if readOnly {
			if err = tx.Commit(); err != nil {
				write(w, 503, map[string]string{"error": "storage unavailable"})
				return
			}
		}
		if s.data.Sessions == nil {
			s.data.Sessions = map[string]int64{}
		}
		s.save = func(state state) error {
			b, err := json.Marshal(state)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, "UPDATE familytv_household SET document=$1 WHERE singleton AND version=2", string(b)); err != nil {
				return err
			}
			return tx.Commit()
		}
		s.Routes().ServeHTTP(w, r)
	}), nil
}

func OpenPostgres(ctx context.Context, dsn, path string, secure bool) (http.Handler, func() error, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(12)
	scanCtx, cancel := context.WithCancel(context.Background())
	scanner := &LibraryScanner{ctx: scanCtx}
	scanner.known = func(ctx context.Context) (map[string]bool, string, error) {
		var raw []byte
		var folder string
		err := db.QueryRowContext(ctx, "SELECT COALESCE(document->'media','[]'::jsonb), COALESCE(document->>'library_folder','') FROM familytv_household WHERE singleton AND version=2").Scan(&raw, &folder)
		if err != nil {
			return nil, "", err
		}
		var items []Media
		if err = json.Unmarshal(raw, &items); err != nil {
			return nil, "", err
		}
		return knownPaths(items), folder, nil
	}
	scanner.insert = func(ctx context.Context, m Media, existing bool) (int, error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return 500, err
		}
		defer tx.Rollback()
		var raw []byte
		if err = tx.QueryRowContext(ctx, "SELECT document FROM familytv_household WHERE singleton AND version=2 FOR UPDATE").Scan(&raw); err != nil {
			return 500, err
		}
		s := &Server{}
		if err = json.Unmarshal(raw, &s.data); err != nil {
			return 500, err
		}
		s.save = func(st state) error {
			b, e := json.Marshal(st)
			if e != nil {
				return e
			}
			if _, e = tx.ExecContext(ctx, "UPDATE familytv_household SET document=$1 WHERE singleton AND version=2", string(b)); e != nil {
				return e
			}
			return tx.Commit()
		}
		return s.storeScanItem(m, existing)
	}
	handler, err := postgresHandler(ctx, db, path, secure, scanner)
	if err != nil {
		cancel()
		db.Close()
		return nil, nil, err
	}
	go scanner.schedule()
	return handler, func() error { cancel(); return db.Close() }, nil
}
