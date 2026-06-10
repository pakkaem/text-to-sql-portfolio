package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

func initDB() *sql.DB {
	dbName := getEnv("DB_NAME", "./hris.db")
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		log.Fatal(err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS departments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS employees (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		department_id INTEGER,
		job_title TEXT,
		hire_date TEXT NOT NULL,
		FOREIGN KEY(department_id) REFERENCES departments(id)
	);

	CREATE TABLE IF NOT EXISTS attendance_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		employee_id INTEGER,
		log_date TEXT NOT NULL,
		status TEXT CHECK (status IN ('Present', 'Absent', 'Leave')),
		FOREIGN KEY(employee_id) REFERENCES employees(id)
	);

	CREATE TABLE IF NOT EXISTS payroll (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		employee_id INTEGER,
		month_year TEXT NOT NULL,
		base_salary REAL NOT NULL,
		bonus REAL DEFAULT 0,
		FOREIGN KEY(employee_id) REFERENCES employees(id)
	);

	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_name TEXT NOT NULL,
		budget REAL NOT NULL,
		status TEXT CHECK (status IN ('Ongoing', 'Completed', 'On Hold'))
	);

	CREATE TABLE IF NOT EXISTS employee_projects (
		employee_id INTEGER,
		project_id INTEGER,
		role TEXT NOT NULL,
		FOREIGN KEY(employee_id) REFERENCES employees(id),
		FOREIGN KEY(project_id) REFERENCES projects(id),
		PRIMARY KEY (employee_id, project_id)
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal("Gagal membuat skema: ", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM departments").Scan(&count)

	if count == 0 {
		// Baca seed data dari file eksternal (gitignored)
		seedFile := "seed-data.sql"
		seedBytes, err := os.ReadFile(seedFile)
		if err != nil {
			log.Printf("WARNING: File '%s' tidak ditemukan atau tidak bisa dibaca: %v", seedFile, err)
			log.Println("Database kosong! Silakan buat file 'seed-data.sql' atau jalankan seed secara manual.")
			return db
		}

		seedSQL := string(seedBytes)
		// Filter komentar dan baris kosong
		var statements []string
		for _, stmt := range strings.Split(seedSQL, ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}
			statements = append(statements, stmt)
		}

		for _, stmt := range statements {
			_, err = db.Exec(stmt)
			if err != nil {
				log.Printf("WARNING: Gagal eksekusi seed statement: %s | Error: %v", stmt, err)
			}
		}
		log.Println("Database SQLite berhasil diinisialisasi beserta data dari seed-data.sql!")
	} else {
		log.Println("Database SQLite sudah berisi data, siap digunakan.")
	}

	return db
}