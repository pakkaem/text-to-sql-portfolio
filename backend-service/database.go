package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite" 
)

func initDB() *sql.DB {
	db, err := sql.Open("sqlite", "./hris.db")
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

	-- TABEL BARU: Proyek dan Penugasan
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
		seedData := `
		INSERT INTO departments (name) VALUES ('Engineering'), ('Human Resources'), ('Sales');
		
		INSERT INTO employees (name, department_id, job_title, hire_date) VALUES 
		('Budi Santoso', 1, 'Backend Engineer', '2023-01-15'),
		('Siti Aminah', 1, 'AI Engineer', '2023-06-01'),
		('Andi Wijaya', 2, 'HR Manager', '2022-03-10'),
		('Rina Melati', 3, 'Sales Executive', '2024-02-20'),
		('Tono Mulyadi', 1, 'DevOps Engineer', '2024-01-10');

		INSERT INTO payroll (employee_id, month_year, base_salary, bonus) VALUES 
		(1, '2024-03', 12000000, 1500000), (2, '2024-03', 15000000, 2000000),
		(3, '2024-03', 10000000, 1000000), (4, '2024-03', 8000000, 3000000);

		-- DATA BARU
		INSERT INTO projects (project_name, budget, status) VALUES 
		('Smart City CCTV Analytics', 50000000, 'Ongoing'),
		('MBG Kitchen Hygiene AI', 35000000, 'Ongoing'),
		('HRIS Migration', 15000000, 'Completed');

		INSERT INTO employee_projects (employee_id, project_id, role) VALUES 
		(1, 1, 'Backend API Developer'), (2, 1, 'YOLO Model Trainer'),
		(2, 2, 'Lead AI Engineer'), (3, 3, 'Project Manager');
		`
		_, err = db.Exec(seedData)
		if err != nil {
			log.Fatal("Gagal mengisi data awal: ", err)
		}
		log.Println("Database SQLite berhasil diinisialisasi beserta dummy data!")
	} else {
		log.Println("Database SQLite sudah berisi data, siap digunakan.")
	}

	return db
}