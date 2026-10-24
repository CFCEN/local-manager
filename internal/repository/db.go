package repository

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dbPath string) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create db directory: %v", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	DB.SetMaxOpenConns(1)

	if err := ensureSchema(); err != nil {
		log.Fatalf("failed to ensure database schema: %v", err)
	}

	if err := runMigrations(); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Println("database initialized successfully")
}

func ensureSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS config_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			group_name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS log_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS log_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			group_id INTEGER,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (group_id) REFERENCES log_groups(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS git_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			rag_path TEXT NOT NULL DEFAULT '',
			index_path TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS git_projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			branch TEXT NOT NULL DEFAULT '',
			remote TEXT NOT NULL DEFAULT '',
			group_id INTEGER,
			rag_path TEXT NOT NULL DEFAULT '',
			index_path TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (group_id) REFERENCES git_groups(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS project_contexts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_id INTEGER NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			rag_path TEXT NOT NULL,
			index_path TEXT NOT NULL,
			index_status TEXT NOT NULL DEFAULT 'NOT_INDEXED',
			last_indexed_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (group_id) REFERENCES git_groups(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS applications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			commands TEXT NOT NULL DEFAULT '[]',
			working_directory TEXT NOT NULL DEFAULT '',
			environment TEXT NOT NULL DEFAULT '{}',
			auto_start INTEGER NOT NULL DEFAULT 0,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS application_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			application_id INTEGER NOT NULL,
			pid INTEGER,
			command_name TEXT NOT NULL DEFAULT '',
			started_at DATETIME,
			stopped_at DATETIME,
			exit_code INTEGER,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_log_files_group_id ON log_files(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_git_projects_group_id ON git_projects(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_project_contexts_group_id ON project_contexts(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_application_runs_application_id ON application_runs(application_id)`,
		`CREATE INDEX IF NOT EXISTS idx_application_runs_status ON application_runs(status)`,
		`CREATE TABLE IF NOT EXISTS automation_adapters (id INTEGER PRIMARY KEY AUTOINCREMENT, adapter_id TEXT NOT NULL UNIQUE, name TEXT NOT NULL DEFAULT '', version TEXT NOT NULL DEFAULT '', online INTEGER NOT NULL DEFAULT 0, last_seen DATETIME NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS automation_projects (id INTEGER PRIMARY KEY AUTOINCREMENT, adapter_id TEXT NOT NULL, project_id TEXT NOT NULL, name TEXT NOT NULL DEFAULT '', path TEXT NOT NULL DEFAULT '', payload TEXT NOT NULL DEFAULT '{}', updated_at DATETIME NOT NULL, UNIQUE(adapter_id, project_id))`,
		`CREATE TABLE IF NOT EXISTS automation_sessions (id INTEGER PRIMARY KEY AUTOINCREMENT, adapter_id TEXT NOT NULL, project_id TEXT NOT NULL DEFAULT '', session_id TEXT NOT NULL, status TEXT NOT NULL DEFAULT '', payload TEXT NOT NULL DEFAULT '{}', cursor TEXT NOT NULL DEFAULT '', updated_at DATETIME NOT NULL, UNIQUE(adapter_id, session_id))`,
		`CREATE TABLE IF NOT EXISTS automation_events (id INTEGER PRIMARY KEY AUTOINCREMENT, adapter_id TEXT NOT NULL, project_id TEXT NOT NULL DEFAULT '', session_id TEXT NOT NULL DEFAULT '', event_id TEXT NOT NULL, type TEXT NOT NULL, payload TEXT NOT NULL DEFAULT '{}', created_at DATETIME NOT NULL, UNIQUE(adapter_id, event_id))`,
		`CREATE TABLE IF NOT EXISTS automation_actions (id INTEGER PRIMARY KEY AUTOINCREMENT, adapter_id TEXT NOT NULL, project_id TEXT NOT NULL DEFAULT '', session_id TEXT NOT NULL DEFAULT '', action_id TEXT NOT NULL, type TEXT NOT NULL, payload TEXT NOT NULL DEFAULT '{}', status TEXT NOT NULL DEFAULT 'pending', result TEXT NOT NULL DEFAULT '{}', created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, UNIQUE(adapter_id, action_id))`,
		`CREATE INDEX IF NOT EXISTS idx_automation_events_session ON automation_events(adapter_id, session_id, id)`,
		`CREATE TABLE IF NOT EXISTS automation_operations (id INTEGER PRIMARY KEY AUTOINCREMENT, request_id TEXT NOT NULL UNIQUE, adapter_id TEXT NOT NULL, session_id TEXT NOT NULL, status TEXT NOT NULL, error TEXT NOT NULL DEFAULT '{}', result TEXT NOT NULL DEFAULT '{}', created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL)`,
	}

	for _, statement := range statements {
		if _, err := DB.Exec(statement); err != nil {
			return err
		}
	}
	var hasCursor bool
	rows, err := DB.Query(`PRAGMA table_info(automation_sessions)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "cursor" {
			hasCursor = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasCursor {
		if _, err := DB.Exec(`ALTER TABLE automation_sessions ADD COLUMN cursor TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	return nil
}

func runMigrations() error {
	dir := "migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Each migration runs in its own statement batch; ignore "duplicate column" errors
		_, err = DB.Exec(string(data))
		if err != nil {
			log.Printf("migration %s: %v (continuing)", entry.Name(), err)
		}
	}

	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
