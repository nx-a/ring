<script>
  import { onMount } from 'svelte';
  import { api } from '../api.js';

  let buckets = $state([]);
  let selectedBucket = $state('');
  let ext = $state('');
  let level = $state('');
  let dataQuery = $state('');
  let timeStart = $state('');
  let timeEnd = $state('');
  let limit = $state(50);
  let offset = $state(0);
  let logs = $state([]);
  let error = $state('');
  let loading = $state(false);
  let total = $state(0);

  onMount(async () => {
    try {
      buckets = await api.getBuckets();
    } catch (e) {
      error = e.message;
    }
  });

  async function search() {
    if (!selectedBucket) {
      error = 'Select a bucket';
      return;
    }
    loading = true;
    error = '';
    try {
      const res = await api.getLogs({
        bucket: selectedBucket,
        ext: ext ? ext.split(',').map(s => s.trim()).filter(Boolean) : undefined,
        level: level ? level.split(',').map(s => s.trim().toUpperCase()).filter(Boolean) : undefined,
        data: dataQuery ? dataQuery.split(',').map(s => s.trim()).filter(Boolean) : undefined,
        timeStart: timeStart ? new Date(timeStart).toISOString() : undefined,
        timeEnd: timeEnd ? new Date(timeEnd).toISOString() : undefined,
        limit,
        offset,
      });
      logs = res.map(item => {
        let parsed = {};
        try {
          parsed = item.val ? JSON.parse(item.val) : {};
        } catch (e) {
          parsed = { raw: item.val };
        }
        return { ...item, parsed };
      });
      total = logs.length;
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function prevPage() {
    offset = Math.max(0, offset - limit);
    search();
  }

  function nextPage() {
    offset += limit;
    search();
  }

  function getParams() {
    return {
      bucket: selectedBucket,
      ext: ext ? ext.split(',').map(s => s.trim()).filter(Boolean) : undefined,
      level: level ? level.split(',').map(s => s.trim().toUpperCase()).filter(Boolean) : undefined,
      data: dataQuery ? dataQuery.split(',').map(s => s.trim()).filter(Boolean) : undefined,
      timeStart: timeStart ? new Date(timeStart).toISOString() : undefined,
      timeEnd: timeEnd ? new Date(timeEnd).toISOString() : undefined,
    };
  }

  function exportLogs(format) {
    window.open(api.exportUrl(getParams(), format), '_blank');
  }

  function formatTime(t) {
    if (!t) return '-';
    return new Date(t).toLocaleString();
  }
</script>

<h1>Logs</h1>

{#if error}
  <div class="alert error">{error}</div>
{/if}

<div class="filters card">
  <div class="row">
    <label>
      Bucket
      <select bind:value={selectedBucket}>
        <option value="">Select bucket</option>
        {#each buckets as b}
          <option value={b.systemName}>{b.systemName}</option>
        {/each}
      </select>
    </label>
    <label>
      Level
      <input type="text" bind:value={level} placeholder="INFO,ERROR" />
    </label>
    <label>
      External ID
      <input type="text" bind:value={ext} placeholder="component.name" />
    </label>
  </div>
  <div class="row">
    <label>
      From
      <input type="datetime-local" bind:value={timeStart} />
    </label>
    <label>
      To
      <input type="datetime-local" bind:value={timeEnd} />
    </label>
    <label>
      Data keys
      <input type="text" bind:value={dataQuery} placeholder="userId,traceId" />
    </label>
  </div>
  <div class="row">
    <label>
      Limit
      <input type="number" bind:value={limit} min="1" max="1000" />
    </label>
    <button onclick={search} disabled={loading}>
      {loading ? 'Loading...' : 'Search'}
    </button>
  </div>
  <div class="row export">
    <span>Export:</span>
    <button class="secondary" onclick={() => exportLogs('json')}>JSON</button>
    <button class="secondary" onclick={() => exportLogs('csv')}>CSV</button>
    <button class="secondary" onclick={() => exportLogs('txt')}>TXT</button>
  </div>
</div>

<div class="results">
  {#if logs.length === 0 && !loading}
    <div class="empty">No logs found</div>
  {:else}
    <table>
      <thead>
        <tr>
          <th>Time</th>
          <th>Level</th>
          <th>Point</th>
          <th>Structured data</th>
        </tr>
      </thead>
      <tbody>
        {#each logs as log}
          <tr>
            <td class="time">{formatTime(log.time)}</td>
            <td>
              <span class="badge level-{log.level}">{log.level}</span>
            </td>
            <td>{log.pointId}</td>
            <td class="json">
              <pre>{JSON.stringify(log.parsed, null, 2)}</pre>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

{#if logs.length > 0}
  <div class="pagination">
    <button onclick={prevPage} disabled={offset === 0}>Previous</button>
    <span>Offset {offset}, limit {limit}, loaded {total}</span>
    <button onclick={nextPage}>Next</button>
  </div>
{/if}

<style>
  h1 {
    margin-bottom: 1rem;
  }
  .card {
    background: white;
    padding: 1.5rem;
    border-radius: 12px;
    box-shadow: 0 2px 12px rgba(0,0,0,0.06);
    margin-bottom: 1.5rem;
  }
  .row {
    display: flex;
    gap: 1rem;
    flex-wrap: wrap;
    align-items: flex-end;
    margin-bottom: 1rem;
  }
  .row:last-child {
    margin-bottom: 0;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 0.85rem;
    color: #475569;
    flex: 1;
    min-width: 180px;
  }
  input, select {
    padding: 0.5rem 0.75rem;
    border: 1px solid #cbd5e1;
    border-radius: 6px;
    font-size: 0.95rem;
  }
  button {
    background: #2563eb;
    color: white;
    border: none;
    padding: 0.6rem 1.2rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.95rem;
  }
  button.secondary {
    background: #e2e8f0;
    color: #1e293b;
    padding: 0.5rem 0.75rem;
  }
  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .export {
    align-items: center;
  }
  .export span {
    color: #475569;
    font-size: 0.9rem;
  }
  .results {
    overflow-x: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    background: white;
    border-radius: 12px;
    overflow: hidden;
    box-shadow: 0 2px 12px rgba(0,0,0,0.06);
  }
  th, td {
    padding: 0.75rem 1rem;
    text-align: left;
    border-bottom: 1px solid #e2e8f0;
  }
  th {
    background: #f8fafc;
    font-weight: 600;
  }
  .time {
    white-space: nowrap;
    font-family: monospace;
  }
  .json pre {
    margin: 0;
    font-size: 0.8rem;
    max-width: 400px;
    overflow: auto;
    background: #f8fafc;
    padding: 0.5rem;
    border-radius: 6px;
  }
  .badge {
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
  }
  .level-ERROR { background: #fee2e2; color: #991b1b; }
  .level-WARN { background: #fef3c7; color: #92400e; }
  .level-INFO { background: #dbeafe; color: #1e40af; }
  .level-DEBUG { background: #e2e8f0; color: #475569; }
  .level-TRACE { background: #f1f5f9; color: #64748b; }
  .empty {
    padding: 3rem;
    text-align: center;
    color: #64748b;
  }
  .pagination {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 1.5rem;
  }
  .alert {
    padding: 0.75rem 1rem;
    border-radius: 6px;
    margin-bottom: 1rem;
  }
  .error {
    background: #fee2e2;
    color: #991b1b;
  }
</style>
