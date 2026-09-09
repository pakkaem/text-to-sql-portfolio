-- =====================================================
-- SEED DATA TAMBAHAN HRIS (SENSITIVE - DO NOT COMMIT)
-- Jalankan dengan: sqlite3 hris.db < seed-extra.sql
-- Atau gunakan VS Code SQLite extension
-- =====================================================

-- Department baru
INSERT OR IGNORE INTO departments (id, name) VALUES 
(4, 'Finance'),
(5, 'Marketing'),
(6, 'Product');

-- Employees baru (id 6-10)
INSERT OR IGNORE INTO employees (id, name, department_id, job_title, hire_date) VALUES 
(6, 'Dewi Lestari', 4, 'Finance Analyst', '2023-09-15'),
(7, 'Rizky Pratama', 5, 'Marketing Lead', '2022-11-01'),
(8, 'Aulia Rahma', 6, 'Product Manager', '2024-03-25'),
(9, 'Fajar Nugroho', 1, 'Frontend Engineer', '2024-06-10'),
(10, 'Maya Sari', 2, 'HR Specialist', '2023-12-01');

-- Attendance logs
INSERT INTO attendance_logs (employee_id, log_date, status) VALUES 
(1, '2024-03-01', 'Present'), (2, '2024-03-01', 'Present'),
(3, '2024-03-01', 'Present'), (4, '2024-03-01', 'Absent'),
(5, '2024-03-01', 'Present'), (6, '2024-03-01', 'Leave'),
(7, '2024-03-01', 'Present'), (8, '2024-03-01', 'Present'),
(9, '2024-03-01', 'Present'), (10, '2024-03-01', 'Present'),
(1, '2024-03-02', 'Present'), (2, '2024-03-02', 'Absent'),
(3, '2024-03-02', 'Leave'), (4, '2024-03-02', 'Present'),
(5, '2024-03-02', 'Present'), (6, '2024-03-02', 'Present');

-- Payroll data Maret-April
INSERT INTO payroll (employee_id, month_year, base_salary, bonus) VALUES 
(5, '2024-03', 13000000, 1000000),
(1, '2024-04', 12000000, 500000),
(2, '2024-04', 15000000, 3000000),
(3, '2024-04', 10000000, 1500000),
(4, '2024-04', 8000000, 2000000),
(5, '2024-04', 13000000, 1200000),
(6, '2024-03', 9500000, 800000),
(6, '2024-04', 9500000, 1000000),
(7, '2024-03', 11000000, 1500000),
(7, '2024-04', 11000000, 2000000),
(8, '2024-03', 14000000, 1000000),
(8, '2024-04', 14000000, 2500000),
(9, '2024-03', 10000000, 500000),
(9, '2024-04', 10000000, 800000),
(10, '2024-03', 8500000, 700000),
(10, '2024-04', 8500000, 900000);

-- Projects baru
INSERT OR IGNORE INTO projects (id, project_name, budget, status) VALUES 
(4, 'E-Commerce Platform Revamp', 75000000, 'Ongoing'),
(5, 'Employee Mobile App', 25000000, 'On Hold'),
(6, 'Data Warehouse BI', 40000000, 'Ongoing');

-- Penugasan employee_projects
INSERT OR IGNORE INTO employee_projects (employee_id, project_id, role) VALUES 
(9, 4, 'Frontend Developer'),
(1, 4, 'Backend Lead'),
(8, 4, 'Product Owner'),
(7, 5, 'Marketing Strategist'),
(9, 5, 'Mobile Developer'),
(6, 6, 'BI Analyst'),
(2, 6, 'Data Engineer'),
(10, 3, 'HR Coordinator');