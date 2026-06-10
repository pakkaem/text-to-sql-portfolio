package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
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

// Struktur untuk retry: request ke AI dengan error feedback
type AIRetryRequest struct {
	SchemaContext string `json:"schema_context"`
	Question      string `json:"question"`
	PrevSQL       string `json:"prev_sql"`
	ErrorMsg      string `json:"error_msg"`
}

// getSchemaFromDB membaca DDL aktual dari sqlite_master agar AI mendapat schema yang 100% akurat
// termasuk FOREIGN KEY, CHECK constraints, dan kolom terbaru
func getSchemaFromDB(db *sql.DB) string {
	rows, err := db.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		log.Printf("WARNING: Gagal membaca schema dari database: %v. Menggunakan fallback.", err)
		return hrisSchemaFallback
	}
	defer rows.Close()

	var schemaBuilder strings.Builder
	for rows.Next() {
		var sqlStmt string
		if err := rows.Scan(&sqlStmt); err != nil {
			continue
		}
		if sqlStmt != "" {
			schemaBuilder.WriteString(sqlStmt + ";\n\n")
		}
	}

	schema := strings.TrimSpace(schemaBuilder.String())
	if schema == "" {
		log.Println("WARNING: Schema kosong dari database. Menggunakan fallback.")
		return hrisSchemaFallback
	}
	return schema
}

// Fallback schema jika gagal baca dari database
const hrisSchemaFallback = `
CREATE TABLE departments (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL);
CREATE TABLE employees (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, department_id INTEGER, job_title TEXT, hire_date TEXT NOT NULL);
CREATE TABLE attendance_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, employee_id INTEGER, log_date TEXT NOT NULL, status TEXT);
CREATE TABLE payroll (id INTEGER PRIMARY KEY AUTOINCREMENT, employee_id INTEGER, month_year TEXT NOT NULL, base_salary REAL NOT NULL, bonus REAL DEFAULT 0);
CREATE TABLE projects (id INTEGER PRIMARY KEY AUTOINCREMENT, project_name TEXT NOT NULL, budget REAL NOT NULL, status TEXT);
CREATE TABLE employee_projects (employee_id INTEGER, project_id INTEGER, role TEXT NOT NULL, PRIMARY KEY (employee_id, project_id));
`

// isReadOnlySQL memvalidasi bahwa query hanya berupa SELECT statement (read-only)
func isReadOnlySQL(query string) bool {
	normalized := strings.TrimSpace(query)
	normalized = strings.ToUpper(normalized)
	return strings.HasPrefix(normalized, "SELECT") || strings.HasPrefix(normalized, "WITH")
}

// ensureLimit menambahkan LIMIT jika query tidak memiliki LIMIT clause
func ensureLimit(query string, maxRows int) string {
	upper := strings.ToUpper(strings.TrimSpace(query))
	if !strings.Contains(upper, " LIMIT ") {
		query = strings.TrimRight(strings.TrimSpace(query), ";")
		query = fmt.Sprintf("%s LIMIT %d", query, maxRows)
	}
	return query
}

// getEnv mengambil environment variable dengan fallback default
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

// callAIService mengirim request ke AI service dan mengembalikan generated SQL
func callAIService(aiServiceURL string, reqBody interface{}) (string, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("gagal marshal request: %w", err)
	}

	aiResp, err := http.Post(aiServiceURL+"/generate-sql", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("gagal terhubung ke AI Service: %w", err)
	}
	defer aiResp.Body.Close()

	if aiResp.StatusCode != 200 {
		errBody, _ := io.ReadAll(aiResp.Body)
		return "", fmt.Errorf("AI Service error (status %d): %s", aiResp.StatusCode, string(errBody))
	}

	bodyBytes, err := io.ReadAll(aiResp.Body)
	if err != nil {
		return "", fmt.Errorf("gagal membaca response AI: %w", err)
	}

	var aiResult AIResponse
	if err := json.Unmarshal(bodyBytes, &aiResult); err != nil {
		return "", fmt.Errorf("gagal parse response AI: %w", err)
	}

	return strings.TrimSpace(aiResult.SQLQuery), nil
}

