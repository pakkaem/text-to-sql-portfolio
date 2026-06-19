#!/usr/bin/env python3
"""
Text-to-SQL Evaluation Benchmark Runner
========================================
Evaluates the accuracy of the Text-to-SQL AI system by running benchmark questions
against the backend service and comparing results.

Usage:
    python run_benchmark.py                              # Run ALL domains (HRIS + Smart City)
    python run_benchmark.py --domain hris                # Run HRIS only
    python run_benchmark.py --domain smartcity           # Run Smart City only
    python run_benchmark.py --domain all                 # Run ALL domains explicitly

Prerequisites:
    - Backend service running on http://localhost:8080 (or specify --backend-url)
    - AI service running and connected to backend
    - Database seeded with data for the chosen domain

Domains:
    hris       - HRIS database (employees, departments, payroll, etc.)
    smartcity  - Smart City database (districts, cameras, traffic, violations, etc.)
"""

import json
import time
import argparse
import sys
import os
from datetime import datetime
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError

# ─── Configuration ───────────────────────────────────────────────────────────

DEFAULT_BACKEND_URL = "http://localhost:8080"
DEFAULT_TIMEOUT = 60  # seconds per question
DEFAULT_DELAY = 10.0  # seconds between questions (rate limit protection)
DEFAULT_DOMAIN = "all"
ALL_DOMAINS = ["hris", "smartcity"]
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
BENCHMARK_FILES = {
    "hris": os.path.join(SCRIPT_DIR, "benchmark.json"),
    "smartcity": os.path.join(SCRIPT_DIR, "benchmark-smartcity.json"),
}
DEFAULT_OUTPUT = os.path.join(SCRIPT_DIR, "results.json")
DOMAIN_OUTPUT = {
    "hris": os.path.join(SCRIPT_DIR, "results.json"),
    "smartcity": os.path.join(SCRIPT_DIR, "results-smartcity.json"),
}


# ─── Helpers ─────────────────────────────────────────────────────────────────

def load_benchmark(path: str) -> dict:
    """Load benchmark questions from JSON file."""
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def ask_question(backend_url: str, question: str, timeout: int, domain: str = "hris") -> dict:
    """
    Send a question to the backend /ask endpoint.
    Returns a dict with keys: generated_sql, data, response_ms, error
    """
    url = f"{backend_url}/ask"
    payload = json.dumps({"question": question, "domain": domain}).encode("utf-8")
    req = Request(url, data=payload, headers={"Content-Type": "application/json"}, method="POST")

    start = time.time()
    try:
        with urlopen(req, timeout=timeout) as resp:
            body = json.loads(resp.read().decode("utf-8"))
            elapsed_ms = int((time.time() - start) * 1000)
            return {
                "generated_sql": body.get("generated_sql", ""),
                "data": body.get("data", []),
                "response_ms": body.get("response_ms", elapsed_ms),
                "error": None,
            }
    except HTTPError as e:
        elapsed_ms = int((time.time() - start) * 1000)
        error_body = ""
        try:
            error_body = e.read().decode("utf-8")
        except Exception:
            pass
        return {
            "generated_sql": "",
            "data": [],
            "response_ms": elapsed_ms,
            "error": f"HTTP {e.code}: {error_body}",
        }
    except (URLError, TimeoutError, Exception) as e:
        elapsed_ms = int((time.time() - start) * 1000)
        return {
            "generated_sql": "",
            "data": [],
            "response_ms": elapsed_ms,
            "error": str(e),
        }


def normalize_sql(sql: str) -> str:
    """Normalize SQL for comparison: lowercase, strip whitespace, remove trailing semicolons."""
    return " ".join(sql.strip().lower().rstrip(";").split())


def normalize_data(data: list) -> set:
    """
    Convert result data to a set of frozensets for order-independent comparison.
    Each row becomes a frozenset of (key, value) tuples.
    """
    result = set()
    for row in data:
        if isinstance(row, dict):
            # Normalize values to strings for comparison
            normalized = frozenset((k, str(v)) for k, v in row.items())
            result.add(normalized)
    return result


