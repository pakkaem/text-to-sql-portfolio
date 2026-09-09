package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

// Fallback schema untuk HRIS domain
const hrisSchemaFallback = `
CREATE TABLE departments (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL);
CREATE TABLE employees (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, department_id INTEGER REFERENCES departments(id), job_title TEXT, hire_date TEXT NOT NULL);
CREATE TABLE attendance_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, employee_id INTEGER REFERENCES employees(id), log_date TEXT NOT NULL, status TEXT CHECK (status IN ('Present', 'Absent', 'Leave')));
CREATE TABLE payroll (id INTEGER PRIMARY KEY AUTOINCREMENT, employee_id INTEGER REFERENCES employees(id), month_year TEXT NOT NULL, base_salary REAL NOT NULL, bonus REAL DEFAULT 0);
CREATE TABLE projects (id INTEGER PRIMARY KEY AUTOINCREMENT, project_name TEXT NOT NULL, budget REAL NOT NULL, status TEXT CHECK (status IN ('Ongoing', 'Completed', 'On Hold')));
CREATE TABLE employee_projects (employee_id INTEGER REFERENCES employees(id), project_id INTEGER REFERENCES projects(id), role TEXT NOT NULL, PRIMARY KEY (employee_id, project_id));
`

// DomainInfo menyimpan metadata tentang satu domain beserta koneksi DB-nya
type DomainInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	TableCount  int      `json:"table_count"`
	DB          *sql.DB  `json:"-"`
	Schema      string   `json:"-"`
	Fallback    string   `json:"-"`
}

// DomainRegistry menyimpan semua domain yang terdaftar
var domainRegistry map[string]*DomainInfo

func initAllDatabases() map[string]*DomainInfo {
	domainRegistry = make(map[string]*DomainInfo)

	// Init HRIS domain
	hrisDB := initHRISDB()
	hrisSchema := getSchemaFromDB(hrisDB)
	if hrisSchema == "" {
		hrisSchema = hrisSchemaFallback
	}
	tableCount := getTableCount(hrisDB)
	domainRegistry["hris"] = &DomainInfo{
		Name:        "hris",
		DisplayName: "🏢 HRIS",
		Description: "Human Resource Information System — Departments, Employees, Payroll, Attendance, Projects",
		TableCount:  tableCount,
		DB:          hrisDB,
		Schema:      hrisSchema,
		Fallback:    hrisSchemaFallback,
	}

	// Init Smart City domain
	scDB := initSmartCityDB()
	scSchema := getSchemaFromDB(scDB)
	if scSchema == "" {
		scSchema = smartCitySchemaFallback
	}
	scTableCount := getTableCount(scDB)
	domainRegistry["smartcity"] = &DomainInfo{
		Name:        "smartcity",
		DisplayName: "🏙️ Smart City",
		Description: "Smart City Traffic & Infrastructure — Districts, Roads, Cameras, Traffic Events, Violations, Incidents",
		TableCount:  scTableCount,
		DB:          scDB,
		Schema:      scSchema,
		Fallback:    smartCitySchemaFallback,
	}

	log.Printf("Berhasil menginisialisasi %d domain", len(domainRegistry))
	return domainRegistry
}

func getTableCount(db *sql.DB) int {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&count)
	return count
}

func getDomainDB(domain string) (*DomainInfo, error) {
	info, ok := domainRegistry[domain]
	if !ok {
		available := make([]string, 0, len(domainRegistry))
		for k := range domainRegistry {
			available = append(available, k)
		}
		return nil, fmt.Errorf("domain '%s' tidak tersedia. Domain yang tersedia: %v", domain, available)
	}
	return info, nil
}

func executeSeedFile(db *sql.DB, filename string) error {
    seedBytes, err := os.ReadFile(filename)
    if err != nil {
        return fmt.Errorf("file tidak ditemukan atau tidak bisa dibaca: %w", err)
    }

    seedSQL := string(seedBytes)

    // Hapus komentar dan baris kosong
    var cleanLines []string

    for _, line := range strings.Split(seedSQL, "\n") {
        line = strings.TrimSpace(line)

        if line == "" || strings.HasPrefix(line, "--") {
            continue
        }

        cleanLines = append(cleanLines, line)
    }

    cleanSQL := strings.Join(cleanLines, "\n")

    statements := strings.Split(cleanSQL, ";")

    executed := 0

    for _, stmt := range statements {
        stmt = strings.TrimSpace(stmt)

        if stmt == "" {
            continue
        }

        if _, err := db.Exec(stmt); err != nil {
            return fmt.Errorf("gagal eksekusi statement: %w", err)
        }

        executed++
    }

    log.Printf("Seed %s: %d statement berhasil dieksekusi", filename, executed)

    return nil
}

func initHRISDB() *sql.DB {
	dbName := getEnv("HRIS_DB_PATH", getEnv("DB_NAME", "./hris.db"))
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
    if err := executeSeedFile(db, "seed-data.sql"); err != nil {
        log.Printf("WARNING: Gagal menjalankan seed-data.sql: %v", err)
    } else {
        log.Println("Database SQLite berhasil diinisialisasi dari seed-data.sql!")
    }
}

// Jalankan extra seed hanya jika employee tambahan belum ada.
var employeeCount int
db.QueryRow("SELECT COUNT(*) FROM employees").Scan(&employeeCount)

if employeeCount < 10 {
    if err := executeSeedFile(db, "seed-extra.sql"); err != nil {
        log.Printf("WARNING: Gagal menjalankan seed-extra.sql: %v", err)
    } else {
        log.Println("Extra seed berhasil dijalankan dari seed-extra.sql!")
    }
} else {
    log.Printf("Extra seed dilewati: database sudah memiliki %d employees.", employeeCount)
}

	return db
}
