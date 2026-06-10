# 🧠 Text-to-SQL AI Assistant (HRIS Domain)

Proyek ini adalah implementasi *End-to-End* dari arsitektur AI Text-to-SQL. Sistem ini memungkinkan pengguna untuk mengekstrak data dari database relasional (HRIS) menggunakan pertanyaan bahasa natural (Bahasa Inggris).

Sistem ini menunjukkan pemisahan beban kerja (*separation of concerns*) antara layanan Inferensi AI dan layanan Backend utama.

---

## 🏗️ Arsitektur Sistem

Proyek ini mengadopsi arsitektur *microservices* sederhana:

| Layer | Teknologi | Deskripsi |
|-------|-----------|-----------|
| **Layanan AI** | Python / FastAPI | Bertindak sebagai "otak" sistem. Mendukung **3 mode inference**: **Groq API** (default, cloud), **Zero-Shot** (model lokal), dan **LoRA Fine-tuned** (model lokal + adapter). Mode dipilih via environment variable tanpa ubah kode. |
| **Layanan Backend** | Golang / Gin | Bertindak sebagai "jembatan" utama. Menerima request HTTP dari pengguna, membaca **DDL schema secara dinamis** dari sqlite_master, meminta query SQL dari layanan AI, melakukan validasi keamanan, dan mengeksekusi query ke database. Dilengkapi **retry logic** otomatis jika query gagal. |
| **Frontend** | SvelteKit 2 / Svelte 5 | UI interaktif dengan input pertanyaan dan tampilan hasil dalam bentuk tabel HTML dinamis. |
| **Database** | SQLite | Penyimpanan data terintegrasi (No-Ops) yang dibuat otomatis beserta *dummy data* saat aplikasi pertama kali dijalankan. |

### Alur Data

```
User → Frontend (SvelteKit :5173) → Backend (Go/Gin :8080) → AI Service (FastAPI :8000)
                                                                      │
                                                      ┌───────────────┼───────────────┐
                                                      ▼               ▼               ▼
                                                  Groq API      Zero-Shot         LoRA
                                                  (cloud)       (lokal)         (lokal)
                                                      │               │               │
                                                      └───────────────┼───────────────┘
                                                                      ▼
                                                                SQL Query
                                                                      │
                                                                      ▼
                                                            Backend executes → SQLite
                                                                      │
                                                                      ▼
                                                            Response → User (tabel)
```

---

## 🛠️ AI Model Modes

| Mode | Env Var | Model | RAM | Kecepatan | Kebutuhan |
|------|---------|-------|-----|-----------|-----------|
| **Groq** (default) | `AI_MODEL_MODE=groq` | Llama 3.1 8B (cloud) | 0 GB | ~1-2 detik | `GROQ_API_KEY` |
| **Zero-Shot** | `AI_MODEL_MODE=zero-shot` | Qwen2.5-1.5B-Instruct | ~3 GB | 30-60 detik | `torch`, `transformers` |
| **LoRA** | `AI_MODEL_MODE=lora` | TinyLlama 1.1B + adapter | ~2 GB | 10-30 detik | `torch`, `transformers`, `peft` + checkpoint |

---

## 🚀 Cara Menjalankan Aplikasi Secara Lokal

### Prasyarat

