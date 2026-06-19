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
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Struktur request dari User/Frontend
type AskRequest struct {
	Question string `json:"question" binding:"required"`
	Domain   string `json:"domain"`
}

// Struktur request yang akan dikirim ke AI Python
type AIRequest struct {
	SchemaContext string `json:"schema_context"`
	Question      string `json:"question"`
	Domain        string `json:"domain"`
}

// Struktur response dari AI Python
type AIResponse struct {
	SQLQuery string `json:"sql_query"`
}

// Struktur request untuk AI Insight
type AIInsightRequest struct {
	Question  string                   `json:"question"`
	SQLQuery  string                   `json:"sql_query"`
	Data      []map[string]interface{} `json:"data"`
	Domain    string                   `json:"domain"`
}

// Struktur response dari AI Insight
type AIInsightResponse struct {
	InsightSummary     string   `json:"insight_summary"`
	BusinessExplanation string   `json:"business_explanation"`
	TopFindings        []string `json:"top_findings"`
}

// Struktur untuk retry: request ke AI dengan error feedback
type AIRetryRequest struct {
	SchemaContext string `json:"schema_context"`
	Question      string `json:"question"`
	PrevSQL       string `json:"prev_sql"`
	ErrorMsg      string `json:"error_msg"`
	Domain        string `json:"domain"`
}

// getSchemaFromDB membaca DDL aktual dari sqlite_master agar AI mendapat schema yang 100% akurat
func getSchemaFromDB(db *sql.DB) string {
	rows, err := db.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		log.Printf("WARNING: Gagal membaca schema dari database: %v", err)
		return ""
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
	return schema
}

// isReadOnlySQL memvalidasi bahwa query hanya berupa SELECT statement (read-only)
func isReadOnlySQL(query string) bool {
	normalized := strings.TrimSpace(query)
	normalized = strings.ToUpper(normalized)
	return strings.HasPrefix(normalized, "SELECT") || strings.HasPrefix(normalized, "WITH")
}

// cleanSQLQuery membersihkan output model AI dari noise
func cleanSQLQuery(query string) string {
	query = strings.TrimSpace(query)

	lines := strings.Split(query, "\n")
	var sqlLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(line), "THE ") ||
			strings.HasPrefix(strings.ToUpper(line), "THIS ") ||
			strings.HasPrefix(strings.ToUpper(line), "HERE ") ||
			strings.HasPrefix(strings.ToUpper(line), "NOTE ") {
			break
		}
		sqlLines = append(sqlLines, line)
	}
	query = strings.Join(sqlLines, " ")

	query = strings.TrimRight(query, ";")

	upper := strings.ToUpper(query)
	incompleteSuffixes := []string{
		" ON", " AND", " OR", " WHERE", " JOIN", " FROM", " SET",
		" GROUP BY", " ORDER BY", " HAVING", " SELECT",
	}
	for _, suffix := range incompleteSuffixes {
		if strings.HasSuffix(upper, suffix) {
			query = strings.TrimRight(query[:len(query)-len(suffix)], " ")
			upper = strings.ToUpper(query)
		}
	}

	return query
}

