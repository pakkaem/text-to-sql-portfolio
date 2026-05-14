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
		('Rina Melati', 3, 'Sales Executive', '2024-02-20');

		INSERT INTO attendance_logs (employee_id, log_date, status) VALUES 
		(1, '2024-03-01', 'Present'), (2, '2024-03-01', 'Present'),
		(3, '2024-03-01', 'Leave'), (4, '2024-03-01', 'Present');

		INSERT INTO payroll (employee_id, month_year, base_salary, bonus) VALUES 
		(1, '2024-03', 12000000, 1500000),
		(2, '2024-03', 15000000, 2000000);
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