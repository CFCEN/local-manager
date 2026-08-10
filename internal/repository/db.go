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
	}

	for _, statement := range statements {
		if _, err := DB.Exec(statement); err != nil {
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
