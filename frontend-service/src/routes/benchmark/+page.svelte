<svelte:head>
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet" />
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4/dist/chart.umd.min.js"></script>
</svelte:head>

<script>
  // @ts-nocheck
  import { onMount } from "svelte";

  let { data } = $props();
  let report = $derived(data.report);

  // Chart instances (for cleanup)
  let accuracyChart, categoryChart, difficultyChart, timeChart, matchBreakdownChart;

  // Filters for detailed table
  let filterCategory = $state("all");
  let filterStatus = $state("all");
  let filterDifficulty = $state("all");
  let sortBy = $state("id");
  let sortDir = $state("asc");
  let expandedRow = $state(null);

  // Dark mode
  let darkMode = $state(false);
  function toggleDarkMode() {
    darkMode = !darkMode;
    document.documentElement.setAttribute("data-theme", darkMode ? "dark" : "light");
    localStorage.setItem("hris-dark-mode", darkMode ? "1" : "0");
  }

  // Derived: filtered & sorted results
  let filteredResults = $derived.by(() => {
    if (!report?.results) return [];
    let rows = [...report.results];
    if (filterCategory !== "all") rows = rows.filter(r => r.category === filterCategory);
    if (filterDifficulty !== "all") rows = rows.filter(r => r.difficulty === filterDifficulty);
    if (filterStatus !== "all") {
      if (filterStatus === "exact") rows = rows.filter(r => r.data_match === "exact_match");
      else if (filterStatus === "good") rows = rows.filter(r => r.data_match === "partial_match");
      else if (filterStatus === "mismatch") rows = rows.filter(r => r.data_match === "mismatch");
      else if (filterStatus === "error") rows = rows.filter(r => !r.execution_success);
    }
    rows.sort((a, b) => {
      let va = a[sortBy], vb = b[sortBy];
      if (typeof va === "string") va = va.toLowerCase();
      if (typeof vb === "string") vb = vb.toLowerCase();
      if (va < vb) return sortDir === "asc" ? -1 : 1;
      if (va > vb) return sortDir === "asc" ? 1 : -1;
      return 0;
    });
    return rows;
  });

  function toggleSort(field) {
    if (sortBy === field) {
      sortDir = sortDir === "asc" ? "desc" : "asc";
    } else {
      sortBy = field;
      sortDir = "asc";
    }
  }

  function statusBadge(dataMatch, executionSuccess) {
    if (!executionSuccess) return { label: "ERROR", class: "badge-error" };
    if (dataMatch === "exact_match") return { label: "EXACT", class: "badge-exact" };
    if (dataMatch === "partial_match") return { label: "GOOD", class: "badge-good" };
    return { label: "MISMATCH", class: "badge-mismatch" };
  }

  function difficultyBadge(d) {
    if (d === "easy") return { label: "Easy", class: "diff-easy" };
    if (d === "medium") return { label: "Medium", class: "diff-medium" };
    return { label: "Hard", class: "diff-hard" };
  }

  const categoryLabels = {
    simple_select: "Simple SELECT",
    filter_where: "Filter / WHERE",
    aggregation: "Aggregation",
    join: "JOIN",
    complex: "Complex",
    advanced: "Advanced",
  };

  const chartColors = {
    exact: "#10b981",
    partial: "#f59e0b",
    mismatch: "#f97316",
    error: "#ef4444",
    blue: "#3b82f6",
    purple: "#8b5cf6",
    slate: "#64748b",
  };

  const palette6 = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#06b6d4"];

  onMount(() => {
    // Restore dark mode
    const saved = localStorage.getItem("hris-dark-mode");
    if (saved === "1") {
      darkMode = true;
      document.documentElement.setAttribute("data-theme", "dark");
    }

    if (!report?.results) return;

    const textColor = "#94a3b8";
    const gridColor = "rgba(148,163,184,0.1)";
    const defaults = { color: textColor, font: { family: "'Plus Jakarta Sans', sans-serif" } };
    Chart.defaults.color = defaults.color;
    Chart.defaults.font.family = defaults.font.family;

    // 1. Overall Accuracy Donut
    const summary = report.summary;
    const passed = summary.result_accuracy_pct;
    const failed = 100 - passed;
    accuracyChart = new Chart(document.getElementById("accuracyChart"), {
      type: "doughnut",
      data: {
        labels: ["Passed", "Failed"],
        datasets: [{
          data: [passed, failed],
          backgroundColor: [chartColors.exact, chartColors.error],
          borderWidth: 0,
          cutout: "72%",
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: { label: (ctx) => `${ctx.label}: ${ctx.parsed.toFixed(1)}%` }
          },
        },
      },
    });

    // 2. Accuracy by Category (horizontal bar)
    const catKeys = Object.keys(report.stats_by_category);
    const catLabels = catKeys.map(k => categoryLabels[k] || k);
    const catMatchPct = catKeys.map(k => {
      const s = report.stats_by_category[k];
      return s.total > 0 ? (s.matched / s.total * 100) : 0;
    });
    categoryChart = new Chart(document.getElementById("categoryChart"), {
      type: "bar",
      data: {
        labels: catLabels,
        datasets: [{
          label: "Match %",
          data: catMatchPct,
          backgroundColor: palette6,
          borderRadius: 6,
          barThickness: 24,
        }],
      },
      options: {
        indexAxis: "y",
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: {
            max: 100,
            ticks: { callback: (v) => v + "%" },
            grid: { color: gridColor },
          },
          y: { grid: { display: false } },
        },
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              label: (ctx) => {
                const k = catKeys[ctx.dataIndex];
                const s = report.stats_by_category[k];
                return `${s.matched}/${s.total} matched (${ctx.parsed.x.toFixed(1)}%)`;
              }
            }
          },
        },
      },
    });

    // 3. Accuracy by Difficulty (grouped bar: executed vs matched)
    const diffKeys = ["easy", "medium", "hard"];
    const diffLabels = ["Easy", "Medium", "Hard"];
    const diffExec = diffKeys.map(k => {
      const s = report.stats_by_difficulty[k];
      return s ? (s.executed / s.total * 100) : 0;
    });
    const diffMatch = diffKeys.map(k => {
      const s = report.stats_by_difficulty[k];
      return s ? (s.matched / s.total * 100) : 0;
    });
    difficultyChart = new Chart(document.getElementById("difficultyChart"), {
      type: "bar",
      data: {
        labels: diffLabels,
        datasets: [
          {
            label: "Executed %",
            data: diffExec,
            backgroundColor: "rgba(59,130,246,0.3)",
            borderColor: chartColors.blue,
            borderWidth: 1,
            borderRadius: 4,
          },
          {
            label: "Match %",
            data: diffMatch,
            backgroundColor: "rgba(16,185,129,0.3)",
            borderColor: chartColors.exact,
            borderWidth: 1,
            borderRadius: 4,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: {
            max: 100,
            ticks: { callback: (v) => v + "%" },
            grid: { color: gridColor },
          },
          x: { grid: { display: false } },
        },
        plugins: {
          tooltip: {
            callbacks: { label: (ctx) => `${ctx.dataset.label}: ${ctx.parsed.y.toFixed(1)}%` }
          },
        },
      },
    });

    // 4. Response Time Distribution (bar chart per question)
    const qIds = report.results.map(r => `Q${String(r.id).padStart(2, "0")}`);
    const times = report.results.map(r => r.response_ms);
    const timeColors = times.map(t => t < 1000 ? chartColors.exact : t < 2000 ? chartColors.partial : chartColors.error);
    timeChart = new Chart(document.getElementById("timeChart"), {
      type: "bar",
      data: {
        labels: qIds,
        datasets: [{
          label: "Response Time (ms)",
          data: times,
          backgroundColor: timeColors,
          borderRadius: 3,
          barThickness: 12,
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: {
            ticks: { callback: (v) => v + "ms" },
            grid: { color: gridColor },
          },
          x: {
            grid: { display: false },
            ticks: { maxRotation: 45, font: { size: 10 } },
          },
        },
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: { label: (ctx) => `${ctx.parsed.y}ms` }
          },
        },
      },
    });

    // 5. Match Breakdown by Category (stacked bar)
    const matchTypes = ["exact_match", "partial_match", "mismatch"];
    const matchLabels = ["Exact Match", "Partial Match", "Mismatch"];
    const matchColorMap = [chartColors.exact, chartColors.partial, chartColors.mismatch];
    const matchDatasets = matchTypes.map((mt, idx) => ({
      label: matchLabels[idx],
      data: catKeys.map(k => {
        return report.results.filter(r => r.category === k && r.data_match === mt).length;
      }),
      backgroundColor: matchColorMap[idx],
      borderRadius: 2,
    }));
    matchBreakdownChart = new Chart(document.getElementById("matchBreakdownChart"), {
      type: "bar",
      data: {
        labels: catLabels,
        datasets: matchDatasets,
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: { stacked: true, grid: { display: false } },
          y: { stacked: true, ticks: { stepSize: 1 }, grid: { color: gridColor } },
        },
        plugins: {
          tooltip: { mode: "index" },
        },
      },
    });
  });