// ensureLimit menambahkan LIMIT jika query tidak memiliki LIMIT clause
func ensureLimit(query string, maxRows int) string {
	query = strings.TrimSpace(query)
	upper := strings.ToUpper(query)

	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return query
	}

	if len(query) < 15 {
		return query
	}

	if !strings.Contains(upper, " LIMIT ") {
		query = strings.TrimRight(query, ";")
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

// callAIInsightService mengirim request ke AI service untuk menghasilkan insight bisnis
func callAIInsightService(aiServiceURL string, reqBody AIInsightRequest) (*AIInsightResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal insight request: %w", err)
	}

	aiResp, err := http.Post(aiServiceURL+"/generate-insight", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke AI Service (insight): %w", err)
	}
	defer aiResp.Body.Close()

	if aiResp.StatusCode != 200 {
		errBody, _ := io.ReadAll(aiResp.Body)
		return nil, fmt.Errorf("AI Insight Service error (status %d): %s", aiResp.StatusCode, string(errBody))
	}

	bodyBytes, err := io.ReadAll(aiResp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca insight response: %w", err)
	}

	var insightResult AIInsightResponse
	if err := json.Unmarshal(bodyBytes, &insightResult); err != nil {
		return nil, fmt.Errorf("gagal parse insight response: %w", err)
	}

	return &insightResult, nil
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

// executeQueryOnDomain menjalankan query SQL pada database domain tertentu
func executeQueryOnDomain(domainInfo *DomainInfo, query string) ([]map[string]interface{}, error) {
	query = cleanSQLQuery(query)

	if !isReadOnlySQL(query) {
		return nil, fmt.Errorf("query ditolak: hanya SELECT yang diizinkan")
	}

	query = ensureLimit(query, 100)

	rows, err := domainInfo.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
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
		result = append(result, rowData)
	}

	if result == nil {
		result = []map[string]interface{}{}
	}

	return result, nil
}

