<!-- @ts-nocheck -->
<script lang="ts">
  import { onDestroy } from 'svelte';
  import Chart from 'chart.js/auto';
  import { exportChartPNG } from './charts.js';

  let {
    type = 'bar',
    config = {},
    title = ''
  } = $props();

  let canvas: HTMLCanvasElement;
  let chartContainer: HTMLDivElement;

  let chartInstance: Chart | null = null;

  $effect(() => {
    if (canvas && config) {
      createChart();
    }
  });

  onDestroy(() => {
    if (chartInstance) {
      chartInstance.destroy();
    }
  });

  function createChart() {
    if (chartInstance) {
      chartInstance.destroy();
    }

    // Deep clone to detach from Svelte 5 $props proxy (avoids state_descriptors_fixed error)
    const safeConfig = JSON.parse(JSON.stringify(config));

    // Merge responsive defaults
    const mergedOptions = {
      ...safeConfig.options,
      responsive: true,
      maintainAspectRatio: false
    };

    chartInstance = new Chart(canvas, {
      type: safeConfig.type || type,
      data: safeConfig.data,
      options: mergedOptions
    });
  }

  function handleExportPNG() {
    if (canvas) {
      const filename = (title || 'chart').replace(/[^a-zA-Z0-9]/g, '_').toLowerCase() + '.png';
      exportChartPNG(canvas, filename);
    }
  }
</script>

<div class="chart-wrapper" bind:this={chartContainer}>
  <div class="chart-header">
    <h4 class="chart-title">{title}</h4>
    <div class="chart-actions">
      <button class="btn-export" on:click={handleExportPNG} title="Export as PNG">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="7 10 12 15 17 10"/>
          <line x1="12" y1="15" x2="12" y2="3"/>
        </svg>
        PNG
      </button>
    </div>
  </div>
  <div class="chart-canvas-container">
    <canvas bind:this={canvas}></canvas>
  </div>
</div>

<style>
  .chart-wrapper {
    background: var(--card-bg, #ffffff);
    border: 1px solid var(--border-color, #e2e8f0);
    border-radius: 10px;
    padding: 16px;
    margin-bottom: 12px;
  }
  :global([data-theme="dark"]) .chart-wrapper {
    background: var(--card-bg, #1e293b);
    border-color: var(--border-color, #334155);
  }
  .chart-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
    gap: 8px;
  }
  .chart-title {
    margin: 0;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-secondary, #475569);
    line-height: 1.3;
  }
  :global([data-theme="dark"]) .chart-title {
    color: #94a3b8;
  }
  .chart-actions {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }
  .btn-export {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    font-size: 11px;
    font-weight: 500;
    background: var(--hover-bg, #f1f5f9);
    color: var(--text-secondary, #64748b);
    border: 1px solid var(--border-color, #e2e8f0);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
    min-width: auto;
  }
  .btn-export:hover {
    background: var(--border-color, #e2e8f0);
    color: var(--text-primary, #1e293b);
  }
  :global([data-theme="dark"]) .btn-export {
    background: #334155;
    color: #94a3b8;
    border-color: #475569;
  }
  :global([data-theme="dark"]) .btn-export:hover {
    background: #475569;
    color: #e2e8f0;
  }
  .chart-canvas-container {
    position: relative;
    height: 280px;
  }
</style>