func main() {
	// 0. Baca konfigurasi dari environment variables (dengan default)
	backendPort := getEnv("BACKEND_PORT", "8080")
	aiServiceURL := getEnv("AI_SERVICE_URL", "http://127.0.0.1:8000")
	corsOrigins := getEnv("CORS_ORIGINS", "*")

	// 1. Inisialisasi Database (memanggil fungsi dari database.go)
	db := initDB()
	defer db.Close()

	// 2. Baca schema aktual dari database (Fix: Dynamic Schema)
	hrisSchema := getSchemaFromDB(db)
	log.Printf("Schema berhasil dibaca dari database (%d karakter)", len(hrisSchema))

	// 3. Setup Gin Router
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(corsOrigins, ","),
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"Content-Length"},
	}))

	// 4. Health Check Endpoint
	r.GET("/health", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"db":     "disconnected",
				"ai_url": aiServiceURL,
				"error":  err.Error(),
			})
			return
		}

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
			httpStatus = http.StatusPartialContent
		}

		c.JSON(httpStatus, gin.H{
			"status":     status,
			"db":         "connected",
			"ai_service": aiHealthy,
			"ai_url":     aiServiceURL,
		})
	})

	// 5. Endpoint Utama
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

		generatedSQL, err := callAIService(aiServiceURL, aiReqBody)
		if err != nil {
			log.Printf("ERROR: %v", err)
			if strings.Contains(err.Error(), "gagal terhubung") {
				c.JSON(http.StatusBadGateway, gin.H{"error": "Gagal terhubung ke AI Service", "ai_url": aiServiceURL})
			} else {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			}
			return
		}

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
		// Tambahkan LIMIT otomatis jika tidak ada (mencegah result set berlebihan)
		generatedSQL = ensureLimit(generatedSQL, 100)

		rows, err := db.Query(generatedSQL)

		// --- RETRY LOGIC: Jika gagal, minta AI perbaiki query ---
		if err != nil {
			log.Printf("WARNING: SQL pertama gagal: %s | Error: %v. Mencoba retry...", generatedSQL, err)

			retryReq := AIRetryRequest{
				SchemaContext: hrisSchema,
				Question:      req.Question,
				PrevSQL:       generatedSQL,
				ErrorMsg:      err.Error(),
			}

			retrySQL, retryErr := callAIService(aiServiceURL, retryReq)
			if retryErr != nil {
				log.Printf("ERROR: Retry juga gagal: %v", retryErr)
				c.JSON(http.StatusBadRequest, gin.H{
					"error":          "AI menghasilkan SQL yang tidak valid (retry gagal)",
					"generated_sql": generatedSQL,
					"db_message":     err.Error(),
				})
				return
			}

			if !isReadOnlySQL(retrySQL) {
				log.Printf("WARNING: AI retry menghasilkan non-SELECT query: %s", retrySQL)
				c.JSON(http.StatusForbidden, gin.H{
					"error":          "Query ditolak: hanya SELECT yang diizinkan (setelah retry)",
					"generated_sql": retrySQL,
				})
				return
			}

			retrySQL = ensureLimit(retrySQL, 100)
			rows, err = db.Query(retrySQL)
			if err != nil {
				log.Printf("ERROR: Retry SQL juga gagal dieksekusi: %s | Error: %v", retrySQL, err)
				c.JSON(http.StatusBadRequest, gin.H{
					"error":          "AI menghasilkan SQL yang tidak valid (setelah retry)",
					"generated_sql": retrySQL,
					"original_sql":   generatedSQL,
					"db_message":     err.Error(),
				})
				return
			}
			generatedSQL = retrySQL // Update generated SQL untuk response
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
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := rows.Scan(columnPointers...); err != nil {
				log.Printf("WARNING: Gagal scan row: %v", err)
				continue
			}

			rowData := make(map[string]interface{})
			for i, colName := range cols {
				val := columnPointers[i].(*interface{})
				if b, ok := (*val).([]byte); ok {
					rowData[colName] = string(b)
				} else {
					rowData[colName] = *val
				}
			}
			finalResult = append(finalResult, rowData)
		}

		if err := rows.Err(); err != nil {
			log.Printf("WARNING: Error saat iterasi rows: %v", err)
		}

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