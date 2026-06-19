// @ts-nocheck
/**
 * Smart Chart Selection Utility
 * Analyzes SQL query results and automatically selects the best chart type
 * and prepares data for visualization.
 */

/**
 * Detect if a column contains date/time values
 */
function isDateColumn(values) {
  if (!values || values.length === 0) return false;
  const sample = values.slice(0, 10);
  let dateCount = 0;
  for (const v of sample) {
    if (v === null || v === '') continue;
    const str = String(v);
    if (
      /^\d{4}-\d{2}-\d{2}/.test(str) ||
      /^\d{2}\/\d{2}\/\d{4}/.test(str) ||
      /^\d{1,2}\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)/i.test(str) ||
      /^(January|February|March|April|May|June|July|August|September|October|November|December)/i.test(str)
    ) {
      dateCount++;
    }
  }
  return dateCount / sample.length > 0.5;
}

/**
 * Check if column name suggests it's a date
 */
function isDateColumnName(name) {
  const lower = name.toLowerCase();
  return /date|time|year|month|day|period|timestamp|created|updated|tanggal|bulan|tahun/i.test(lower);
}

/**
 * Check if column name suggests it's a label/category
 */
function isLabelColumn(name) {
  const lower = name.toLowerCase();
  return /name|label|category|type|status|department|division|region|district|city|province|nama|departemen|divisi|kategori|jenis|kecamatan|kelurahan/i.test(lower);
}

/**
 * Check if column name suggests it's a numeric metric
 */
function isNumericColumnName(name) {
  const lower = name.toLowerCase();
  return /count|total|sum|amount|avg|average|salary|gaji|revenue|cost|price|jumlah|nilai|frekuensi|frequency|incident|violation|population|luas|area/i.test(lower);
}

/**
 * Check if the query suggests aggregation (GROUP BY pattern)
 */
function detectAggregationPattern(headers, rows) {
  if (rows.length < 2) return false;
  // If we have exactly 2 columns, first is likely label, second is numeric
  if (headers.length === 2) {
    const firstType = typeof rows[0][headers[0]];
    const secondType = typeof rows[0][headers[1]];
    return true; // 2-column result is almost always aggregation
  }
  return false;
}

/**
 * Determine if a column is numeric
 */
function isNumericColumn(values) {
  if (!values || values.length === 0) return false;
  const sample = values.slice(0, 10);
  let numCount = 0;
  for (const v of sample) {
    if (v === null || v === '') continue;
    if (typeof v === 'number' || (!isNaN(Number(v)) && v !== '')) {
      numCount++;
    }
  }
  return numCount / sample.length > 0.6;
}

/**
 * Smart Chart Selection
 * Analyzes query results and returns chart configuration(s)
 * @param {string[]} headers - Column names
 * @param {Array<Object>} rows - Data rows
 * @param {string} question - The original user question
 * @returns {Array<{type: string, title: string, config: Object}>} Array of chart configs
 */
