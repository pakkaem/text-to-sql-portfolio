-- =====================================================
-- SEED DATA AWAL HRIS (SENSITIVE - DO NOT COMMIT)
-- File ini dibaca otomatis oleh database.go saat tabel masih kosong
-- =====================================================

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

INSERT INTO projects (project_name, budget, status) VALUES 
('Smart City CCTV Analytics', 50000000, 'Ongoing'),
('MBG Kitchen Hygiene AI', 35000000, 'Ongoing'),
('HRIS Migration', 15000000, 'Completed');

INSERT INTO employee_projects (employee_id, project_id, role) VALUES 
(1, 1, 'Backend API Developer'), (2, 1, 'YOLO Model Trainer'),
(2, 2, 'Lead AI Engineer'), (3, 3, 'Project Manager');