- **Python 3.9+**
- **Golang** (dengan CGO/MinGW untuk go-sqlite3)
- **Node.js 18+** (untuk frontend)
- **Groq API Key** (gratis dari [console.groq.com](https://console.groq.com)) — untuk mode default

### 1. Menjalankan Layanan AI (Port 8000)

```bash
cd ai-service
python -m venv venv
# Aktifkan venv
# Windows:   venv\Scripts\activate
# Mac/Linux: source venv/bin/activate
pip install -r requirements.txt
```

**Setup `.env`:**

```bash
# Copy template
cp .env.example .env
```

Edit `.env` dan isi API key Groq kamu:

```env
AI_MODEL_MODE=groq
GROQ_API_KEY=gsk_your_api_key_here
GROQ_MODEL=llama-3.1-8b-instant
```

**Jalankan:**

```bash
python app.py
```

> **Catatan:** Mode default (Groq) **tidak perlu download model** — langsung siap pakai. Untuk mode lokal (`zero-shot`/`lora`), uncomment dependencies di `requirements.txt` dan install ulang.

### 2. Menjalankan Layanan Backend (Port 8080)

```bash
cd backend-service
go run .
```

> Backend akan secara otomatis membuat file `hris.db` dan mengisinya dengan data HRIS dummy. Schema database dibaca **secara dinamis** dari `sqlite_master`.

### 3. Menjalankan Frontend (Port 5173)

```bash
cd frontend-service
npm install
npm run dev
```

---

## ⚙️ Environment Variables

### AI Service (`ai-service/.env`)

| Variabel | Default | Keterangan |
|----------|---------|------------|
| `AI_MODEL_MODE` | `groq` | Mode AI: `groq`, `zero-shot`, atau `lora` |
| `GROQ_API_KEY` | *(wajib)* | API key dari [console.groq.com](https://console.groq.com) |
| `GROQ_MODEL` | `llama-3.1-8b-instant` | Model Groq yang digunakan |
| `ZEROSHOT_MODEL` | `Qwen/Qwen2.5-1.5B-Instruct` | HuggingFace model ID (mode zero-shot) |
| `LORA_BASE_MODEL` | `TinyLlama/TinyLlama-1.1B-Chat-v1.0` | Base model (mode lora) |
| `LORA_ADAPTER_PATH` | `./checkpoint-1950` | Path ke LoRA adapter (mode lora) |

### Backend Service

| Variabel | Default | Keterangan |
|----------|---------|------------|
| `AI_SERVICE_URL` | `http://127.0.0.1:8000` | URL AI service |
| `DB_PATH` | `./hris.db` | Path ke file SQLite |
| `BACKEND_PORT` | `8080` | Port backend service |
| `CORS_ORIGINS` | `*` | Allowed CORS origins (comma-separated) |

---

## 🛡️ Fitur Keamanan & Robustness

| Fitur | Lokasi | Fungsi |
|-------|--------|--------|
| **Dynamic Schema** | `main.go` | Membaca DDL aktual dari `sqlite_master` — otomatis akurat walau struktur tabel berubah |
| **Prompt Engineering** | `app.py` | Rules ketat (SELECT only, JOIN yang benar) + few-shot examples untuk konsistensi output AI |
| **SQL Validation** | `main.go` | `isReadOnlySQL()` — hanya SELECT/WITH yang diizinkan |
| **Output Cleaning** | `main.go` | `cleanSQLQuery()` — hapus noise, komentar, teks non-SQL dari output AI |
| **Auto LIMIT** | `main.go` | `ensureLimit()` — tambahkan LIMIT 100 otomatis mencegah result set berlebihan |
| **Retry Logic** | `main.go` | Jika SQL pertama gagal, otomatis retry dengan mengirim error context ke AI untuk perbaikan |
| **CORS** | `main.go` | Aktif via `gin-contrib/cors`, configurable via `CORS_ORIGINS` env var |

---

## 🧪 Contoh Penggunaan (cURL)

**Request:**

```bash
curl -X POST http://localhost:8080/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "How many employees are in the Engineering department?"}'
```

**Response:**

```json
{
  "data": [
    {
      "COUNT(*)": 2
    }
  ],
  "generated_sql": "SELECT COUNT(*) FROM employees WHERE department_id = 1 LIMIT 100",
  "question": "How many employees are in the Engineering department?"
}
```

---

## 📁 Struktur Database (6 Tabel HRIS)

| Tabel | Kolom Utama |
|-------|-------------|
| `departments` | id, name, budget, created_at |
| `employees` | id, first_name, last_name, email, phone, hire_date, job_title, salary, department_id |
| `attendance_logs` | id, employee_id, clock_in, clock_out |
| `payroll` | id, employee_id, pay_date, basic_salary, bonus, deductions, net_salary |
| `projects` | id, name, status, start_date, end_date |
| `employee_projects` | employee_id, project_id, role |

---

## 📂 Struktur Project

```
text-to-sql-portfolio/
├── ai-service/                  # AI Inference Service (Python/FastAPI)
│   ├── app.py                   # Entry point: model loading & /generate-sql endpoint
│   ├── requirements.txt         # Python dependencies (core + optional)
│   ├── .env                     # Konfigurasi (tidak di-commit)
│   └── .env.example             # Template .env
│
├── backend-service/             # API Gateway (Go/Gin)
│   ├── main.go                  # Entry point: HTTP server, routing, retry logic
│   ├── database.go              # SQLite init, seed data, dynamic SQL scanner
│   ├── run_seed.py              # Script seed data
│   └── go.mod / go.sum          # Go dependencies
│
├── frontend-service/            # UI (SvelteKit 2 / Svelte 5)
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +page.svelte     # Main page
│   │   │   └── +layout.svelte
│   │   └── app.html
│   ├── package.json
│   ├── svelte.config.js
│   └── vite.config.js
│
├── README.md
└── .gitignore