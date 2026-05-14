# 🧠 Text-to-SQL AI Assistant (HRIS Domain)

Proyek ini adalah implementasi *End-to-End* dari arsitektur AI Text-to-SQL. Sistem ini memungkinkan pengguna untuk mengekstrak data dari database relasional (HRIS) menggunakan pertanyaan bahasa natural (Bahasa Inggris).

Sistem ini menunjukkan pemisahan beban kerja (*separation of concerns*) antara layanan Inferensi AI dan layanan Backend utama.

---

## 🏗️ Arsitektur Sistem

Proyek ini mengadopsi arsitektur *microservices* sederhana:

| Layer | Teknologi | Deskripsi |
|-------|-----------|-----------|
| **Layanan AI** | Python / FastAPI | Bertindak sebagai "otak" sistem. Memuat base model **TinyLlama-1.1B** yang telah di-*fine-tune* menggunakan metode **LoRA (Low-Rank Adaptation)** untuk menghasilkan query SQL yang valid berdasarkan konteks skema database. |
| **Layanan Backend** | Golang / Gin | Bertindak sebagai "jembatan" utama. Menerima request HTTP dari pengguna, menyuntikkan skema DDL, meminta query SQL dari layanan AI, dan mengeksekusi query tersebut langsung ke database menggunakan teknik *Dynamic SQL Scanning*. |
| **Database** | SQLite | Penyimpanan data terintegrasi (No-Ops) yang dibuat otomatis beserta *dummy data* saat aplikasi pertama kali dijalankan. |

---

## 🚀 Cara Menjalankan Aplikasi Secara Lokal

### 1. Menjalankan Layanan AI (Port 8000)

Pastikan Anda memiliki **Python 3.9+** dan menginstal library yang dibutuhkan.

```bash
cd ai-service
python -m venv venv
# Aktifkan venv
# Windows:   venv\Scripts\activate
# Mac/Linux: source venv/bin/activate
pip install -r requirements.txt
python app.py
```

> **Catatan:** Saat dijalankan pertama kali, skrip akan mengunduh base model secara otomatis ke dalam cache lokal.

### 2. Menjalankan Layanan Backend (Port 8080)

Pastikan Anda memiliki **Golang** terinstal di sistem Anda.

```bash
cd backend-service
go run .
```

> Backend akan secara otomatis membuat file `hris.db` dan mengisinya dengan data HRIS dummy.

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
  "generated_sql": "SELECT COUNT(*) FROM employees WHERE department_id = 1;",
  "question": "How many employees are in the Engineering department?"
}