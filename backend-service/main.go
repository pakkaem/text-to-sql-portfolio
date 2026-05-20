package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
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
CREATE TABLE projects (id INTEGER PRIMARY KEY AUTOINCREMENT, project_name TEXT NOT NULL, budget REAL NOT NULL, status TEXT);
CREATE TABLE employee_projects (employee_id INTEGER, project_id INTEGER, role TEXT NOT NULL, PRIMARY KEY (employee_id, project_id));
`

// getEnv mengambil environment variable dengan fallback default
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

// isReadOnlySQL memvalidasi bahwa query hanya berupa SELECT statement (read-only)
func isReadOnlySQL(query string) bool {
	// Trim whitespace dan newline, ambil kata pertama
	normalized := strings.TrimSpace(query)
	normalized = strings.ToUpper(normalized)

	// Hanya izinkan SELECT dan WITH (CTE yang diakhiri SELECT)
	return strings.HasPrefix(normalized, "SELECT") || strings.HasPrefix(normalized, "WITH")
}

func main() {
	// 0. Baca konfigurasi dari environment variables (dengan default)
	backendPort := getEnv("BACKEND_PORT", "8080")
	aiServiceURL := getEnv("AI_SERVICE_URL", "http://127.0.0.1:8000")
	corsOrigins := getEnv("CORS_ORIGINS", "*")

	// 1. Inisialisasi Database (memanggil fungsi dari database.go)
	db := initDB()
	defer db.Close()

	// 2. Setup Gin Router
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(corsOrigins, ","),
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"Content-Length"},
	}))

	// 3. Health Check Endpoint
	r.GET("/health", func(c *gin.Context) {
		// Cek koneksi database
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"db":      "disconnected",
				"ai_url":  aiServiceURL,
				"error":   err.Error(),
			})
			return
		}

		// Cek koneksi ke AI Service
		aiHealthy := false
		aiCheckResp, err := http.Get(aiServiceURL + "/docs")
		if err == nil && aiCheckResp.StatusCode == 200 {
			aiHealthy = true
			aiCheckResp.Body.Close()
		}

		status := "healthy"
		httpStatus := http.StatusOK
		if !aiHealthy {
			status = "degraded"
			httpStatus = http.StatusPartialContent // 206: DB ok, AI tidak tersedia
		}

		c.JSON(httpStatus, gin.H{
			"status":     status,
			"db":         "connected",
			"ai_service": aiHealthy,
			"ai_url":     aiServiceURL,
		})
	})

	// 4. Endpoint Utama
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
		jsonData, err := json.Marshal(aiReqBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses request internal"})
			return
		}

		// Tembak ke API FastAPI
		aiResp, err := http.Post(aiServiceURL+"/generate-sql", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("ERROR: Gagal terhubung ke AI Service: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "Gagal terhubung ke AI Service", "ai_url": aiServiceURL})
			return
		}
		defer aiResp.Body.Close()

		if aiResp.StatusCode != 200 {
			// Baca body error dari AI service untuk debugging
			errBody, _ := io.ReadAll(aiResp.Body)
			log.Printf("ERROR: AI Service mengembalikan status %d: %s", aiResp.StatusCode, string(errBody))
			c.JSON(http.StatusBadGateway, gin.H{
				"error":    "AI Service mengembalikan error",
				"ai_error": string(errBody),
			})
			return
		}

		bodyBytes, err := io.ReadAll(aiResp.Body)
		if err != nil {
			log.Printf("ERROR: Gagal membaca response dari AI Service: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca response dari AI Service"})
			return
		}

		var aiResult AIResponse
		if err := json.Unmarshal(bodyBytes, &aiResult); err != nil {
			log.Printf("ERROR: Gagal parse response AI: %v, body: %s", err, string(bodyBytes))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal parse response dari AI Service"})
			return
		}

		generatedSQL := strings.TrimSpace(aiResult.SQLQuery)

		// --- VALIDASI SQL READ-ONLY ---
		if !isReadOnlySQL(generatedSQL) {
			log.Printf("WARNING: AI menghasilkan non-SELECT query: %s", generatedSQL)
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Query ditolak: hanya SELECT yang diizinkan",
				"generated_sql": generatedSQL,
			})
			return
		}

		// --- FASE 2: EKSEKUSI SQL KE DATABASE (DYNAMIC SCANNING) ---
		rows, err := db.Query(generatedSQL)
		if err != nil {
			log.Printf("ERROR: Gagal eksekusi SQL: %s | Error: %v", generatedSQL, err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":          "AI menghasilkan SQL yang tidak valid",
				"generated_sql": generatedSQL,
				"db_message":     err.Error(),
			})
			return
		}
		defer rows.Close()

		// Ambil nama-nama kolom dari hasil query
		cols, err := rows.Columns()
		if err != nil {
			log.Printf("ERROR: Gagal mengambil kolom: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca struktur hasil query"})
			return
		}

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
				log.Printf("WARNING: Gagal scan row: %v", err)
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

		// Cek error dari iterasi rows
		if err := rows.Err(); err != nil {
			log.Printf("WARNING: Error saat iterasi rows: %v", err)
		}

		// Handle case data kosong (bukan null)
		if finalResult == nil {
			finalResult = []map[string]interface{}{}
		}

		// --- FASE 3: KEMBALIKAN HASIL KE USER ---
		c.JSON(http.StatusOK, gin.H{
			"question":      req.Question,
			"generated_sql": generatedSQL,
			"data":          finalResult,
		})
	})

	log.Printf("Backend Golang berjalan di http://localhost:%s", backendPort)
	log.Printf("AI Service URL: %s", aiServiceURL)
	r.Run(":" + backendPort)
}