def check_sql_similarity(generated: str, expected: str) -> dict:
    """
    Compare generated SQL with expected SQL.
    Returns similarity metrics.
    """
    gen_norm = normalize_sql(generated)
    exp_norm = normalize_sql(expected)

    # Exact match
    exact_match = gen_norm == exp_norm

    # Check if key SQL keywords/patterns are present
    gen_upper = gen_norm.upper()
    exp_upper = exp_norm.upper()

    # Extract key components from expected SQL
    keywords_found = []
    keywords_missing = []

    # Check for key tables mentioned
    important_words = []
    for word in exp_upper.split():
        if word not in ("SELECT", "FROM", "WHERE", "AND", "OR", "ON", "AS",
                         "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "GROUP",
                         "BY", "ORDER", "HAVING", "LIMIT", "DISTINCT", "*",
                         "DESC", "ASC", "NOT", "IN", "LIKE", "=", ">", "<",
                         ">=", "<=", "!=", "<>", "(", ")", ",", ";", "UNION",
                         "ALL", "BETWEEN", "IS", "NULL", "CASE", "WHEN", "THEN",
                         "ELSE", "END", "WITH", "OVER", "PARTITION", "RANK",
                         "ROW_NUMBER", "SUM", "COUNT", "AVG", "MAX", "MIN"):
            if len(word) > 2:  # Skip very short tokens
                important_words.append(word)

    for word in set(important_words):
        if word in gen_upper:
            keywords_found.append(word)
        else:
            keywords_missing.append(word)

    total_keywords = len(set(important_words))
    keyword_match_ratio = len(keywords_found) / total_keywords if total_keywords > 0 else 1.0

    return {
        "exact_match": exact_match,
        "keyword_match_ratio": round(keyword_match_ratio, 3),
        "keywords_found": keywords_found,
        "keywords_missing": keywords_missing,
    }


def execute_expected_sql(backend_url: str, expected_sql: str, timeout: int, domain: str = "hris") -> list:
    """
    Execute the expected SQL via /benchmark/execute to get ground truth results.
    Returns the data rows.
    """
    url = f"{backend_url}/benchmark/execute"
    payload = json.dumps({"sql": expected_sql, "domain": domain}).encode("utf-8")
    req = Request(url, data=payload, headers={"Content-Type": "application/json"}, method="POST")

    try:
        with urlopen(req, timeout=timeout) as resp:
            body = json.loads(resp.read().decode("utf-8"))
            return body.get("data", [])
    except Exception as e:
        print(f"           ⚠️  Failed to execute expected SQL: {e}")
        return []


def normalize_row_values(row: dict) -> dict:
    """
    Normalize values in a row dict to comparable string representations.
    Handles None, numeric types, and whitespace.
    """
    normalized = {}
    for k, v in row.items():
        key = k.lower().strip()
        if v is None:
            normalized[key] = "null"
        elif isinstance(v, float):
            # Normalize floats: e.g. 25000000.0 == 25000000
            normalized[key] = f"{v:.10f}".rstrip("0").rstrip(".")
        else:
            normalized[key] = str(v).strip().lower()
    return normalized


def compare_results(expected_data: list, generated_data: list) -> str:
    """
    Compare expected and generated query results.
    Returns: 'exact_match', 'partial_match', or 'mismatch'.

    - exact_match:   same data (order-independent)
    - partial_match: some overlap in rows or values
    - mismatch:      completely different results
    """
    if not expected_data and not generated_data:
        return "exact_match"  # Both empty = match

    if not expected_data or not generated_data:
        # One is empty, the other is not
        if not expected_data and generated_data:
            return "mismatch"  # Expected nothing but got something
        return "mismatch"  # Expected something but got nothing

    # Normalize all rows
    exp_normalized = [normalize_row_values(row) for row in expected_data]
    gen_normalized = [normalize_row_values(row) for row in generated_data]

    # Convert to sets of frozensets for order-independent comparison
    exp_set = set(frozenset(row.items()) for row in exp_normalized)
    gen_set = set(frozenset(row.items()) for row in gen_normalized)

    # Exact match: all expected rows found in generated, and vice versa
    if exp_set == gen_set:
        return "exact_match"

    # Partial match: some overlap
    overlap = exp_set & gen_set
    if len(overlap) > 0:
        return "partial_match"

    # For single-value results (like COUNT, MAX, MIN, AVG, SUM),
    # compare the actual values regardless of column name
    if len(exp_normalized) == 1 and len(gen_normalized) == 1:
        exp_vals = set(exp_normalized[0].values())
        gen_vals = set(gen_normalized[0].values())
        if exp_vals == gen_vals:
            return "exact_match"

    return "mismatch"


# ─── Main Benchmark Runner ──────────────────────────────────────────────────

