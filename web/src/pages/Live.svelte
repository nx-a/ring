<script>
  import { onMount, onDestroy } from 'svelte';
  import { createEventSource } from '../api.js';

  let logs = $state([]);
  let error = $state('');
  let connected = $state(false);
  let es = $state(null);

  onMount(() => {
    es = createEventSource();
    es.onopen = () => {
      connected = true;
      error = '';
    };
    es.onmessage = (e) => {
      if (e.lastEventId === 'connected') return;
      try {
        const log = JSON.parse(e.data);
        logs.unshift(log);
        if (logs.length > 500) logs.pop();
        logs = logs;
      } catch (err) {
        error = err.message;
      }
    };
    es.onerror = () => {
      connected = false;
      error = 'Connection lost. Reconnecting...';
    };
  });

  onDestroy(() => {
    if (es) es.close();
  });

  function clear() {
    logs = [];
  }

  function formatTime(t) {
    if (!t) return '-';
    return new Date(t).toLocaleString();
  }
</script>

<h1>Live logs</h1>

<div class="toolbar">
  <span class="status" class:connected>{connected ? 'Connected' : 'Disconnected'}</span>
  <button onclick={clear}>Clear</button>
</div>

{#if error}
  <div class="alert error">{error}</div>
{/if}

<div class="log-stream">
  {#each logs as log (log.dataId + log.time)}
    <div class="log-line level-{log.level}">
      <span class="time">{formatTime(log.time)}</span>
      <span class="badge">{log.level}</span>
      <span class="point">point={log.pointId}</span>
      <span class="data">{JSON.stringify(log.val)}</span>
    </div>
  {/each}
</div>

<style>
  h1 {
    margin-bottom: 1rem;
  }
  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }
  .status {
    padding: 0.35rem 0.75rem;
    border-radius: 4px;
    background: #fee2e2;
    color: #991b1b;
    font-size: 0.85rem;
  }
  .status.connected {
    background: #dcfce7;
    color: #166534;
  }
  button {
    background: #2563eb;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
  }
  .log-stream {
    background: #0f172a;
    color: #e2e8f0;
    border-radius: 12px;
    padding: 1rem;
    font-family: monospace;
    font-size: 0.85rem;
    max-height: 70vh;
    overflow-y: auto;
  }
  .log-line {
    display: grid;
    grid-template-columns: 140px 60px 90px 1fr;
    gap: 0.75rem;
    padding: 0.35rem 0;
    border-bottom: 1px solid #1e293b;
  }
  .time {
    color: #94a3b8;
  }
  .badge {
    text-align: center;
    padding: 0.1rem 0.35rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
  }
  .level-ERROR .badge { background: #7f1d1d; color: #fecaca; }
  .level-WARN .badge { background: #713f12; color: #fef08a; }
  .level-INFO .badge { background: #1e3a8a; color: #bfdbfe; }
  .level-DEBUG .badge { background: #334155; color: #cbd5e1; }
  .point {
    color: #94a3b8;
  }
  .data {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
