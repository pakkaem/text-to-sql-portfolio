<svelte:head>
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet" />
</svelte:head>

<script>
  // @ts-nocheck
  import { onMount } from "svelte";

  let question = $state("");
  let loading = $state(false);
  let errorMsg = $state("");
  
  let generatedSQL = $state("");
  let resultData = $state([]);
  let sqlExplanation = $state("");
  let explanationLoading = $state(false);
  let responseTime = $state("");

  // --- Dark Mode ---
  let darkMode = $state(false);

  function toggleDarkMode() {
    darkMode = !darkMode;
    document.documentElement.setAttribute("data-theme", darkMode ? "dark" : "light");
    localStorage.setItem("hris-dark-mode", darkMode ? "1" : "0");
  }

  // --- Health Status ---
  let healthStatus = $state({ db: false, ai: false, checked: false });

  async function checkHealth() {
    try {
      const resp = await fetch("http://localhost:8080/health");
      const data = await resp.json();
      healthStatus = { db: data.db === "connected", ai: data.ai_service === true, checked: true };
    } catch {
      healthStatus = { db: false, ai: false, checked: true };
    }
  }

  // --- Schema Explorer ---
  let schemaData = $state([]);
  let showSchema = $state(false);
  let expandedTable = $state(null);

  async function loadSchema() {
    try {
      const resp = await fetch("http://localhost:8080/schema");
      const data = await resp.json();
      schemaData = data.tables || [];
    } catch { schemaData = []; }
  }

  function toggleTable(tableName) {
    expandedTable = expandedTable === tableName ? null : tableName;
  }

  // --- Suggested Questions ---
  const suggestedQuestions = [
    "How many employees are in each department?",
    "Who earns the highest salary?",
    "List all ongoing projects",
    "Show attendance records for this month",
    "Which employees are in multiple projects?",
    "What is the total payroll by department?"
  ];

  // --- Query History (localStorage) ---
  let history = $state([]);
  let showHistory = $state(false);

  function loadHistory() {
    try {
      const saved = localStorage.getItem("hris-query-history");
      if (saved) history = JSON.parse(saved);
    } catch { history = []; }
  }

  function saveToHistory(q, sql, count) {
    const entry = { question: q, sql, rowCount: count, time: new Date().toLocaleString("id-ID") };
    history = [entry, ...history.filter(h => h.question !== q)].slice(0, 20);
    localStorage.setItem("hris-query-history", JSON.stringify(history));
  }

  function loadFromHistory(entry) {
    question = entry.question;
    showHistory = false;
    askDatabase();
  }

  function clearHistory() {
    history = [];
    localStorage.removeItem("hris-query-history");
  }

  // --- Export to CSV ---
  function exportCSV() {
    if (!resultData.length) return;
    const columns = Object.keys(resultData[0]);
    const csvRows = [
      columns.join(","),
      ...resultData.map(row => columns.map(col => {
        const val = String(row[col] ?? "");
        return val.includes(",") || val.includes('"') || val.includes("\n")
          ? `"${val.replace(/"/g, '""')}"` : val;
      }).join(","))
    ];
    const blob = new Blob([csvRows.join("\n")], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `hris-query-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  // --- Pagination ---
  let currentPage = $state(0);
  const PAGE_SIZE = 20;
  let totalPages = $derived(resultData.length > 0 ? Math.ceil(resultData.length / PAGE_SIZE) : 0);
  let pagedData = $derived(resultData.slice(currentPage * PAGE_SIZE, (currentPage + 1) * PAGE_SIZE));
  let displayColumns = $derived(resultData.length > 0 ? Object.keys(resultData[0]) : []);

  function goToPage(p) {
    if (p >= 0 && p < totalPages) currentPage = p;
  }

  // --- Typing Animation State ---
  let displayedSQL = $state("");
  let typingInterval = null;

  function typeSQL(sql) {
    displayedSQL = "";
    if (typingInterval) clearInterval(typingInterval);
    let i = 0;
    typingInterval = setInterval(() => {
      if (i < sql.length) {
        displayedSQL += sql[i];
        i++;
      } else {
        clearInterval(typingInterval);
        typingInterval = null;
      }
    }, 15);
  }

  // --- Main Ask Function ---
  async function askDatabase() {
    if (!question.trim()) return;
    
    loading = true;
    errorMsg = "";
    generatedSQL = "";
    displayedSQL = "";
    resultData = [];
    sqlExplanation = "";
    responseTime = "";
    currentPage = 0;

    try {
      const response = await fetch("http://localhost:8080/ask", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question })
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || "Terjadi kesalahan pada server");
      }

      generatedSQL = data.generated_sql;
      resultData = data.data || [];
      responseTime = data.response_time || "";

      // Start typing animation
      typeSQL(generatedSQL);

      // Save to history
      saveToHistory(question, generatedSQL, resultData.length);

    } catch (err) {
      errorMsg = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  // --- SQL Explanation ---
  async function explainSQL() {
    if (!generatedSQL) return;
    explanationLoading = true;
    sqlExplanation = "";

    try {
      const response = await fetch("http://localhost:8080/explain", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sql: generatedSQL, question: question })
      });

      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gagal menjelaskan SQL");
      sqlExplanation = data.explanation;
    } catch (err) {
      sqlExplanation = "Gagal menjelaskan SQL: " + (err instanceof Error ? err.message : String(err));
    } finally {
      explanationLoading = false;
    }
  }

  // Init
  onMount(() => {
    const savedTheme = localStorage.getItem("hris-dark-mode");
    if (savedTheme === "1") {
      darkMode = true;
      document.documentElement.setAttribute("data-theme", "dark");
    }
    loadHistory();
    checkHealth();
    loadSchema();
  });
</script>

<div class="app-layout">
  <!-- Schema Sidebar -->
  <aside class="sidebar" class:sidebar-open={showSchema}>
    <div class="sidebar-header">
      <h3>🗄️ Database Schema</h3>
      <button class="btn-close" onclick={() => showSchema = false}>✕</button>
    </div>
    <div class="sidebar-content">
      {#if schemaData.length === 0}
        <p class="sidebar-empty">Loading schema...</p>
      {/if}
      {#each schemaData as table}
        <div class="schema-table">
          <button class="table-header" onclick={() => toggleTable(table.name)}>
            <span class="table-icon">{expandedTable === table.name ? '📂' : '📁'}</span>
            <span class="table-name">{table.name}</span>
            <span class="col-count">{table.columns?.length || 0} cols</span>
          </button>
          {#if expandedTable === table.name}
            <div class="table-columns">
              {#each table.columns || [] as col}
                <div class="column-item">
                  <span class="col-icon">{col.pk ? '🔑' : '📄'}</span>
                  <span class="col-name">{col.name}</span>
                  <span class="col-type">{col.type}</span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  </aside>

  <!-- Sidebar Overlay (mobile) -->
  {#if showSchema}
    <div class="sidebar-overlay" onclick={() => showSchema = false}></div>
  {/if}

  <!-- Main Content -->
  <main class="container">
    <header>
      <div class="header-top">
        <div class="header-left">
          <button class="btn-icon" onclick={() => showSchema = !showSchema} title="Toggle Schema Explorer">
            🗄️
          </button>
        </div>
        <div class="header-center">
          <h1>🧠 HRIS Data Assistant</h1>
          <p>Tanya AI menggunakan bahasa Inggris untuk melihat data HRIS (Departemen, Karyawan, Gaji, Absensi).</p>
        </div>
        <div class="header-right">
          <!-- Health Status -->
          <div class="health-dots" title={healthStatus.db && healthStatus.ai ? "Semua service aktif" : "Ada service yang tidak aktif"}>
            <span class="dot" class:dot-green={healthStatus.db} class:dot-red={!healthStatus.db} title="Backend DB"></span>
            <span class="dot" class:dot-green={healthStatus.ai} class:dot-red={!healthStatus.ai} title="AI Service"></span>
          </div>
          <!-- Dark Mode Toggle -->
          <button class="btn-icon" onclick={toggleDarkMode} title="Toggle Dark Mode">
            {darkMode ? '☀️' : '🌙'}
          </button>
        </div>
      </div>
    </header>

    <!-- Suggested Questions -->
    <section class="suggestions">
      <span class="suggestions-label">💡 Coba tanyakan:</span>
      <div class="suggestion-chips">
        {#each suggestedQuestions as sq}
          <button class="chip" onclick={() => { question = sq; askDatabase(); }} disabled={loading}>
            {sq}
          </button>
        {/each}
      </div>
    </section>

    <section class="search-box">
      <input 
        type="text" 
        bind:value={question} 
        onkeydown={(e) => e.key === 'Enter' && askDatabase()}
        placeholder="Contoh: Show me all employees in Sales department..."
        disabled={loading}
      />
      <button onclick={askDatabase} disabled={loading}>
        {loading ? '🤔' : '🔍'} {loading ? 'Berpikir...' : 'Tanya AI'}
      </button>
    </section>

    <!-- History Toggle -->
    <section class="history-toggle">
      <button class="btn-ghost" onclick={() => showHistory = !showHistory}>
        📜 {showHistory ? 'Sembunyikan' : 'Riwayat'} ({history.length})
      </button>
      {#if history.length > 0}
        <button class="btn-ghost btn-danger" onclick={clearHistory}>🗑️ Hapus</button>
      {/if}
    </section>

    <!-- History Panel -->
    {#if showHistory && history.length > 0}
      <section class="history-panel">
        {#each history as entry}
          <div class="history-item" onclick={() => loadFromHistory(entry)} role="button" tabindex="0"
               onkeydown={(e) => e.key === 'Enter' && loadFromHistory(entry)}>
            <div class="history-question">{entry.question}</div>
            <div class="history-meta">
              <span>{entry.rowCount} rows</span>
              <span>{entry.time}</span>
            </div>
          </div>
        {/each}
      </section>
    {/if}

    {#if errorMsg}
      <div class="error">❌ {errorMsg}</div>
    {/if}

    {#if generatedSQL}
      <section class="result-section">
        <div class="sql-box">
          <div class="sql-header">
            <div class="sql-header-left">
              <h4>Generated SQL:</h4>
              {#if responseTime}
                <span class="response-time">⚡ {responseTime}</span>
              {/if}
            </div>
            <button class="btn-ghost btn-sm" onclick={() => { navigator.clipboard.writeText(generatedSQL); }}>
              📋 Copy
            </button>
          </div>
          <code>{displayedSQL || generatedSQL}</code>
        </div>

        <!-- Action Bar -->
        <div class="action-bar">
          <button class="btn-secondary" onclick={explainSQL} disabled={explanationLoading}>
            📖 {explanationLoading ? 'Menjelaskan...' : 'Jelaskan SQL'}
          </button>
          <button class="btn-secondary" onclick={exportCSV}>
            📥 Export CSV ({resultData.length} rows)
          </button>
        </div>

        <!-- SQL Explanation -->
        {#if sqlExplanation}
          <div class="explanation-box">
            <h4>💡 Penjelasan:</h4>
            <p>{sqlExplanation}</p>
          </div>
        {/if}

        <div class="table-container">
          {#if loading}
            <!-- Skeleton Loading -->
            <div class="skeleton-table">
              {#each [1,2,3,4,5] as _}
                <div class="skeleton-row">
                  <div class="skeleton-cell"></div>
                  <div class="skeleton-cell"></div>
                  <div class="skeleton-cell"></div>
                </div>
              {/each}
            </div>
          {:else if resultData && resultData.length > 0}
            <table>
              <thead>
                <tr>
                  {#each displayColumns as col}
                    <th>{col}</th>
                  {/each}
                </tr>
              </thead>
              <tbody>
                {#each pagedData as row}
                  <tr>
                    {#each displayColumns as col}
                      <td>{row[col]}</td>
                    {/each}
                  </tr>
                {/each}
              </tbody>
            </table>

            <!-- Pagination -->
            {#if totalPages > 1}
              <div class="pagination">
                <button class="btn-page" onclick={() => goToPage(currentPage - 1)} disabled={currentPage === 0}>
                  ← Prev
                </button>
                <div class="page-info">
                  {#each Array(totalPages) as _, i}
                    <button class="btn-page-num" class:active-page={i === currentPage} onclick={() => goToPage(i)}>
                      {i + 1}
                    </button>
                  {/each}
                </div>
                <button class="btn-page" onclick={() => goToPage(currentPage + 1)} disabled={currentPage >= totalPages - 1}>
                  Next →
                </button>
              </div>
            {/if}

            <div class="row-count">
              Menampilkan {currentPage * PAGE_SIZE + 1}–{Math.min((currentPage + 1) * PAGE_SIZE, resultData.length)} dari {resultData.length} baris
              {#if responseTime} · ⚡ {responseTime}{/if}
            </div>
          {:else}
            <p class="empty-state">Query berhasil dieksekusi, tapi tidak ada data yang ditemukan.</p>
          {/if}
        </div>
      </section>
    {/if}
  </main>
</div>

<style>
  /* ===== Layout ===== */
  .app-layout {
    display: flex;
    min-height: 100vh;
  }

  /* ===== Sidebar ===== */
  .sidebar {
    width: 0;
    overflow: hidden;
    background: var(--sidebar-bg, white);
    border-right: 1px solid var(--border-color, #e2e8f0);
    transition: width 0.3s ease;
    flex-shrink: 0;
    position: sticky;
    top: 0;
    height: 100vh;
  }
  .sidebar-open {
    width: 300px;
  }
  .sidebar-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-color, #e2e8f0);
  }
  .sidebar-header h3 {
    margin: 0;
    font-size: 15px;
    color: var(--text-primary, #0f172a);
  }
  .btn-close {
    background: none;
    border: none;
    font-size: 18px;
    cursor: pointer;
    color: var(--text-muted, #64748b);
    padding: 4px 8px;
    min-width: auto;
  }
  .sidebar-content {
    padding: 12px;
    overflow-y: auto;
    height: calc(100vh - 60px);
  }
  .sidebar-empty {
    text-align: center;
    color: var(--text-muted, #94a3b8);
    font-size: 13px;
    padding: 20px;
  }
  .schema-table {
    margin-bottom: 4px;
  }
  .table-header {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: none;
    border: none;
    cursor: pointer;
    border-radius: 6px;
    transition: background 0.15s;
    font-size: 13px;
    color: var(--text-primary, #1e293b);
    min-width: auto;
    text-align: left;
    font-family: inherit;
  }
  .table-header:hover {
    background: var(--hover-bg, #f1f5f9);
  }
  .table-icon { font-size: 14px; }
  .table-name { font-weight: 600; flex: 1; }
  .col-count { font-size: 11px; color: var(--text-muted, #94a3b8); }
  .table-columns {
    padding: 4px 0 4px 28px;
    animation: fadeIn 0.2s ease;
  }
  .column-item {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    font-size: 12px;
    color: var(--text-secondary, #475569);
  }
  .col-icon { font-size: 11px; }
  .col-name { flex: 1; font-family: 'Courier New', monospace; }
  .col-type { font-size: 11px; color: var(--text-muted, #94a3b8); background: var(--tag-bg, #f1f5f9); padding: 1px 6px; border-radius: 3px; }

  .sidebar-overlay {
    display: none;
  }

  /* ===== Global Theming ===== */
  :global(body) {
    font-family: 'Plus Jakarta Sans', sans-serif;
    background-color: var(--bg-main, #f8fafc);
    color: var(--text-primary, #334155);
    margin: 0;
    padding: 0;
  }
  :global(:root) {
    --bg-main: #f8fafc;
    --bg-card: white;
    --bg-code: #1e293b;
    --text-primary: #0f172a;
    --text-secondary: #475569;
    --text-muted: #64748b;
    --border-color: #e2e8f0;
    --hover-bg: #f1f5f9;
    --input-bg: white;
    --input-border: #cbd5e1;
    --sidebar-bg: white;
    --tag-bg: #f1f5f9;
    --chip-bg: white;
    --chip-border: #e2e8f0;
    --chip-text: #475569;
  }
  :global([data-theme="dark"]) {
    --bg-main: #0f172a;
    --bg-card: #1e293b;
    --bg-code: #0d1117;
    --text-primary: #e2e8f0;
    --text-secondary: #94a3b8;
    --text-muted: #64748b;
    --border-color: #334155;
    --hover-bg: #334155;
    --input-bg: #1e293b;
    --input-border: #475569;
    --sidebar-bg: #1e293b;
    --tag-bg: #334155;
    --chip-bg: #1e293b;
    --chip-border: #334155;
    --chip-text: #94a3b8;
  }

  .container {
    flex: 1;
    max-width: 860px;
    margin: 0 auto;
    padding: 40px 20px;
  }

  /* Header */
  header { margin-bottom: 24px; }
  .header-top {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
  }
  .header-left, .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
    padding-top: 8px;
  }
  .header-center {
    flex: 1;
    text-align: center;
  }
  h1 {
    color: var(--text-primary, #0f172a);
    margin: 0;
  }
  header p {
    color: var(--text-muted, #64748b);
    font-size: 15px;
    margin: 4px 0 0 0;
  }

  /* Health Dots */
  .health-dots {
    display: flex;
    gap: 6px;
    align-items: center;
    cursor: default;
  }
  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    display: inline-block;
  }
  .dot-green { background: #22c55e; box-shadow: 0 0 6px rgba(34,197,94,0.5); }
  .dot-red { background: #ef4444; box-shadow: 0 0 6px rgba(239,68,68,0.5); }

  .btn-icon {
    background: var(--bg-card, white);
    border: 1px solid var(--border-color, #e2e8f0);
    border-radius: 8px;
    padding: 8px 10px;
    font-size: 18px;
    cursor: pointer;
    transition: background 0.15s;
    min-width: auto;
  }
  .btn-icon:hover {
    background: var(--hover-bg, #f1f5f9);
  }

  /* Suggested Questions */
  .suggestions { margin-bottom: 16px; }
  .suggestions-label {
    font-size: 13px;
    color: var(--text-muted, #64748b);
    display: block;
    margin-bottom: 8px;
  }
  .suggestion-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .chip {
    padding: 6px 14px;
    background: var(--chip-bg, white);
    border: 1px solid var(--chip-border, #e2e8f0);
    border-radius: 20px;
    font-size: 13px;
    color: var(--chip-text, #475569);
    cursor: pointer;
    transition: all 0.15s;
    font-family: inherit;
  }
  .chip:hover:not(:disabled) {
    background: #eff6ff;
    border-color: #93c5fd;
    color: #1d4ed8;
  }
  .chip:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Search */
  .search-box {
    display: flex;
    gap: 10px;
    margin-bottom: 12px;
  }
  input {
    flex: 1;
    padding: 12px 16px;
    border: 1px solid var(--input-border, #cbd5e1);
    border-radius: 8px;
    font-size: 16px;
    outline: none;
    transition: border-color 0.2s;
    font-family: inherit;
    background: var(--input-bg, white);
    color: var(--text-primary, #0f172a);
  }
  input:focus {
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59,130,246,0.1);
  }
  button {
    padding: 12px 24px;
    background-color: #3b82f6;
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
    transition: background-color 0.2s;
    min-width: 130px;
    font-family: inherit;
  }
  button:hover:not(:disabled) {
    background-color: #2563eb;
  }
  button:disabled {
    background-color: #94a3b8;
    cursor: not-allowed;
  }

  /* Ghost Buttons */
  .btn-ghost {
    background: transparent;
    border: none;
    color: var(--text-muted, #64748b);
    padding: 6px 12px;
    font-size: 13px;
    min-width: auto;
    font-weight: 500;
  }
  .btn-ghost:hover { color: var(--text-primary, #1e293b); }
  .btn-danger { color: #dc2626; }
  .btn-danger:hover { color: #991b1b; }

  /* History Toggle */
  .history-toggle {
    display: flex;
    gap: 8px;
    margin-bottom: 8px;
  }

  /* History Panel */
  .history-panel {
    background: var(--bg-card, white);
    border: 1px solid var(--border-color, #e2e8f0);
    border-radius: 10px;
    margin-bottom: 20px;
    max-height: 240px;
    overflow-y: auto;
  }
  .history-item {
    padding: 10px 16px;
    border-bottom: 1px solid var(--border-color, #f1f5f9);
    cursor: pointer;
    transition: background 0.15s;
  }
  .history-item:last-child { border-bottom: none; }
  .history-item:hover { background: var(--hover-bg, #f8fafc); }
  .history-question {
    font-size: 14px;
    color: var(--text-primary, #1e293b);
    font-weight: 500;
  }
  .history-meta {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: #94a3b8;
    margin-top: 2px;
  }

  /* Error */
  .error {
    background-color: #fee2e2;
    color: #991b1b;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 20px;
  }

  /* Result Section */
  .result-section {
    background-color: var(--bg-card, white);
    padding: 24px;
    border-radius: 12px;
    box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
    animation: fadeIn 0.3s ease;
  }
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  /* SQL Box */
  .sql-box {
    background-color: var(--bg-code, #1e293b);
    color: #10b981;
    padding: 16px;
    border-radius: 8px;
    margin-bottom: 16px;
    overflow-x: auto;
  }
  .sql-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .sql-header-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .sql-box h4 {
    color: #94a3b8;
    margin: 0 0 8px 0;
    font-size: 14px;
  }
  .response-time {
    font-size: 12px;
    color: #fbbf24;
    background: rgba(251,191,36,0.1);
    padding: 2px 8px;
    border-radius: 4px;
    font-weight: 600;
    margin-bottom: 8px;
  }
  .btn-sm {
    padding: 4px 10px;
    font-size: 12px;
    min-width: auto;
    background: #334155;
    color: #94a3b8;
    border-radius: 4px;
  }
  .btn-sm:hover { background: #475569; color: white; }
  code {
    font-family: 'Courier New', Courier, monospace;
    font-size: 15px;
  }

  /* Action Bar */
  .action-bar {
    display: flex;
    gap: 10px;
    margin-bottom: 16px;
  }
  .btn-secondary {
    padding: 8px 16px;
    background: var(--hover-bg, #f1f5f9);
    color: var(--text-secondary, #475569);
    border: 1px solid var(--border-color, #e2e8f0);
    border-radius: 6px;
    font-size: 13px;
    min-width: auto;
    font-weight: 500;
  }
  .btn-secondary:hover:not(:disabled) {
    background: var(--border-color, #e2e8f0);
    color: var(--text-primary, #1e293b);
  }

  /* Explanation Box */
  .explanation-box {
    background: #eff6ff;
    border: 1px solid #bfdbfe;
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 16px;
  }
  .explanation-box h4 {
    margin: 0 0 8px 0;
    font-size: 14px;
    color: #1e40af;
  }
  .explanation-box p {
    margin: 0;
    font-size: 14px;
    color: #1e3a5f;
    line-height: 1.6;
  }

  /* Table */
  .table-container {
    overflow-x: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
  }
  th, td {
    padding: 12px;
    text-align: left;
    border-bottom: 1px solid var(--border-color, #e2e8f0);
    font-size: 14px;
  }
  th {
    background-color: var(--hover-bg, #f1f5f9);
    font-weight: 600;
    color: var(--text-secondary, #475569);
    position: sticky;
    top: 0;
  }
  tr:hover td {
    background: var(--hover-bg, #f8fafc);
  }

  /* Pagination */
  .pagination {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--border-color, #e2e8f0);
  }
  .page-info {
    display: flex;
    gap: 4px;
  }
  .btn-page {
    padding: 6px 14px;
    font-size: 13px;
    min-width: auto;
    background: var(--hover-bg, #f1f5f9);
    color: var(--text-secondary, #475569);
    border: 1px solid var(--border-color, #e2e8f0);
    border-radius: 6px;
  }
  .btn-page:hover:not(:disabled) {
    background: var(--border-color, #e2e8f0);
  }
  .btn-page-num {
    padding: 6px 12px;
    font-size: 13px;
    min-width: auto;
    background: transparent;
    color: var(--text-secondary, #475569);
    border: 1px solid transparent;
    border-radius: 6px;
  }
  .btn-page-num:hover {
    background: var(--hover-bg, #f1f5f9);
  }
  .active-page {
    background: #3b82f6 !important;
    color: white !important;
    font-weight: 600;
  }

  .row-count {
    text-align: right;
    font-size: 12px;
    color: #94a3b8;
    margin-top: 8px;
  }
  .empty-state {
    text-align: center;
    color: var(--text-muted, #64748b);
    font-style: italic;
    padding: 24px;
  }

  /* Skeleton Loading */
  .skeleton-table {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .skeleton-row {
    display: flex;
    gap: 12px;
  }
  .skeleton-cell {
    height: 36px;
    flex: 1;
    background: linear-gradient(90deg, #e2e8f0 25%, #f1f5f9 50%, #e2e8f0 75%);
    background-size: 200% 100%;
    border-radius: 4px;
    animation: shimmer 1.5s infinite;
  }
  @keyframes shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
  }

  /* Responsive */
  @media (max-width: 768px) {
    .sidebar-open {
      position: fixed;
      z-index: 100;
      width: 280px;
      box-shadow: 4px 0 20px rgba(0,0,0,0.3);
    }
    .sidebar-overlay {
      display: block;
      position: fixed;
      inset: 0;
      background: rgba(0,0,0,0.4);
      z-index: 99;
    }
    .header-top {
      flex-wrap: wrap;
      justify-content: center;
    }
    .header-left, .header-right {
      padding-top: 0;
    }
    .search-box {
      flex-direction: column;
    }
    button {
      min-width: 100%;
    }
    .action-bar {
      flex-direction: column;
    }
    .btn-secondary {
      width: 100%;
      text-align: center;
    }
  }
</style>