def run_benchmark(backend_url: str, timeout: int, output_path: str, verbose: bool = True, delay: float = DEFAULT_DELAY, domain: str = DEFAULT_DOMAIN):
    """Run the full benchmark suite."""

    # Resolve benchmark file for domain
    if domain not in BENCHMARK_FILES:
        print(f"\n❌ Unknown domain '{domain}'. Available: {', '.join(BENCHMARK_FILES.keys())}")
        sys.exit(1)

    benchmark_path = BENCHMARK_FILES[domain]

    # Load benchmark
    print(f"\n{'='*70}")
    print("  TEXT-TO-SQL EVALUATION BENCHMARK")
    print(f"{'='*70}")

    benchmark = load_benchmark(benchmark_path)
    questions = benchmark["questions"]
    total = len(questions)

    print(f"\n  Domain:      {domain}")
    print(f"  Questions:   {total}")
    print(f"  Backend:     {backend_url}")
    print(f"  Timeout:     {timeout}s per question")
    print(f"  Delay:       {delay}s between questions (rate limit protection)")
    print(f"  Est. time:   ~{(timeout + delay) * total / 60:.0f} minutes")
    print(f"{'='*70}\n")

    # Check backend health first
    print("⏳ Checking backend health...")
    try:
        health_req = Request(f"{backend_url}/health", method="GET")
        with urlopen(health_req, timeout=10) as resp:
            health = json.loads(resp.read().decode("utf-8"))
            db_statuses = health.get("databases") or {}
            if isinstance(db_statuses, dict) and db_statuses:
                connected = all(status == "connected" for status in db_statuses.values())
                status_line = "✅" if connected else "❌"
                print(f"  Backend:   {status_line} DB ({', '.join(f'{name}:{status}' for name, status in db_statuses.items())})")
            else:
                connected = health.get("db") == "connected"
                print(f"  Backend:   {'✅' if connected else '❌'} DB")

            print(f"  AI Service: {'✅' if health.get('ai_service') else '❌'} AI")
            if not connected:
                print("\n❌ Database not connected. Aborting.")
                sys.exit(1)
    except Exception as e:
        print(f"\n❌ Cannot connect to backend: {e}")
        print(f"   Make sure backend is running at {backend_url}")
        sys.exit(1)

    results = []
    passed = 0
    failed = 0
    errors = 0
    total_time_ms = 0

    # Per-category/difficulty tracking
    stats = {
        "by_category": {},
        "by_difficulty": {},
    }

    for i, q in enumerate(questions, 1):
        qid = q["id"]
        question = q["question"]
        expected_sql = q["expected_sql"]
        category = q["category"]
        difficulty = q["difficulty"]

        # Init stats buckets
        if category not in stats["by_category"]:
            stats["by_category"][category] = {"total": 0, "executed": 0, "matched": 0}
        if difficulty not in stats["by_difficulty"]:
            stats["by_difficulty"][difficulty] = {"total": 0, "executed": 0, "matched": 0}

        stats["by_category"][category]["total"] += 1
        stats["by_difficulty"][difficulty]["total"] += 1

        # Progress
        progress = f"[{i:2d}/{total}]"
        print(f"{progress} Q{qid:02d} ({difficulty:6s}) {question[:55]}...", end=" ", flush=True)

        # Ask the backend
        response = ask_question(backend_url, question, timeout, domain=domain)
        generated_sql = response["generated_sql"]
        response_ms = response["response_ms"]
        error = response["error"]
        data = response["data"]

        total_time_ms += response_ms

        # Evaluate
        execution_success = error is None and generated_sql != ""
        sql_similarity = check_sql_similarity(generated_sql, expected_sql) if execution_success else {}

        # Execute expected SQL to get ground truth results
        expected_data = execute_expected_sql(backend_url, expected_sql, timeout, domain=domain)

        # Compare generated results with expected results (data-level accuracy)
        data_match = "mismatch"
        if execution_success:
            data_match = compare_results(expected_data, data)

        status = ""
        if not execution_success:
            status = "❌ ERROR"
            errors += 1
        elif data_match == "exact_match":
            # SQL matches in result data — truly correct
            status = "✅ EXACT"
            passed += 1
            stats["by_category"][category]["executed"] += 1
            stats["by_category"][category]["matched"] += 1
            stats["by_difficulty"][difficulty]["executed"] += 1
            stats["by_difficulty"][difficulty]["matched"] += 1
        elif data_match == "partial_match":
            # Some data overlap — partially correct
            status = "🟡 GOOD"
            passed += 1
            stats["by_category"][category]["executed"] += 1
            stats["by_category"][category]["matched"] += 1
            stats["by_difficulty"][difficulty]["executed"] += 1
            stats["by_difficulty"][difficulty]["matched"] += 1
        elif data_match == "mismatch":
            # Executed but data is completely different — false positive caught!
            status = "🟠 MISMATCH"
            failed += 1
            stats["by_category"][category]["executed"] += 1
            stats["by_difficulty"][difficulty]["executed"] += 1

        print(f"{status} ({response_ms}ms)")

        if verbose and not execution_success:
            print(f"           Error: {error[:80]}")
        elif verbose and data_match == "mismatch" and execution_success:
            print(f"           ⚠️  Result mismatch! Expected {len(expected_data)} rows, got {len(data)}")
        elif verbose and sql_similarity.get("keywords_missing"):
            print(f"           Missing: {', '.join(sql_similarity['keywords_missing'][:5])}")

        # Record result
        results.append({
            "id": qid,
            "category": category,
            "difficulty": difficulty,
            "question": question,
            "expected_sql": expected_sql,
            "generated_sql": generated_sql,
            "execution_success": execution_success,
            "data_match": data_match,
            "expected_data_count": len(expected_data),
            "generated_data_count": len(data),
            "sql_similarity": sql_similarity,
            "response_ms": response_ms,
            "error": error,
        })

        # Rate limit protection: delay between questions (skip after last question)
        if i < total and delay > 0:
            print(f"           ⏳ Waiting {delay:.0f}s...", flush=True)
            time.sleep(delay)

    # ─── Summary ─────────────────────────────────────────────────────────────

    avg_time = total_time_ms / total if total > 0 else 0
    execution_accuracy = ((total - errors) / total * 100) if total > 0 else 0
    result_accuracy = (passed / total * 100) if total > 0 else 0

    print(f"\n{'='*70}")
    print("  RESULTS SUMMARY")
    print(f"{'='*70}")
    print(f"\n  Total Questions:    {total}")
    print(f"  Executed (no error):{total - errors} ({execution_accuracy:.1f}%)")
    print(f"  Passed (≥70% match):{passed} ({result_accuracy:.1f}%)")
    print(f"  Exact Match:        {sum(1 for r in results if r['sql_similarity'].get('exact_match'))}")
    print(f"  Errors:             {errors}")
    print(f"\n  Avg Response Time:  {avg_time:.0f}ms")
    print(f"  Total Time:         {total_time_ms / 1000:.1f}s")

    # Per difficulty
    print(f"\n  {'Difficulty':<12} {'Total':>6} {'Executed':>9} {'Match':>6} {'Exec %':>7} {'Match %':>8}")
    print(f"  {'-'*50}")
    for diff in ["easy", "medium", "hard"]:
        if diff in stats["by_difficulty"]:
            s = stats["by_difficulty"][diff]
            exec_pct = (s["executed"] / s["total"] * 100) if s["total"] > 0 else 0
            match_pct = (s["matched"] / s["total"] * 100) if s["total"] > 0 else 0
            print(f"  {diff:<12} {s['total']:>6} {s['executed']:>9} {s['matched']:>6} {exec_pct:>6.1f}% {match_pct:>7.1f}%")

    # Per category
    print(f"\n  {'Category':<20} {'Total':>6} {'Executed':>9} {'Match':>6} {'Exec %':>7} {'Match %':>8}")
    print(f"  {'-'*58}")
    for cat in ["simple_select", "filter_where", "aggregation", "join", "complex", "advanced"]:
        if cat in stats["by_category"]:
            s = stats["by_category"][cat]
            exec_pct = (s["executed"] / s["total"] * 100) if s["total"] > 0 else 0
            match_pct = (s["matched"] / s["total"] * 100) if s["total"] > 0 else 0
            print(f"  {cat:<20} {s['total']:>6} {s['executed']:>9} {s['matched']:>6} {exec_pct:>6.1f}% {match_pct:>7.1f}%")

    # ─── Save Results ────────────────────────────────────────────────────────

    report = {
        "metadata": {
            "benchmark_version": benchmark["metadata"]["version"],
            "domain": domain,
            "run_date": datetime.now().isoformat(),
            "backend_url": backend_url,
            "total_questions": total,
        },
        "summary": {
            "execution_accuracy_pct": round(execution_accuracy, 1),
            "result_accuracy_pct": round(result_accuracy, 1),
            "exact_match_count": sum(1 for r in results if r["sql_similarity"].get("exact_match")),
            "error_count": errors,
            "avg_response_ms": round(avg_time, 0),
            "total_time_ms": total_time_ms,
        },
        "stats_by_difficulty": stats["by_difficulty"],
        "stats_by_category": stats["by_category"],
        "results": results,
    }

    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(report, f, indent=2, ensure_ascii=False)

    print(f"\n  📄 Detailed results saved to: {output_path}")
    print(f"{'='*70}\n")

    return report


