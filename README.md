# 🧠 Text-to-SQL AI Assistant (Multi-Domain)

Proyek ini adalah implementasi *End-to-End* dari arsitektur AI Text-to-SQL yang mendukung **multi-domain database**. Sistem ini memungkinkan pengguna untuk mengekstrak data dari database relasional menggunakan pertanyaan bahasa natural (Bahasa Inggris).

**Dua domain tersedia:**
- **HRIS** — Human Resource Information System (karyawan, departemen, payroll, dll.)
- **Smart City** — Sistem monitoring kota pintar (kamera CCTV, lalu lintas, pelanggaran, insiden, dll.)

Sistem ini menunjukkan pemisahan beban kerja (*separation of concerns*) antara layanan Inferensi AI dan layanan Backend utama.

---

## 🔗 Live Demo (Production Release)

*   **Frontend Web App**: [https://frontend-service-production-52c3.up.railway.app](https://frontend-service-production-52c3.up.railway.app)
*   **Backend API**: [https://text-to-sql-portfolio-production.up.railway.app](https://text-to-sql-portfolio-production.up.railway.app)

---

## 🏗️ Arsitektur Sistem

Proyek ini mengadopsi arsitektur *microservices* sederhana:

| Layer | Teknologi | Deskripsi |
|-------|-----------|-----------|
| **Layanan AI** | Python / FastAPI | Bertindak sebagai "otak" sistem. Mendukung **3 mode inference**: **Groq API** (default, cloud), **Zero-Shot** (model lokal), dan **LoRA Fine-tuned** (model lokal + adapter). Menerima context DDL yang berbeda per domain. |
| **Layanan Backend** | Golang / Gin | Bertindak sebagai "jembatan" utama. Mengelola **multi-database** (HRIS + Smart City), membaca DDL schema secara dinamis, meminta query SQL dari layanan AI, melakukan validasi keamanan, dan mengeksekusi query. Dilengkapi **retry logic** otomatis. |
| **Frontend** | SvelteKit 2 / Svelte 5 | UI interaktif dengan **Domain Selector**, fitur **Dark Mode**, **Schema Explorer**, **Query History**, **Pagination**, **Export CSV**, dan **SQL Explanation**. |
| **Database** | SQLite (multi-file) | Dua file database terpisah: `hris.db` dan `smartcity.db`, dibuat otomatis beserta *dummy data* saat pertama kali dijalankan. |

### Alur Data

```mermaid
graph TD
    subgraph Client Space
        User([User Browser])
    end

    subgraph Production Cloud Environment (e.g., Railway)
        Frontend[Frontend Service SvelteKit<br>Port 3000]
        
        subgraph Private Network (analytics-network)
            Backend[Backend Service Go/Gin<br>Port 8080]
            AIService[FastAPI AI Service<br>Port 8000]
        end
        
        Volume[(Persistent SQLite Volume<br>/app/data)]
    end

    subgraph External Cloud Services
        Groq[Groq API Cloud<br>Llama 3.1 8B]
    end

    User -->|HTTPS| Frontend
    User -->|Client HTTP Fetch /ask| Backend
    Backend -->|Internal DNS Resolve| AIService
    AIService -->|Secure API RPC| Groq
    Backend -->|Local Read/Write| Volume
```

---

## ✨ Fitur Utama

### Fitur Inti (Essential)
| Fitur | Deskripsi |
|-------|-----------|
| **Text-to-SQL** | Konversi pertanyaan bahasa natural ke SQL menggunakan AI |
| **Multi-Domain** | Dukungan HRIS dan Smart City dengan database, schema, dan prompt terpisah |
| **Domain Selector** | Dropdown di frontend untuk memilih domain (HRIS / Smart City) |
| **SQL Execution & Display** | Eksekusi query otomatis dan tampilkan hasil dalam tabel |
| **Error Handling & Retry** | Retry otomatis jika SQL pertama gagal, dengan error feedback ke AI |
| **SQL Validation** | Hanya SELECT/WITH yang diizinkan (read-only) |
| **Auto LIMIT** | Tambahkan LIMIT 100 otomatis mencegah result set berlebihan |
| **Skeleton Loading** | Animasi loading shimmer saat menunggu response |

### Fitur Medium
| Fitur | Lokasi | Deskripsi |
|-------|--------|-----------|
| **⚡ Response Time** | Frontend + Backend | Badge waktu eksekusi query (ms) ditampilkan di SQL box dan footer tabel |
| **🗄️ Schema Explorer** | Frontend + Backend | Sidebar interaktif yang menampilkan semua tabel, kolom, tipe data, dan primary key secara real-time (via `GET /schema?domain=...`) |
| **📄 Pagination** | Frontend | Tabel dibagi per 20 baris dengan navigasi halaman (Prev/Next + nomor halaman) |
| **🌙 Dark Mode** | Frontend | Toggle dark/light mode di header, menggunakan CSS custom properties, preferensi disimpan di localStorage |
| **🩺 Health Status** | Frontend + Backend | 2 indikator dot hijau/merah di header menampilkan status koneksi Backend DB dan AI Service (via `GET /health`) |

### Fitur Tambahan
| Fitur | Lokasi | Deskripsi |
|-------|--------|-----------|
| **💡 Suggested Questions** | Frontend | Contoh pertanyaan per domain dalam bentuk chip yang bisa diklik langsung |
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

> Backend akan secara otomatis membuat file `hris.db` dan `smartcity.db` serta mengisinya dengan data dummy. Schema database dibaca **secara dinamis** dari `sqlite_master`.

### 3. Menjalankan Frontend (Port 5173)

```bash
cd frontend-service
npm install
npm run dev
```

### ⚡ Menjalankan dengan Docker Compose (Rekomendasi Produksi Lokal)

Seluruh sistem dapat dijalankan secara instan dalam kontainer Docker menggunakan **Docker Compose**. 

1. **Setup `.env` di Root Workspace**:
   Salin template dan isi API key Groq kamu:
   ```bash
   cp .env.example .env
   ```
   Buka file `.env` lalu masukkan API key Groq kamu:
   ```env
   GROQ_API_KEY=gsk_your_actual_groq_api_key_here
   ```

2. **Jalankan Docker Compose**:
   ```bash
   docker compose up --build
   ```

Setelah seluruh container sehat:
- **Frontend** dapat diakses di: `http://localhost:3000`
- **Backend API** dapat diakses di: `http://localhost:8080`
- **AI Service** dapat diakses di: `http://localhost:8000`

> **Penyimpanan Database**: Database SQLite akan secara otomatis disimpan secara persisten di folder `./data` di root workspace. Anda dapat menginspeksi file database (`hris.db`, `smartcity.db`) menggunakan tool database visual apa pun dari host machine Anda.

---

## 🌐 API Endpoints

| Method | Endpoint | Parameter | Deskripsi |
|--------|----------|-----------|-----------|
| `POST` | `/ask` | `question`, `domain` | **Endpoint utama.** Kirim pertanyaan + domain, dapatkan SQL + hasil query + response time |
| `GET` | `/health` | — | Health check — status koneksi DB dan AI Service |
| `GET` | `/schema` | `domain` | Struktur database per domain — daftar tabel, kolom, tipe data, primary key |
| `POST` | `/explain` | `question`, `sql`, `domain` | Penjelasan natural language dari SQL query |
| `POST` | `/benchmark/execute` | `sql`, `domain` | Eksekusi raw SQL (khusus benchmark) |

### Contoh Request `/ask`

```json
POST /ask
{
  "question": "How many employees are in the Engineering department?",
  "domain": "hris"
}
```

### Contoh Response

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

### Backend Service (`backend-service/.env`)

| Variabel | Default | Keterangan |
|----------|---------|------------|
| `AI_SERVICE_URL` | `http://127.0.0.1:8000` | URL AI service |
| `HRIS_DB_PATH` | `./hris.db` | Path ke file SQLite HRIS |
| `SMARTCITY_DB_PATH` | `./smartcity.db` | Path ke file SQLite Smart City |
| `BACKEND_PORT` | `8080` | Port backend service |
| `CORS_ORIGINS` | `*` | Allowed CORS origins (comma-separated) |

---

## 🛡️ Fitur Keamanan & Robustness

| Fitur | Lokasi | Fungsi |
|-------|--------|--------|
| **Dynamic Schema** | `main.go` | Membaca DDL aktual dari `sqlite_master` per database — otomatis akurat walau struktur tabel berubah |
| **Domain-Aware Prompt** | `app.py` | Prompt engineering yang berbeda per domain (HRIS vs Smart City) dengan rules dan few-shot examples yang sesuai |
| **SQL Validation** | `main.go` | `isReadOnlySQL()` — hanya SELECT/WITH yang diizinkan |
| **Output Cleaning** | `main.go` | `cleanSQLQuery()` — hapus noise, komentar, teks non-SQL dari output AI |
| **Auto LIMIT** | `main.go` | `ensureLimit()` — tambahkan LIMIT 100 otomatis mencegah result set berlebihan |
| **Retry Logic** | `main.go` | Jika SQL pertama gagal, otomatis retry dengan mengirim error context ke AI untuk perbaikan |
| **CORS** | `main.go` | Aktif via `gin-contrib/cors`, configurable via `CORS_ORIGINS` env var |

---

## 📁 Struktur Database

### Domain HRIS (6 Tabel)

| Tabel | Kolom Utama |
|-------|-------------|
| `departments` | id, name, budget, created_at |
| `employees` | id, first_name, last_name, email, phone, hire_date, job_title, salary, department_id |
| `attendance_logs` | id, employee_id, clock_in, clock_out |
| `payroll` | id, employee_id, pay_date, basic_salary, bonus, deductions, net_salary |
| `projects` | id, name, status, start_date, end_date |
| `employee_projects` | employee_id, project_id, role |

### Domain Smart City (9 Tabel)

| Tabel | Kolom Utama |
|-------|-------------|
| `districts` | id, name, area_km2, population, budget, created_at |
| `zones` | id, name, district_id, zone_type, area_km2 |
| `roads` | id, road_name, district_id, zone_id, road_type, lanes, length_km, speed_limit |
| `camera_locations` | id, road_id, location_desc, camera_type, installed_date, is_active |
| `traffic_readings` | id, camera_id, reading_time, vehicle_count, avg_speed, congestion_level, is_peak_hour |
| `violations` | id, camera_id, vehicle_plate, violation_type, violation_time, speed_recorded, speed_limit, fine_amount, status |
| `weather_data` | id, zone_id, recorded_at, data_type, temperature, humidity, wind_speed, pm25, pm10, air_quality_index, weather_condition |
| `incidents` | id, district_id, incident_type, severity, description, reported_at, resolved_at, status, response_time_minutes |
| `infrastructure_projects` | id, project_name, district_id, project_type, budget, status, start_date, end_date |

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
│   ├── database.go              # HRIS SQLite init + seed data
│   ├── database_smartcity.go    # Smart City SQLite init + seed data (9 tabel, 500+ baris)
│   ├── .env                     # Konfigurasi (tidak di-commit)
│   └── go.mod / go.sum          # Go dependencies
│
├── frontend-service/            # UI (SvelteKit 2 / Svelte 5)
│   ├── src/
│   │   └── routes/
│   │       ├── +page.svelte         # Main page (domain selector + all features)
│   │       └── benchmark/
│   │           ├── +page.svelte     # Benchmark dashboard (multi-domain tabs, charts, tables)
│   │           └── +page.server.js  # Server loader (reads HRIS + Smart City results)
│   │   └── app.html
│   ├── package.json
│   ├── svelte.config.js
│   └── vite.config.js
│
├── benchmark/                   # Evaluation Benchmark Suite
│   ├── benchmark.json           # 50 pertanyaan evaluasi HRIS + expected SQL
│   ├── benchmark-smartcity.json # 30 pertanyaan evaluasi Smart City + expected SQL
│   ├── run_benchmark.py         # Script eksekusi benchmark otomatis (multi-domain)
│   ├── results.json             # Hasil benchmark HRIS
│   ├── results-smartcity.json   # Hasil benchmark Smart City
│   └── README.md                # Template & dokumentasi hasil
│
├── README.md
└── .gitignore
```

---

## 📊 Evaluation Benchmark

Proyek ini dilengkapi dengan **benchmark suite** otomatis untuk mengukur akurasi Text-to-SQL secara kuantitatif, mendukung **multi-domain**.

### Menjalankan Benchmark

```bash
# Pastikan semua service sudah running (ai-service :8000, backend :8080)
cd benchmark

# Benchmark SEMUA domain (HRIS + Smart City, default)
python run_benchmark.py

# Benchmark HRIS saja (50 pertanyaan)
python run_benchmark.py --domain hris

# Benchmark Smart City saja (30 pertanyaan)
python run_benchmark.py --domain smartcity

# Opsi tambahan
python run_benchmark.py --domain all --timeout 120 --delay 8
```

> **Default `--domain all`**: Menjalankan kedua domain secara berurutan. Hasil HRIS disimpan ke `results.json`, Smart City ke `results-smartcity.json`.

### Benchmark Dashboard (Frontend)

Dashboard benchmark tersedia di `/benchmark` dengan fitur:

| Fitur | Deskripsi |
|-------|-----------|
| **Domain Tabs** | Tab 👥 HRIS dan 🏙️ Smart City untuk switch antar domain |
| **KPI Cards** | Result Accuracy, Execution Rate, Avg Response, Exact Match |
| **Chart: Accuracy Donut** | Visualisasi persentase passed vs failed |
| **Chart: Category Bar** | Akurasi per kategori query (horizontal bar) |
| **Chart: Difficulty Grouped** | Executed % vs Match % per difficulty level |
| **Chart: Response Time** | Waktu response per pertanyaan (bar chart) |
| **Chart: Match Breakdown** | Stacked bar — exact/partial/mismatch per kategori |
| **Detailed Results Table** | Filterable & sortable, expandable rows untuk SQL comparison |
| **Dark Mode** | Toggle dark/light, preferensi tersimpan di localStorage |

### Benchmark Structure

#### HRIS (50 pertanyaan)

| Kategori | Jumlah | Difficulty | Deskripsi |
|----------|--------|------------|-----------|
| `simple_select` | 10 | Easy | Basic SELECT, COUNT, DISTINCT |
| `filter_where` | 8 | Easy | WHERE clauses, comparisons, LIKE |
| `aggregation` | 10 | Medium | GROUP BY, SUM, AVG, MAX, MIN |
| `join` | 10 | Medium | Single & multi-table JOINs |
| `complex` | 7 | Hard | HAVING, subqueries, ORDER BY + LIMIT |
| `advanced` | 5 | Hard | Window functions, CTEs, correlated subqueries |

#### Smart City (30 pertanyaan)

| Kategori | Jumlah | Difficulty | Deskripsi |
|----------|--------|------------|-----------|
| `simple_select` | 4 | Easy | Basic SELECT on cameras, districts, zones, violations |
| `filter_where` | 4 | Easy | Filter by status, type, numeric comparison |
| `aggregation` | 4 | Medium | Multi-table JOIN with GROUP BY, AVG, SUM, COUNT |
| `join` | 4 | Medium | Camera-road-district-zone JOINs |
| `complex` | 7 | Hard | Cross-domain queries, HAVING, subqueries |
| `advanced` | 7 | Hard | Window functions, CTEs, CASE WHEN, date functions |

### Metrics

| Metrik | Deskripsi |
|--------|-----------|
| **Execution Accuracy** | % query yang bisa dieksekusi tanpa error |
| **Result Accuracy** | % query yang menghasilkan data sesuai expected (exact/partial match) |
| **Exact Match** | % query yang hasilnya persis sama dengan expected |
| **Avg Response Time** | Rata-rata waktu response |

> Hasil benchmark tersimpan di `benchmark/results.json` (HRIS) dan `benchmark/results-smartcity.json` (Smart City). Detail lengkap: [benchmark/README.md](benchmark/README.md)

---

## 🎨 Screenshots

*Dashboard Utama (HRIS & Smart City Query Interface)*
![Dashboard Utama](docs/assets/dashboard_main.png)

*Benchmark Evaluation Dashboard (/benchmark)*
![Benchmark Dashboard](docs/assets/dashboard_benchmark.png)

---

## 📄 License

This project is open source and available for portfolio/educational purposes.