</script>

<div class="app" class:dark={darkMode}>
  <!-- Header -->
  <header>
    <div class="header-top">
      <div class="header-left">
        <h1>📊 Benchmark Report</h1>
        <p class="subtitle">Text-to-SQL Evaluation Dashboard</p>
      </div>
      <div class="header-right">
        <button class="btn-icon" onclick={toggleDarkMode} title="Toggle dark mode">
          {darkMode ? "☀️" : "🌙"}
        </button>
        <a href="/" class="btn-link">← Back to App</a>
      </div>
    </div>
    {#if report?.metadata}
      <div class="meta-bar">
        <span>📅 {new Date(report.metadata.run_date).toLocaleDateString("id-ID", { year: "numeric", month: "long", day: "numeric", hour: "2-digit", minute: "2-digit" })}</span>
        <span>📋 {report.metadata.total_questions} Questions</span>
        <span>⚡ Avg {report.summary.avg_response_ms}ms</span>
        <span>⏱ Total {(report.summary.total_time_ms / 1000).toFixed(1)}s</span>
      </div>
    {/if}
  </header>

  {#if report?.error}
    <div class="card error-card">
      <p>⚠️ {report.error}</p>
      <p class="muted">Run the benchmark first: <code>python benchmark/run_benchmark.py</code></p>
    </div>
  {:else}
    <!-- KPI Cards -->
    <div class="kpi-grid">
      <div class="kpi-card kpi-passed">
        <div class="kpi-value">{report.summary.result_accuracy_pct}%</div>
        <div class="kpi-label">Result Accuracy</div>
        <div class="kpi-detail">{Math.round(report.summary.result_accuracy_pct * report.metadata.total_questions / 100)}/{report.metadata.total_questions} passed</div>
      </div>
      <div class="kpi-card kpi-exec">
        <div class="kpi-value">{report.summary.execution_accuracy_pct}%</div>
        <div class="kpi-label">Execution Rate</div>
        <div class="kpi-detail">{report.metadata.total_questions - report.summary.error_count} executed</div>
      </div>
      <div class="kpi-card kpi-time">
        <div class="kpi-value">{report.summary.avg_response_ms}ms</div>
        <div class="kpi-label">Avg Response</div>
        <div class="kpi-detail">Total: {(report.summary.total_time_ms / 1000).toFixed(1)}s</div>
      </div>
      <div class="kpi-card kpi-exact">
        <div class="kpi-value">{report.summary.exact_match_count}</div>
        <div class="kpi-label">Exact SQL Match</div>
        <div class="kpi-detail">Character-identical queries</div>
      </div>
    </div>

    <!-- Charts Row 1: Accuracy donut + Category bar -->
    <div class="charts-row">
      <div class="card chart-card chart-donut">
        <h3>Overall Accuracy</h3>
        <div class="donut-container">
          <canvas id="accuracyChart"></canvas>
          <div class="donut-center">
            <span class="donut-pct">{report.summary.result_accuracy_pct}%</span>
            <span class="donut-label">Passed</span>
          </div>
        </div>
      </div>
      <div class="card chart-card chart-bar-wide">
        <h3>Accuracy by Category</h3>
        <div class="chart-container-hbar">
          <canvas id="categoryChart"></canvas>
        </div>
      </div>
    </div>

    <!-- Charts Row 2: Difficulty + Match Breakdown -->
    <div class="charts-row">
      <div class="card chart-card">
        <h3>Accuracy by Difficulty</h3>
        <div class="chart-container">
          <canvas id="difficultyChart"></canvas>
        </div>
      </div>
      <div class="card chart-card">
        <h3>Match Breakdown by Category</h3>
        <div class="chart-container">
          <canvas id="matchBreakdownChart"></canvas>
        </div>
      </div>
    </div>

    <!-- Response Time Chart (full width) -->
    <div class="card chart-card-full">
      <h3>Response Time per Question</h3>
      <div class="chart-container-wide">
        <canvas id="timeChart"></canvas>
      </div>
    </div>

    <!-- Detailed Results Table -->
    <div class="card table-card">
      <div class="table-header">
        <h3>Detailed Results ({filteredResults.length} items)</h3>
        <div class="filters">
          <select bind:value={filterCategory}>
            <option value="all">All Categories</option>
            {#each Object.keys(report.stats_by_category) as cat}
              <option value={cat}>{categoryLabels[cat] || cat}</option>
            {/each}
          </select>
          <select bind:value={filterDifficulty}>
            <option value="all">All Difficulties</option>
            <option value="easy">Easy</option>
            <option value="medium">Medium</option>
            <option value="hard">Hard</option>
          </select>
          <select bind:value={filterStatus}>
            <option value="all">All Statuses</option>
            <option value="exact">✅ Exact</option>
            <option value="good">🟡 Good</option>
            <option value="mismatch">🟠 Mismatch</option>
            <option value="error">❌ Error</option>
          </select>
        </div>
      </div>

      <div class="table-container">
        <table>
          <thead>
            <tr>
              <th class="sortable" onclick={() => toggleSort("id")}>ID {sortBy === "id" ? (sortDir === "asc" ? "↑" : "↓") : ""}</th>
              <th class="sortable" onclick={() => toggleSort("difficulty")}>Difficulty {sortBy === "difficulty" ? (sortDir === "asc" ? "↑" : "↓") : ""}</th>
              <th class="sortable" onclick={() => toggleSort("category")}>Category {sortBy === "category" ? (sortDir === "asc" ? "↑" : "↓") : ""}</th>
              <th>Question</th>
              <th>Status</th>
              <th class="sortable" onclick={() => toggleSort("response_ms")}>Time {sortBy === "response_ms" ? (sortDir === "asc" ? "↑" : "↓") : ""}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredResults as r}
              {@const st = statusBadge(r.data_match, r.execution_success)}
              {@const df = difficultyBadge(r.difficulty)}
              <tr class="clickable-row" onclick={() => expandedRow = expandedRow === r.id ? null : r.id}>
                <td><span class="qid">Q{String(r.id).padStart(2, "0")}</span></td>
                <td><span class="badge {df.class}">{df.label}</span></td>
                <td>{categoryLabels[r.category] || r.category}</td>
                <td class="question-cell">{r.question}</td>
                <td><span class="badge {st.class}">{st.label}</span></td>
                <td class="time-cell">{r.response_ms}ms</td>
              </tr>
              {#if expandedRow === r.id}
                <tr class="expanded-row">
                  <td colspan="6">
                    <div class="sql-compare">
                      <div class="sql-block">
                        <h4>Expected SQL</h4>
                        <pre>{r.expected_sql}</pre>
                      </div>
                      <div class="sql-block" class:sql-error={!r.execution_success}>
                        <h4>Generated SQL</h4>
                        <pre>{r.generated_sql || "(no output)"}</pre>
                      </div>
                    </div>
                    <div class="match-details">
                      <span>Expected rows: {r.expected_data_count}</span>
                      <span>Generated rows: {r.generated_data_count}</span>
                      {#if r.error}
                        <span class="error-text">Error: {r.error}</span>
                      {/if}
                    </div>
                  </td>
                </tr>
              {/if}
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>

<style>
  :root {
    --bg: #f8fafc;
    --bg-card: #ffffff;
    --text-primary: #1e293b;
    --text-secondary: #475569;
    --text-muted: #94a3b8;
    --border: #e2e8f0;
    --hover-bg: #f1f5f9;
  }
  :global([data-theme="dark"]) {
    --bg: #0f172a;
    --bg-card: #1e293b;
    --text-primary: #f1f5f9;
    --text-secondary: #94a3b8;
    --text-muted: #64748b;
    --border: #334155;
    --hover-bg: #334155;
  }

  .app {
    max-width: 1200px;
    margin: 0 auto;
    padding: 24px;
    font-family: 'Plus Jakarta Sans', system-ui, sans-serif;
    color: var(--text-primary);
    background: var(--bg);
    min-height: 100vh;
  }

  /* Header */
  header { margin-bottom: 24px; }
  .header-top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 12px;
  }
  h1 { margin: 0; font-size: 28px; font-weight: 700; }
  .subtitle { margin: 4px 0 0; color: var(--text-muted); font-size: 14px; }
  .header-right { display: flex; align-items: center; gap: 12px; }
  .btn-icon {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 8px 12px;
    cursor: pointer;
    font-size: 16px;
  }
  .btn-link {
    color: #3b82f6;
    text-decoration: none;
    font-size: 14px;
    font-weight: 500;
  }
  .btn-link:hover { text-decoration: underline; }
  .meta-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    font-size: 13px;
    color: var(--text-muted);
  }

  /* KPI Cards */
  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 24px;
  }
  .kpi-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 20px;
    text-align: center;
  }
  .kpi-value { font-size: 32px; font-weight: 700; margin-bottom: 4px; }
  .kpi-label { font-size: 13px; color: var(--text-muted); font-weight: 500; text-transform: uppercase; letter-spacing: 0.5px; }
  .kpi-detail { font-size: 12px; color: var(--text-muted); margin-top: 4px; }
  .kpi-passed .kpi-value { color: #10b981; }
  .kpi-exec .kpi-value { color: #3b82f6; }
  .kpi-time .kpi-value { color: #f59e0b; }
  .kpi-exact .kpi-value { color: #8b5cf6; }

  /* Cards */
  .card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 16px;
  }
  .error-card { border-color: #fca5a5; background: #fef2f2; }
  .error-card code { background: #fee2e2; padding: 2px 6px; border-radius: 4px; font-size: 13px; }
  .muted { color: var(--text-muted); font-size: 13px; }
  h3 { margin: 0 0 16px; font-size: 16px; font-weight: 600; }

  /* Charts */
  .charts-row {
    display: grid;
    grid-template-columns: 1fr 1.5fr;
    gap: 16px;
    margin-bottom: 16px;
  }
  .chart-donut { position: relative; }
  .donut-container { position: relative; width: 100%; max-width: 220px; margin: 0 auto; }
  .donut-center {
    position: absolute;
    top: 50%; left: 50%;
    transform: translate(-50%, -50%);
    text-align: center;
  }
  .donut-pct { display: block; font-size: 28px; font-weight: 700; color: #10b981; }
  .donut-label { display: block; font-size: 12px; color: var(--text-muted); text-transform: uppercase; }
  .chart-container { height: 260px; }
  .chart-container-hbar { height: 260px; }
  .chart-card-full { margin-bottom: 16px; }
  .chart-container-wide { height: 200px; }

  /* Table */
  .table-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 12px;
    margin-bottom: 16px;
  }
  .filters { display: flex; gap: 8px; flex-wrap: wrap; }
  .filters select {
    padding: 6px 10px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-card);
    color: var(--text-primary);
    font-size: 13px;
    cursor: pointer;
  }
  .table-container { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; }
  th, td { padding: 10px 12px; text-align: left; border-bottom: 1px solid var(--border); font-size: 13px; }
  th {
    background: var(--hover-bg);
    font-weight: 600;
    color: var(--text-secondary);
    position: sticky; top: 0;
    user-select: none;
  }
  th.sortable { cursor: pointer; }
  th.sortable:hover { color: #3b82f6; }
  .clickable-row { cursor: pointer; }
  .clickable-row:hover td { background: var(--hover-bg); }
  .qid { font-weight: 600; color: #3b82f6; font-family: 'Courier New', monospace; }
  .question-cell { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .time-cell { font-family: 'Courier New', monospace; font-size: 12px; }

  /* Badges */
  .badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }
  .badge-exact { background: #d1fae5; color: #065f46; }
  .badge-good { background: #fef3c7; color: #92400e; }
  .badge-mismatch { background: #ffedd5; color: #9a3412; }
  .badge-error { background: #fee2e2; color: #991b1b; }
  .diff-easy { background: #dbeafe; color: #1e40af; }
  .diff-medium { background: #fef3c7; color: #92400e; }
  .diff-hard { background: #fce7f3; color: #9d174d; }

  /* Expanded row */
  .expanded-row td { padding: 16px; background: var(--hover-bg); }
  .sql-compare { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 12px; }
  .sql-block {
    background: #1e293b;
    border-radius: 8px;
    padding: 12px;
    overflow-x: auto;
  }
  .sql-block h4 { margin: 0 0 8px; font-size: 12px; color: #94a3b8; text-transform: uppercase; }
  .sql-block pre { margin: 0; font-size: 13px; color: #10b981; white-space: pre-wrap; word-break: break-word; font-family: 'Courier New', monospace; }
  .sql-block.sql-error pre { color: #f87171; }
  .match-details { display: flex; gap: 16px; font-size: 12px; color: var(--text-muted); }
  .error-text { color: #ef4444; }

  /* Responsive */
  @media (max-width: 768px) {
    .kpi-grid { grid-template-columns: repeat(2, 1fr); }
    .charts-row { grid-template-columns: 1fr; }
    .header-top { flex-direction: column; gap: 8px; }
    .sql-compare { grid-template-columns: 1fr; }
  }
</style>