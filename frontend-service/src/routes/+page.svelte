<svelte:head>
  <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet" />
</svelte:head>

<script>
  let question = $state("");
  let loading = $state(false);
  let errorMsg = $state("");
  
  let generatedSQL = $state("");
  let resultData = $state([]);

  async function askDatabase() {
    if (!question.trim()) return;
    
    loading = true;
    errorMsg = "";
    generatedSQL = "";
    resultData = [];

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
      // Menangani case jika data kosong / null dari Golang
      resultData = data.data || [];
    } catch (err) {
      errorMsg = err.message;
    } finally {
      loading = false;
    }
  }
</script>

<main class="container">
  <header>
    <h1>🧠 HRIS Data Assistant</h1>
    <p>Tanya AI menggunakan bahasa Inggris untuk melihat data HRIS (Departemen, Karyawan, Gaji, Absensi).</p>
  </header>

  <section class="search-box">
    <!-- Di Svelte 5, event menggunakan onkeydown (tanpa titik dua) -->
    <input 
      type="text" 
      bind:value={question} 
      onkeydown={(e) => e.key === 'Enter' && askDatabase()}
      placeholder="Contoh: Show me all employees in Sales department..."
      disabled={loading}
    />
    <!-- Event on:click diganti menjadi onclick -->
    <button onclick={askDatabase} disabled={loading}>
      {loading ? 'Berpikir...' : 'Tanya AI'}
    </button>
  </section>

  {#if errorMsg}
    <div class="error">❌ {errorMsg}</div>
  {/if}

  {#if generatedSQL}
    <section class="result-section">
      <div class="sql-box">
        <h4>Generated SQL:</h4>
        <code>{generatedSQL}</code>
      </div>

      <div class="table-container">
        {#if resultData && resultData.length > 0}
          <!-- Render kolom tabel secara dinamis berdasarkan key JSON -->
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
    max-width: 800px;
    margin: 40px auto;
    padding: 20px;
  }
  header {
    text-align: center;
    margin-bottom: 30px;
  }
  h1 {
    color: #0f172a;
  }
  .search-box {
    display: flex;
    gap: 10px;
    margin-bottom: 20px;
  }
  input {
    flex: 1;
    padding: 12px 16px;
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    font-size: 16px;
    outline: none;
    transition: border-color 0.2s;
  }
  input:focus {
    border-color: #3b82f6;
  }
  button {
    padding: 12px 24px;
    background-color: #3b82f6;
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    transition: background-color 0.2s;
    min-width: 120px;
  }
  button:hover:not(:disabled) {
    background-color: #2563eb;
  }
  button:disabled {
    background-color: #94a3b8;
    cursor: not-allowed;
  }
  .error {
    background-color: #fee2e2;
    color: #991b1b;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 20px;
  }
  .result-section {
    background-color: white;
    padding: 24px;
    border-radius: 12px;
    box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
  }
  .sql-box {
    background-color: #1e293b;
    color: #10b981;
    padding: 16px;
    border-radius: 8px;
    margin-bottom: 20px;
    overflow-x: auto;
  }
  .sql-box h4 {
    color: #94a3b8;
    margin: 0 0 8px 0;
    font-size: 14px;
  }
  code {
    font-family: 'Courier New', Courier, monospace;
    font-size: 15px;
  }
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
  }
  th {
    background-color: #f1f5f9;
    font-weight: 600;
    color: #475569;
  }
  .empty-state {
    text-align: center;
    color: #64748b;
    font-style: italic;
  }
</style>