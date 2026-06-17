<svelte:head>
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet" />
</svelte:head>

<script>
  // @ts-nocheck
  let question = $state("");
  let loading = $state(false);
  let errorMsg = $state("");
  
  let generatedSQL = $state("");
  let resultData = $state([]);
  let sqlExplanation = $state("");
  let explanationLoading = $state(false);

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
  loadHistory();
</script>

<main class="container">
  <header>
    <h1>🧠 HRIS Data Assistant</h1>
    <p>Tanya AI menggunakan bahasa Inggris untuk melihat data HRIS (Departemen, Karyawan, Gaji, Absensi).</p>
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
          <h4>Generated SQL:</h4>
          <button class="btn-ghost btn-sm" onclick={() => { navigator.clipboard.writeText(generatedSQL); }}>
            📋 Copy
          </button>
        </div>
        <code>{displayedSQL || generatedSQL}</code>
      </div>

      <!-- Explain SQL Button -->
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
          {@const columns = Object.keys(resultData[0])}
          <table>
            <thead>
              <tr>
                {#each columns as col}
                  <th>{col}</th>
                {/each}
              </tr>
            </thead>
            <tbody>
              {#each resultData as row}
                <tr>
                  {#each columns as col}
                    <td>{row[col]}</td>
                  {/each}
                </tr>
              {/each}
            </tbody>
          </table>
          <div class="row-count">Menampilkan {resultData.length} baris</div>
        {:else}
          <p class="empty-state">Query berhasil dieksekusi, tapi tidak ada data yang ditemukan.</p>
        {/if}
      </div>
    </section>
  {/if}
</main>

<style>
  /* Styling dasar */
  :global(body) {
    font-family: 'Plus Jakarta Sans', sans-serif;
    background-color: #f8fafc;
    color: #334155;
    margin: 0;
    padding: 0;
  }
  .container {
    max-width: 860px;
    margin: 40px auto;
    padding: 20px;
  }
  header {
    text-align: center;
    margin-bottom: 24px;
  }
  h1 {
    color: #0f172a;
  }
  header p {
    color: #64748b;
    font-size: 15px;
  }

  /* Suggested Questions */
  .suggestions {
    margin-bottom: 16px;
  }
  .suggestions-label {
    font-size: 13px;
    color: #64748b;
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
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 20px;
    font-size: 13px;
    color: #475569;
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
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    font-size: 16px;
    outline: none;
    transition: border-color 0.2s;
    font-family: inherit;
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
    color: #64748b;
    padding: 6px 12px;
    font-size: 13px;
    min-width: auto;
    font-weight: 500;
  }
  .btn-ghost:hover { color: #1e293b; }
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
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 10px;
    margin-bottom: 20px;
    max-height: 240px;
    overflow-y: auto;
  }
  .history-item {
    padding: 10px 16px;
    border-bottom: 1px solid #f1f5f9;
    cursor: pointer;
    transition: background 0.15s;
  }
  .history-item:last-child { border-bottom: none; }
  .history-item:hover { background: #f8fafc; }
  .history-question {
    font-size: 14px;
    color: #1e293b;
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
    background-color: white;
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
    background-color: #1e293b;
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
  .sql-box h4 {
    color: #94a3b8;
    margin: 0 0 8px 0;
    font-size: 14px;
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
    background: #f1f5f9;
    color: #475569;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 13px;
    min-width: auto;
    font-weight: 500;
  }
  .btn-secondary:hover:not(:disabled) {
    background: #e2e8f0;
    color: #1e293b;
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
    border-bottom: 1px solid #e2e8f0;
    font-size: 14px;
  }
  th {
    background-color: #f1f5f9;
    font-weight: 600;
    color: #475569;
    position: sticky;
    top: 0;
  }
  tr:hover td {
    background: #f8fafc;
  }
  .row-count {
    text-align: right;
    font-size: 12px;
    color: #94a3b8;
    margin-top: 8px;
  }
  .empty-state {
    text-align: center;
    color: #64748b;
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
</style>