export function selectCharts(headers, rows, question = '') {
  if (!headers || !rows || headers.length < 2 || rows.length < 2) {
    return [];
  }

  const charts = [];
  const questionLower = question.toLowerCase();

  // Identify columns
  const numericCols = headers.filter((h) => isNumericColumn(rows.map((r) => r[h])));
  const dateCols = headers.filter(
    (h) => isDateColumn(rows.map((r) => r[h])) || isDateColumnName(h)
  );
  const labelCols = headers.filter((h) => isLabelColumn(h) || !numericCols.includes(h));
  const nonNumericCols = headers.filter((h) => !numericCols.includes(h));

  // Detect percentage columns
  const percentageCols = headers.filter((h) => {
    const lower = h.toLowerCase();
    if (/percent|%|ratio|rate|proporsi/i.test(lower)) return true;
    const values = rows.map((r) => r[h]);
    return values.every((v) => v !== null && Number(v) >= 0 && Number(v) <= 100);
  });

  // ─── Pattern 1: Aggregation (2 columns: label + numeric) ────────────────
  if (headers.length === 2 && numericCols.length >= 1) {
    const labelCol = nonNumericCols[0] || headers[0];
    const valueCol = numericCols[0] || headers[1];

    // Check for percentage → Pie Chart
    if (percentageCols.length > 0 || /percent|proporsi|ratio|persentase/i.test(questionLower)) {
      charts.push({
        type: 'pie',
        title: `Distribusi ${valueCol.replace(/_/g, ' ')} per ${labelCol.replace(/_/g, ' ')}`,
        config: buildPieConfig(rows, labelCol, valueCol)
      });
    }

    // Always add Bar Chart for aggregation
    charts.push({
      type: 'bar',
      title: `${valueCol.replace(/_/g, ' ')} per ${labelCol.replace(/_/g, ' ')}`,
      config: buildBarConfig(rows, labelCol, valueCol)
    });
  }

  // ─── Pattern 2: Date + Numeric = Line Chart ─────────────────────────────
  if (dateCols.length > 0 && numericCols.length > 0) {
    const dateCol = dateCols[0];
    const valueCol = numericCols[0];

    charts.push({
      type: 'line',
      title: `Tren ${valueCol.replace(/_/g, ' ')} berdasarkan ${dateCol.replace(/_/g, ' ')}`,
      config: buildLineConfig(rows, dateCol, valueCol)
    });
  }

  // ─── Pattern 3: Multiple numeric columns with label ─────────────────────
  if (headers.length >= 3 && numericCols.length >= 2 && nonNumericCols.length >= 1) {
    const labelCol = nonNumericCols[0];
    const valueCols = numericCols.slice(0, 4); // Max 4 series

    charts.push({
      type: 'bar',
      title: `Perbandingan ${valueCols.map((c) => c.replace(/_/g, ' ')).join(', ')}`,
      config: buildGroupedBarConfig(rows, labelCol, valueCols)
    });
  }

  // ─── Pattern 4: Percentage distribution ─────────────────────────────────
  if (percentageCols.length > 0 && nonNumericCols.length > 0 && charts.length === 0) {
    const labelCol = nonNumericCols[0];
    const valueCol = percentageCols[0];

    charts.push({
      type: 'pie',
      title: `Distribusi ${valueCol.replace(/_/g, ' ')}`,
      config: buildPieConfig(rows, labelCol, valueCol)
    });
  }

  // ─── Pattern 5: Single value results (KPI) ──────────────────────────────
  if (rows.length === 1 && headers.length <= 4) {
    // Don't chart single rows — better as KPI card (handled elsewhere)
    return [];
  }

  // If no charts matched but we have data, try a simple bar if there's a non-numeric + numeric pair
  if (charts.length === 0 && numericCols.length > 0 && nonNumericCols.length > 0) {
    const labelCol = nonNumericCols[0];
    const valueCol = numericCols[0];
    const limitedRows = rows.slice(0, 20); // Limit to 20 bars

    charts.push({
      type: 'bar',
      title: `${valueCol.replace(/_/g, ' ')} per ${labelCol.replace(/_/g, ' ')}`,
      config: buildBarConfig(limitedRows, labelCol, valueCol)
    });
  }

  // Deduplicate by type+title
  const seen = new Set();
  return charts.filter((c) => {
    const key = `${c.type}:${c.title}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

// ─── Chart Config Builders ─────────────────────────────────────────────────

const CHART_COLORS = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1',
  '#14b8a6', '#e11d48', '#a855f7', '#0ea5e9', '#eab308'
];

const CHART_COLORS_ALPHA = CHART_COLORS.map((c) => c + '80');

function buildBarConfig(rows, labelCol, valueCol) {
  const labels = rows.map((r) => {
    const val = String(r[labelCol] ?? '');
    return val.length > 20 ? val.substring(0, 20) + '…' : val;
  });
  const data = rows.map((r) => {
    const v = r[valueCol];
    return typeof v === 'number' ? v : parseFloat(v) || 0;
  });

  return {
    type: 'bar',
    data: {
      labels,
      datasets: [{
        label: valueCol.replace(/_/g, ' '),
        data,
        backgroundColor: CHART_COLORS.slice(0, data.length),
        borderColor: CHART_COLORS.slice(0, data.length),
        borderWidth: 1,
        borderRadius: 4
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: (ctx) => `${ctx.dataset.label}: ${formatNumber(ctx.raw)}`
          }
        }
      },
      scales: {
        y: {
          beginAtZero: true,
          ticks: { callback: (v) => formatNumber(v) }
        },
        x: {
          ticks: { maxRotation: 45, minRotation: 0 }
        }
      }
    }
  };
}

function buildGroupedBarConfig(rows, labelCol, valueCols) {
  const labels = rows.map((r) => {
    const val = String(r[labelCol] ?? '');
    return val.length > 20 ? val.substring(0, 20) + '…' : val;
  });
  const datasets = valueCols.map((col, i) => ({
    label: col.replace(/_/g, ' '),
    data: rows.map((r) => {
      const v = r[col];
      return typeof v === 'number' ? v : parseFloat(v) || 0;
    }),
    backgroundColor: CHART_COLORS_ALPHA[i],
    borderColor: CHART_COLORS[i],
    borderWidth: 1,
    borderRadius: 4
  }));

  return {
    type: 'bar',
    data: { labels, datasets },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { position: 'top' },
        tooltip: {
          callbacks: {
            label: (ctx) => `${ctx.dataset.label}: ${formatNumber(ctx.raw)}`
          }
        }
      },
      scales: {
        y: {
          beginAtZero: true,
          ticks: { callback: (v) => formatNumber(v) }
        }
      }
    }
  };
}

function buildLineConfig(rows, dateCol, valueCol) {
  const labels = rows.map((r) => String(r[dateCol] ?? ''));
  const data = rows.map((r) => {
    const v = r[valueCol];
    return typeof v === 'number' ? v : parseFloat(v) || 0;
  });

  return {
    type: 'line',
    data: {
      labels,
      datasets: [{
        label: valueCol.replace(/_/g, ' '),
        data,
        borderColor: CHART_COLORS[0],
        backgroundColor: CHART_COLORS[0] + '20',
        fill: true,
        tension: 0.3,
        pointRadius: 4,
        pointHoverRadius: 6
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: (ctx) => `${ctx.dataset.label}: ${formatNumber(ctx.raw)}`
          }
        }
      },
      scales: {
        y: {
          beginAtZero: true,
          ticks: { callback: (v) => formatNumber(v) }
        }
      }
    }
  };
}

function buildPieConfig(rows, labelCol, valueCol) {
  const labels = rows.map((r) => String(r[labelCol] ?? ''));
  const data = rows.map((r) => {
    const v = r[valueCol];
    return typeof v === 'number' ? v : parseFloat(v) || 0;
  });

  return {
    type: 'pie',
    data: {
      labels,
      datasets: [{
        data,
        backgroundColor: CHART_COLORS.slice(0, data.length),
        borderColor: '#ffffff',
        borderWidth: 2
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: {
          position: 'right',
          labels: { padding: 12, usePointStyle: true }
        },
        tooltip: {
          callbacks: {
            label: (ctx) => {
              const total = ctx.dataset.data.reduce((a, b) => a + b, 0);
              const pct = total > 0 ? ((ctx.raw / total) * 100).toFixed(1) : 0;
              return `${ctx.label}: ${formatNumber(ctx.raw)} (${pct}%)`;
            }
          }
        }
      }
    }
  };
}

function formatNumber(n) {
  if (typeof n !== 'number') return n;
  if (Math.abs(n) >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (Math.abs(n) >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return n % 1 === 0 ? n.toString() : n.toFixed(2);
}

/**
 * Export a chart canvas to PNG
 * @param {HTMLCanvasElement} canvas
 * @param {string} filename
 */
export function exportChartPNG(canvas, filename = 'chart.png') {
  const link = document.createElement('a');
  link.download = filename;
  link.href = canvas.toDataURL('image/png', 1.0);
  link.click();
}

/**
 * Export a chart canvas to SVG (via re-render)
 * @param {HTMLCanvasElement} canvas
 * @param {string} filename
 */
export function exportChartSVG(canvas, filename = 'chart.svg') {
  // Chart.js renders to canvas, so we export as high-res PNG instead
  // For true SVG, a library like Chart.js-plugin-svg would be needed
  const link = document.createElement('a');
  link.download = filename.replace('.svg', '.png');
  link.href = canvas.toDataURL('image/png', 1.0);
  link.click();
}