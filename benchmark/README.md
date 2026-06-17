# 📊 Evaluation Benchmark Results

## How to Run

```bash
# 1. Start all services first
cd ai-service && python app.py          # Terminal 1 (port 8000)
cd backend-service && go run .          # Terminal 2 (port 8080)
cd frontend-service && npm run dev      # Terminal 3 (port 5173) [optional]

# 2. Run benchmark
cd benchmark
python run_benchmark.py

# 3. With custom options
python run_benchmark.py --backend-url http://localhost:8080 --timeout 120 --output results.json
```

## Benchmark Structure

| Category | Count | Difficulty | Description |
|----------|-------|------------|-------------|
| `simple_select` | 10 | Easy | Basic SELECT, COUNT, DISTINCT |
| `filter_where` | 8 | Easy | WHERE clauses, comparisons, LIKE |
| `aggregation` | 10 | Medium | GROUP BY, SUM, AVG, MAX, MIN |
| `join` | 10 | Medium | Single & multi-table JOINs |
| `complex` | 7 | Hard | HAVING, subqueries, ORDER BY + LIMIT |
| `advanced` | 5 | Hard | Window functions, CTEs, correlated subqueries |

**Total: 50 questions**

## Metrics Explained

| Metric | Description |
|--------|-------------|
| **Execution Accuracy** | % of questions where AI generated a valid SQL that executed without error |
| **Result Accuracy** | % of questions where generated SQL matched expected SQL (≥70% keyword match) |
| **Exact Match** | % of questions where generated SQL exactly matches expected SQL |
| **Avg Response Time** | Average time from question submission to result delivery |

## Results

> Run the benchmark and paste results here. Example format below.

### Mode: Groq (llama-3.1-8b-instant)

| Metric | Value |
|--------|-------|
| Execution Accuracy | `XX.X%` |
| Result Accuracy | `XX.X%` |
| Exact Match | `XX / 50` |
| Avg Response Time | `X,XXXms` |
| Total Time | `XX.Xs` |

#### Per Difficulty

| Difficulty | Executed | Matched | Exec % | Match % |
|------------|----------|---------|--------|---------|
| Easy | `XX/18` | `XX/18` | `XX.X%` | `XX.X%` |
| Medium | `XX/20` | `XX/20` | `XX.X%` | `XX.X%` |
| Hard | `XX/12` | `XX/12` | `XX.X%` | `XX.X%` |

#### Per Category

| Category | Executed | Matched | Exec % | Match % |
|----------|----------|---------|--------|---------|
| simple_select | `XX/10` | `XX/10` | `XX.X%` | `XX.X%` |
| filter_where | `XX/8` | `XX/8` | `XX.X%` | `XX.X%` |
| aggregation | `XX/10` | `XX/10` | `XX.X%` | `XX.X%` |
| join | `XX/10` | `XX/10` | `XX.X%` | `XX.X%` |
| complex | `XX/7` | `XX/7` | `XX.X%` | `XX.X%` |
| advanced | `XX/5` | `XX/5` | `XX.X%` | `XX.X%` |

---

### Comparative Results (Multi-Mode)

| Mode | Execution Accuracy | Result Accuracy | Avg Response Time |
|------|-------------------|-----------------|-------------------|
| **Groq** (cloud) | `XX.X%` | `XX.X%` | `X.Xs` |
| **Zero-Shot** (local) | `XX.X%` | `XX.X%` | `XXs` |
| **LoRA** (local) | `XX.X%` | `XX.X%` | `XXs` |

---

## Output Files

- `results.json` — Full detailed results with per-question breakdown
- `benchmark.json` — Benchmark question definitions (50 questions)

## Notes

- The benchmark uses **keyword matching** for SQL comparison, not exact string matching. This is because the same question can have multiple valid SQL solutions (e.g., different JOIN styles, column aliasing).
- **≥70% keyword match** is considered a "pass" — the generated SQL covers most of the expected tables, columns, and conditions.
- Empty result sets from valid queries are counted as successful executions (some questions legitimately return no data).