package main

import (
	"database/sql"
	"fmt"
	"log"

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
		seedStatements := []string{
			// Departments
			`INSERT INTO departments (name) VALUES ('Engineering')`,
			`INSERT INTO departments (name) VALUES ('Human Resources')`,
			`INSERT INTO departments (name) VALUES ('Sales')`,

			// Employees
			`INSERT INTO employees (name, department_id, job_title, hire_date) VALUES ('Budi Santoso', 1, 'Backend Engineer', '2023-01-15')`,
			`INSERT INTO employees (name, department_id, job_title, hire_date) VALUES ('Siti Aminah', 1, 'AI Engineer', '2023-06-01')`,
			`INSERT INTO employees (name, department_id, job_title, hire_date) VALUES ('Andi Wijaya', 2, 'HR Manager', '2022-03-10')`,
			`INSERT INTO employees (name, department_id, job_title, hire_date) VALUES ('Rina Melati', 3, 'Sales Executive', '2024-02-20')`,
			`INSERT INTO employees (name, department_id, job_title, hire_date) VALUES ('Tono Mulyadi', 1, 'DevOps Engineer', '2024-01-10')`,

			// Attendance logs
			`INSERT INTO attendance_logs (employee_id, log_date, status) VALUES (1, '2024-03-01', 'Present')`,
			`INSERT INTO attendance_logs (employee_id, log_date, status) VALUES (2, '2024-03-01', 'Present')`,
			`INSERT INTO attendance_logs (employee_id, log_date, status) VALUES (3, '2024-03-01', 'Absent')`,
			`INSERT INTO attendance_logs (employee_id, log_date, status) VALUES (4, '2024-03-01', 'Present')`,
			`INSERT INTO attendance_logs (employee_id, log_date, status) VALUES (5, '2024-03-01', 'Leave')`,
			`INSERT INTO attendance_logs (employee_id, log_date, status) VALUES (1, '2024-03-02', 'Present')`,
			`INSERT INTO attendance_logs (employee_id, log_date, status) VALUES (2, '2024-03-02', 'Leave')`,
			`INSERT INTO attendance_logs (employee_id, log_date, status) VALUES (3, '2024-03-02', 'Present')`,

			// Payroll
			`INSERT INTO payroll (employee_id, month_year, base_salary, bonus) VALUES (1, '2024-03', 12000000, 1500000)`,
			`INSERT INTO payroll (employee_id, month_year, base_salary, bonus) VALUES (2, '2024-03', 15000000, 2000000)`,
			`INSERT INTO payroll (employee_id, month_year, base_salary, bonus) VALUES (3, '2024-03', 10000000, 1000000)`,
			`INSERT INTO payroll (employee_id, month_year, base_salary, bonus) VALUES (4, '2024-03', 8000000, 3000000)`,
			`INSERT INTO payroll (employee_id, month_year, base_salary, bonus) VALUES (5, '2024-03', 11000000, 500000)`,

			// Projects
			`INSERT INTO projects (project_name, budget, status) VALUES ('Smart City CCTV Analytics', 50000000, 'Ongoing')`,
			`INSERT INTO projects (project_name, budget, status) VALUES ('MBG Kitchen Hygiene AI', 35000000, 'Ongoing')`,
			`INSERT INTO projects (project_name, budget, status) VALUES ('HRIS Migration', 15000000, 'Completed')`,

			// Employee projects
			`INSERT INTO employee_projects (employee_id, project_id, role) VALUES (1, 1, 'Backend API Developer')`,
			`INSERT INTO employee_projects (employee_id, project_id, role) VALUES (2, 1, 'YOLO Model Trainer')`,
			`INSERT INTO employee_projects (employee_id, project_id, role) VALUES (2, 2, 'Lead AI Engineer')`,
			`INSERT INTO employee_projects (employee_id, project_id, role) VALUES (3, 3, 'Project Manager')`,
			`INSERT INTO employee_projects (employee_id, project_id, role) VALUES (5, 1, 'Infrastructure Engineer')`,
		}

		for _, stmt := range seedStatements {
			if _, err = db.Exec(stmt); err != nil {
				log.Printf("WARNING: Gagal eksekusi seed statement: %s | Error: %v", stmt, err)
			}
		}
		log.Println("Database SQLite berhasil diinisialisasi beserta data dummy!")
	} else {
		log.Println("Database SQLite sudah berisi data, siap digunakan.")
	}

	return db
}