func main() {
	// 0. Baca konfigurasi dari environment variables (dengan default)
	backendPort := getEnv("BACKEND_PORT", "8080")
	aiServiceURL := getEnv("AI_SERVICE_URL", "http://127.0.0.1:8000")
	corsOrigins := getEnv("CORS_ORIGINS", "*")

	// 1. Inisialisasi Semua Database (multi-domain)
	domains := initAllDatabases()
	defer func() {
		for _, d := range domains {
			d.DB.Close()
		}
	}()

	// 2. Setup Gin Router
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(corsOrigins, ","),
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"Content-Length"},
	}))

	// 3. Health Check Endpoint — cek semua domain
	r.GET("/health", func(c *gin.Context) {
		dbStatuses := make(map[string]string)
		allHealthy := true

		for name, info := range domains {
			if err := info.DB.Ping(); err != nil {
				dbStatuses[name] = "disconnected"
				allHealthy = false
			} else {
				dbStatuses[name] = "connected"
			}
		}

		aiHealthy := false
		aiCheckResp, err := http.Get(aiServiceURL + "/docs")
		if err == nil && aiCheckResp.StatusCode == 200 {
			aiHealthy = true
			aiCheckResp.Body.Close()
		}

		status := "healthy"
		httpStatus := http.StatusOK
		if !allHealthy || !aiHealthy {
			status = "degraded"
			httpStatus = http.StatusPartialContent
		}

		c.JSON(httpStatus, gin.H{
			"status":     status,
			"databases":  dbStatuses,
			"ai_service": aiHealthy,
			"ai_url":     aiServiceURL,
		})
	})

	// 4. Domains Endpoint — daftar semua domain yang tersedia
	r.GET("/domains", func(c *gin.Context) {
		type DomainMeta struct {
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			Description string `json:"description"`
			TableCount  int    `json:"table_count"`
		}

		var domainList []DomainMeta
		for _, info := range domains {
			domainList = append(domainList, DomainMeta{
				Name:        info.Name,
				DisplayName: info.DisplayName,
				Description: info.Description,
				TableCount:  info.TableCount,
			})
		}

		c.JSON(http.StatusOK, gin.H{"domains": domainList})
	})

	// 5. Endpoint: Get Schema Info (per domain)
	type ColumnInfo struct {
		Name string `json:"name"`
		Type string `json:"type"`
		PK   bool   `json:"pk"`
	}
	type TableInfo struct {
		Name    string       `json:"name"`
		Columns []ColumnInfo `json:"columns"`
	}

	r.GET("/schema", func(c *gin.Context) {
		domainName := c.DefaultQuery("domain", "hris")
		domainInfo, err := getDomainDB(domainName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tables := []TableInfo{}

		tblRows, err := domainInfo.DB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca schema"})
			return
		}
		defer tblRows.Close()

		for tblRows.Next() {
			var tableName string
			if err := tblRows.Scan(&tableName); err != nil {
				continue
			}

			colRows, err := domainInfo.DB.Query(fmt.Sprintf("PRAGMA table_info('%s')", tableName))
			if err != nil {
				continue
			}

			var cols []ColumnInfo
			for colRows.Next() {
				var cid int
				var colName, colType string
				var notnull int
				var dfltValue interface{}
				var pk int
				if err := colRows.Scan(&cid, &colName, &colType, &notnull, &dfltValue, &pk); err != nil {
					continue
				}
				cols = append(cols, ColumnInfo{Name: colName, Type: colType, PK: pk > 0})
			}
			colRows.Close()

			tables = append(tables, TableInfo{Name: tableName, Columns: cols})
		}

		c.JSON(http.StatusOK, gin.H{"domain": domainName, "tables": tables})
	})

	// 6. Endpoint: Explain SQL (per domain)
	type ExplainRequest struct {
		SQL      string `json:"sql" binding:"required"`
		Question string `json:"question"`
		Domain   string `json:"domain"`
	}

	r.POST("/explain", func(c *gin.Context) {
		var req ExplainRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON tidak valid atau sql kosong"})
			return
		}

		if req.Domain == "" {
			req.Domain = "hris"
		}

		explainReqBody := map[string]string{
			"sql":      req.SQL,
			"question": req.Question,
			"domain":   req.Domain,
		}

		jsonData, _ := json.Marshal(explainReqBody)
		aiResp, err := http.Post(aiServiceURL+"/explain-sql", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Gagal terhubung ke AI Service"})
			return
		}
		defer aiResp.Body.Close()

		bodyBytes, _ := io.ReadAll(aiResp.Body)
		if aiResp.StatusCode != 200 {
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("AI Service error (status %d)", aiResp.StatusCode)})
			return
		}

		var result map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal parse response AI"})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	// 7. Benchmark Execute Endpoint — untuk evaluasi expected SQL (per domain)
	type BenchmarkExecuteRequest struct {
		SQL    string `json:"sql" binding:"required"`
		Domain string `json:"domain"`
	}

	r.POST("/benchmark/execute", func(c *gin.Context) {
		var req BenchmarkExecuteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON tidak valid atau sql kosong"})
			return
		}

		if req.Domain == "" {
			req.Domain = "hris"
		}

		domainInfo, err := getDomainDB(req.Domain)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, queryErr := executeQueryOnDomain(domainInfo, req.SQL)
		if queryErr != nil {
			c.JSON(http.StatusOK, gin.H{
				"data":   []map[string]interface{}{},
				"error":  queryErr.Error(),
				"domain": req.Domain,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":   result,
			"error":  nil,
			"domain": req.Domain,
		})
	})

	// 8. Endpoint Utama: /ask (per domain)
	r.POST("/ask", func(c *gin.Context) {
		startTime := time.Now()

		var req AskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format JSON tidak valid atau question kosong"})
			return
		}

		// Default domain
		if req.Domain == "" {
			req.Domain = "hris"
		}

		// Ambil domain info
		domainInfo, err := getDomainDB(req.Domain)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// --- FASE 1: BERTANYA KE AI PYTHON ---
		aiReqBody := AIRequest{
			SchemaContext: domainInfo.Schema,
			Question:      req.Question,
			Domain:        req.Domain,
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

		// --- BERSIHKAN OUTPUT MODEL ---
		generatedSQL = cleanSQLQuery(generatedSQL)
		log.Printf("DEBUG: SQL setelah dibersihkan: %s", generatedSQL)

		// --- VALIDASI SQL READ-ONLY ---
		if !isReadOnlySQL(generatedSQL) {
			log.Printf("WARNING: AI menghasilkan non-SELECT query: %s", generatedSQL)
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Query ditolak: hanya SELECT yang diizinkan",
				"generated_sql": generatedSQL,
			})
			return
		}

		// --- FASE 2: EKSEKUSI SQL KE DATABASE DOMAIN ---
		generatedSQL = ensureLimit(generatedSQL, 100)

		finalResult, queryErr := executeQueryOnDomain(domainInfo, generatedSQL)

		// --- RETRY LOGIC: Jika gagal, minta AI perbaiki query ---
		if queryErr != nil {
			log.Printf("WARNING: SQL pertama gagal: %s | Error: %v. Mencoba retry...", generatedSQL, queryErr)

			retryReq := AIRetryRequest{
				SchemaContext: domainInfo.Schema,
				Question:      req.Question,
				PrevSQL:       generatedSQL,
				ErrorMsg:      queryErr.Error(),
				Domain:        req.Domain,
			}

			retrySQL, retryErr := callAIService(aiServiceURL, retryReq)
			if retryErr != nil {
				log.Printf("ERROR: Retry juga gagal: %v", retryErr)
				c.JSON(http.StatusBadRequest, gin.H{
					"error":         "AI menghasilkan SQL yang tidak valid (retry gagal)",
					"generated_sql": generatedSQL,
					"db_message":    queryErr.Error(),
				})
				return
			}

			retrySQL = cleanSQLQuery(retrySQL)
			if !isReadOnlySQL(retrySQL) {
				log.Printf("WARNING: AI retry menghasilkan non-SELECT query: %s", retrySQL)
				c.JSON(http.StatusForbidden, gin.H{
					"error":         "Query ditolak: hanya SELECT yang diizinkan (setelah retry)",
					"generated_sql": retrySQL,
				})
				return
			}

			retrySQL = ensureLimit(retrySQL, 100)
			finalResult, queryErr = executeQueryOnDomain(domainInfo, retrySQL)
			if queryErr != nil {
				log.Printf("ERROR: Retry SQL juga gagal dieksekusi: %s | Error: %v", retrySQL, queryErr)
				c.JSON(http.StatusBadRequest, gin.H{
					"error":         "AI menghasilkan SQL yang tidak valid (setelah retry)",
					"generated_sql": retrySQL,
					"original_sql":  generatedSQL,
					"db_message":    queryErr.Error(),
				})
				return
			}
			generatedSQL = retrySQL
		}

		// --- FASE 3: GENERATE AI INSIGHT ---
		var insightResponse *AIInsightResponse
		if len(finalResult) > 0 {
			insightReq := AIInsightRequest{
				Question: req.Question,
				SQLQuery: generatedSQL,
				Data:     finalResult,
				Domain:   req.Domain,
			}
			insight, insightErr := callAIInsightService(aiServiceURL, insightReq)
			if insightErr != nil {
				log.Printf("WARNING: Gagal generate insight (non-fatal): %v", insightErr)
				// Insight is optional — don't fail the whole request
			} else {
				insightResponse = insight
			}
		}

		// --- FASE 4: KEMBALIKAN HASIL KE USER ---
		elapsed := time.Since(startTime)
		response := gin.H{
			"question":      req.Question,
			"domain":        req.Domain,
			"generated_sql": generatedSQL,
			"data":          finalResult,
			"response_time": fmt.Sprintf("%.0fms", float64(elapsed.Milliseconds())),
			"response_ms":   elapsed.Milliseconds(),
		}

		if insightResponse != nil {
			response["insight_summary"] = insightResponse.InsightSummary
			response["business_explanation"] = insightResponse.BusinessExplanation
			response["top_findings"] = insightResponse.TopFindings
		}

		c.JSON(http.StatusOK, response)
	})

	log.Printf("Backend Golang berjalan di http://localhost:%s", backendPort)
	log.Printf("AI Service URL: %s", aiServiceURL)
	log.Printf("Domains terdaftar: %d", len(domains))
	r.Run(":" + backendPort)
}