# ─── CLI Entry Point ─────────────────────────────────────────────────────────

if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Text-to-SQL Evaluation Benchmark Runner",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python run_benchmark.py
  python run_benchmark.py --domain smartcity
  python run_benchmark.py --backend-url http://localhost:8080 --domain hris
  python run_benchmark.py --output results.json --timeout 120
  python run_benchmark.py --quiet   # minimal output
        """,
    )
    parser.add_argument(
        "--backend-url",
        default=DEFAULT_BACKEND_URL,
        help=f"Backend service URL (default: {DEFAULT_BACKEND_URL})",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=DEFAULT_TIMEOUT,
        help=f"Timeout per question in seconds (default: {DEFAULT_TIMEOUT})",
    )
    parser.add_argument(
        "--output",
        default=DEFAULT_OUTPUT,
        help=f"Output JSON file path (default: domain-specific output)",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="Minimal output (only show summary)",
    )
    parser.add_argument(
        "--delay",
        type=float,
        default=DEFAULT_DELAY,
        help=f"Delay between questions in seconds (default: {DEFAULT_DELAY}s, set to 0 to disable)",
    )
    parser.add_argument(
        "--domain",
        default=DEFAULT_DOMAIN,
        choices=list(BENCHMARK_FILES.keys()) + ["all"],
        help=f"Domain to benchmark (default: {DEFAULT_DOMAIN})",
    )

    args = parser.parse_args()
    verbose = not args.quiet

    domains_to_run = ALL_DOMAINS if args.domain == "all" else [args.domain]

    # Print overall header when running all domains
    if len(domains_to_run) > 1:
        print(f"\n{'#'*70}")
        print("  MULTI-DOMAIN BENCHMARK RUNNER")
        print(f"  Domains to run: {', '.join(domains_to_run)}")
        print(f"  Results will be saved to domain-specific files.")
        print(f"{'#'*70}")

    all_reports = {}
    for dom in domains_to_run:
        output = DOMAIN_OUTPUT.get(dom, DEFAULT_OUTPUT)
        report = run_benchmark(
            backend_url=args.backend_url,
            timeout=args.timeout,
            output_path=output,
            verbose=verbose,
            delay=args.delay,
            domain=dom,
        )
        all_reports[dom] = report

    # Print combined summary when running all domains
    if len(all_reports) > 1:
        print(f"\n{'#'*70}")
        print("  COMBINED SUMMARY — ALL DOMAINS")
        print(f"{'#'*70}")
        print(f"\n  {'Domain':<15} {'Questions':>9} {'Exec %':>8} {'Result %':>9} {'Exact':>7} {'Errors':>8} {'Avg (ms)':>10}")
        print(f"  {'-'*68}")
        for dom, rpt in all_reports.items():
            s = rpt["summary"]
            print(f"  {dom:<15} {rpt['metadata']['total_questions']:>9} "
                  f"{s['execution_accuracy_pct']:>7.1f}% "
                  f"{s['result_accuracy_pct']:>8.1f}% "
                  f"{s['exact_match_count']:>7} "
                  f"{s['error_count']:>8} "
                  f"{s['avg_response_ms']:>9.0f}")

        total_q = sum(r["metadata"]["total_questions"] for r in all_reports.values())
        total_errors = sum(r["summary"]["error_count"] for r in all_reports.values())
        total_passed_pct = sum(r["summary"]["result_accuracy_pct"] for r in all_reports.values()) / len(all_reports)
        print(f"\n  Total questions across all domains: {total_q}")
        print(f"  Average result accuracy: {total_passed_pct:.1f}%")
        print(f"  Total errors: {total_errors}")
        print(f"{'#'*70}\n")
