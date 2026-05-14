package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Struktur request dari User/Frontend
type AskRequest struct {
	Question string `json:"question" binding:"required"`
}

// Struktur request yang akan dikirim ke AI Python
type AIRequest struct {
	SchemaContext string `json:"schema_context"`
	Question      string `json:"question"`
}

// Struktur response dari AI Python
type AIResponse struct {
	SQLQuery string `json:"sql_query"`
}

// Skema DDL HRIS yang jadi contekan untuk AI (Disamakan dengan skema di database.go)
const hrisSchema = `
CREATE TABLE departments (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL);
CREATE TABLE employees (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, department_id INTEGER, job_title TEXT, hire_date TEXT NOT NULL);
CREATE TABLE attendance_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, employee_id INTEGER, log_date TEXT NOT NULL, status TEXT);
CREATE TABLE payroll (id INTEGER PRIMARY KEY AUTOINCREMENT, employee_id INTEGER, month_year TEXT NOT NULL, base_salary REAL NOT NULL, bonus REAL DEFAULT 0);
`

func main() {
	// 1. Inisialisasi Database (memanggil fungsi dari database.go)
	db := initDB()
	defer db.Close()

	// 2. Setup Gin Router
	r := gin.Default()

	// 3. Endpoint Utama
	r.POST("/ask", func(c *gin.Context) {
		var req AskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON tidak valid atau question kosong"})
			return
		}

		// --- FASE 1: BERTANYA KE AI PYTHON ---
		aiReqBody := AIRequest{
			SchemaContext: hrisSchema,
			Question:      req.Question,
		}
		jsonData, _ := json.Marshal(aiReqBody)

		// Tembak ke API FastAPI (Pastikan port 8000 sesuai dengan Python-mu)
		aiResp, err := http.Post("http://127.0.0.1:8000/generate-sql", "application/json", bytes.NewBuffer(jsonData))
		if err != nil || aiResp.StatusCode != 200 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal terhubung ke AI Service"})
			return
		}
		defer aiResp.Body.Close()

		bodyBytes, _ := io.ReadAll(aiResp.Body)
		var aiResult AIResponse
		json.Unmarshal(bodyBytes, &aiResult)

		generatedSQL := aiResult.SQLQuery

		// --- FASE 2: EKSEKUSI SQL KE DATABASE (DYNAMIC SCANNING) ---
		rows, err := db.Query(generatedSQL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":          "AI menghasilkan SQL yang tidak valid",
				"generated_sql": generatedSQL,
				"db_message":     err.Error(),
			})
			return
		}
		defer rows.Close()

		// Ambil nama-nama kolom dari hasil query
		cols, _ := rows.Columns()
		
		var finalResult []map[string]interface{}

		for rows.Next() {
			// Buat wadah dinamis berdasarkan jumlah kolom
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			// Scan data dari database
			if err := rows.Scan(columnPointers...); err != nil {
				continue
			}

			// Petakan data ke bentuk JSON (Map)
			rowData := make(map[string]interface{})
			for i, colName := range cols {
				val := columnPointers[i].(*interface{})
				
				// Konversi tipe data byte ke string agar terbaca di JSON
				if b, ok := (*val).([]byte); ok {
					rowData[colName] = string(b)
				} else {
					rowData[colName] = *val
				}
			}
			finalResult = append(finalResult, rowData)
		}

		// --- FASE 3: KEMBALIKAN HASIL KE USER ---
		c.JSON(http.StatusOK, gin.H{
			"question":      req.Question,
			"generated_sql": generatedSQL,
			"data":          finalResult,
		})
	})

	log.Println("Backend Golang berjalan di http://localhost:8080")
	r.Run(":8080")
}