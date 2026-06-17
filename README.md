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
| **Frontend** | SvelteKit 2 / Svelte 5 | UI interaktif dengan fitur **Dark Mode**, **Schema Explorer**, **Query History**, **Pagination**, **Export CSV**, dan **SQL Explanation**. |
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

## ✨ Fitur Utama

### Fitur Inti (Essential)
| Fitur | Deskripsi |
|-------|-----------|
| **Text-to-SQL** | Konversi pertanyaan bahasa natural ke SQL menggunakan AI |
| **SQL Execution & Display** | Eksekusi query otomatis dan tampilkan hasil dalam tabel |
| **Error Handling & Retry** | Retry otomatis jika SQL pertama gagal, dengan error feedback ke AI |
| **SQL Validation** | Hanya SELECT/WITH yang diizinkan (read-only) |
| **Auto LIMIT** | Tambahkan LIMIT 100 otomatis mencegah result set berlebihan |
| **Skeleton Loading** | Animasi loading shimmer saat menunggu response |

### Fitur Medium
| Fitur | Lokasi | Deskripsi |
|-------|--------|-----------|
| **⚡ Response Time** | Frontend + Backend | Badge waktu eksekusi query (ms) ditampilkan di SQL box dan footer tabel |
| **🗄️ Schema Explorer** | Frontend + Backend | Sidebar interaktif yang menampilkan semua tabel, kolom, tipe data, dan primary key dari database secara real-time (via `GET /schema`) |
| **📄 Pagination** | Frontend | Tabel dibagi per 20 baris dengan navigasi halaman (Prev/Next + nomor halaman) |
| **🌙 Dark Mode** | Frontend | Toggle dark/light mode di header, menggunakan CSS custom properties, preferensi disimpan di localStorage |
| **🩺 Health Status** | Frontend + Backend | 2 indikator dot hijau/merah di header menampilkan status koneksi Backend DB dan AI Service (via `GET /health`) |

### Fitur Tambahan
| Fitur | Lokasi | Deskripsi |
|-------|--------|-----------|
| **💡 Suggested Questions** | Frontend | 6 contoh pertanyaan dalam bentuk chip yang bisa diklik langsung |
| **📜 Query History** | Frontend | Riwayat 20 query terakhir disimpan di localStorage, bisa di-load ulang atau dihapus |
| **📥 Export CSV** | Frontend | Download hasil query dalam format CSV |
| **📖 SQL Explanation** | Frontend + Backend | Penjelasan natural language dari SQL yang di-generate oleh AI (via `POST /explain`) |
| **📋 Copy SQL** | Frontend | Tombol copy generated SQL ke clipboard |
| **⌨️ Typing Animation** | Frontend | Efek mengetik saat SQL ditampilkan |

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

## 🌐 API Endpoints

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| `POST` | `/ask` | **Endpoint utama.** Kirim pertanyaan, dapatkan SQL + hasil query + response time |
| `GET` | `/health` | Health check — status koneksi DB dan AI Service |
| `GET` | `/schema` | Struktur database — daftar tabel, kolom, tipe data, primary key |
| `POST` | `/explain` | Penjelasan natural language dari SQL query |

### Contoh Response `/ask`

```json
{
  "question": "How many employees are in the Engineering department?",
  "generated_sql": "SELECT COUNT(*) FROM employees WHERE department_id = 1 LIMIT 100",
  "data": [{ "COUNT(*)": 2 }],
  "response_time": "1250ms",
  "response_ms": 1250
}
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
│   ├── app.py                   # Entry point: model loading, /generate-sql, /explain-sql
│   ├── requirements.txt         # Python dependencies (core + optional)
│   ├── .env                     # Konfigurasi (tidak di-commit)
│   └── .env.example             # Template .env
│
├── backend-service/             # API Gateway (Go/Gin)
│   ├── main.go                  # Entry point: HTTP server, routing, /ask, /health, /schema, /explain
│   ├── database.go              # SQLite init, seed data
│   ├── run_seed.py              # Script seed data
│   ├── .env                     # Konfigurasi (tidak di-commit)
│   └── go.mod / go.sum          # Go dependencies
│
├── frontend-service/            # UI (SvelteKit 2 / Svelte 5)
│   ├── src/
│   │   ├── routes/
│   │   │   └── +page.svelte     # Main page (all features in single file)
│   │   └── app.html
│   ├── package.json
│   ├── svelte.config.js
│   └── vite.config.js
│
├── benchmark/                   # Evaluation Benchmark Suite
│   ├── benchmark.json           # 50 pertanyaan evaluasi + expected SQL
│   ├── run_benchmark.py         # Script eksekusi benchmark otomatis
│   └── README.md                # Template & dokumentasi hasil
│
├── README.md
└── .gitignore
```

---

## 📊 Evaluation Benchmark

Proyek ini dilengkapi dengan **benchmark suite** otomatis untuk mengukur akurasi Text-to-SQL secara kuantitatif.

### Menjalankan Benchmark

```bash
# Pastikan semua service sudah running (ai-service :8000, backend :8080)
cd benchmark
python run_benchmark.py

# Opsi tambahan
python run_benchmark.py --backend-url http://localhost:8080 --timeout 120
```

### Benchmark Structure

| Kategori | Jumlah | Difficulty | Deskripsi |
|----------|--------|------------|-----------|
| `simple_select` | 10 | Easy | Basic SELECT, COUNT, DISTINCT |
| `filter_where` | 8 | Easy | WHERE clauses, comparisons, LIKE |
| `aggregation` | 10 | Medium | GROUP BY, SUM, AVG, MAX, MIN |
| `join` | 10 | Medium | Single & multi-table JOINs |
| `complex` | 7 | Hard | HAVING, subqueries, ORDER BY + LIMIT |
| `advanced` | 5 | Hard | Window functions, CTEs, correlated subqueries |

**Total: 50 pertanyaan evaluasi**

### Metrics

| Metrik | Deskripsi |
|--------|-----------|
| **Execution Accuracy** | % query yang bisa dieksekusi tanpa error |
| **Result Accuracy** | % query yang menghasilkan SQL sesuai expected (≥70% keyword match) |
| **Exact Match** | % query yang SQL-nya persis sama dengan expected |
| **Avg Response Time** | Rata-rata waktu response |

> Hasil benchmark tersimpan di `benchmark/results.json`. Detail lengkap: [benchmark/README.md](benchmark/README.md)

---

## 🎨 Screenshots



---

## 📄 License

This project is open source and available for portfolio/educational purposes.