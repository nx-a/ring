<script>
  import { onMount } from 'svelte';
  import { api } from '../api.js';

  let status = $state(null);
  let metrics = $state(null);
  let error = $state('');
  let loading = $state(true);

  onMount(async () => {
    try {
      [status, metrics] = await Promise.all([api.getStatus(), api.getMetrics()]);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });

  function formatTime(ts) {
    if (!ts) return '-';
    return new Date(ts * 1000).toLocaleString();
  }
</script>

<h1>Status & Metrics</h1>

{#if loading}
  <div class="loading">Loading...</div>
{/if}
{#if error}
  <div class="alert error">{error}</div>
{/if}

<div class="grid">
  {#if status}
    <section class="card">
      <h2>System status</h2>
      <div class="metric">
        <span class="label">Started</span>
        <span class="value">{formatTime(status.startTime)}</span>
      </div>
      <div class="metric">
        <span class="label">TCP clients</span>
        <span class="value">{status.tcp.clients}</span>
      </div>
      <h3>Database pool</h3>
      <div class="metric">
        <span class="label">Total</span>
        <span class="value">{status.db.totalConns}</span>
      </div>
      <div class="metric">
        <span class="label">Idle</span>
        <span class="value">{status.db.idleConns}</span>
      </div>
      <div class="metric">
        <span class="label">Acquired</span>
        <span class="value">{status.db.acquiredConns}</span>
      </div>
      <div class="metric">
        <span class="label">Max</span>
        <span class="value">{status.db.maxConns}</span>
      </div>
    </section>
  {/if}

  {#if metrics}
    <section class="card">
      <h2>Metrics</h2>
      <div class="metric">
        <span class="label">Total logs</span>
        <span class="value">{metrics.total.toLocaleString()}</span>
      </div>
      <div class="metric">
        <span class="label">Buckets with data</span>
        <span class="value">{metrics.buckets}</span>
      </div>
      <div class="metric">
        <span class="label">TCP clients</span>
        <span class="value">{metrics.tcpClients}</span>
      </div>
      <h3>Logs per bucket</h3>
      {#if Object.keys(metrics.perBucket).length === 0}
        <div class="empty">No data</div>
      {:else}
        <ul class="list">
          {#each Object.entries(metrics.perBucket) as [bucketId, count]}
            <li>
              <span>Bucket {bucketId}</span>
              <span class="count">{count.toLocaleString()}</span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {/if}
</div>

<style>
  h1 {
    margin-bottom: 1rem;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 1.5rem;
  }
  .card {
    background: white;
    padding: 1.5rem;
    border-radius: 12px;
    box-shadow: 0 2px 12px rgba(0,0,0,0.06);
  }
  .card h2, .card h3 {
    margin-top: 0;
  }
  .card h3 {
    margin-top: 1.5rem;
    font-size: 1rem;
  }
  .metric {
    display: flex;
    justify-content: space-between;
    padding: 0.6rem 0;
    border-bottom: 1px solid #e2e8f0;
  }
  .label {
    color: #475569;
  }
  .value {
    font-weight: 600;
  }
  .list {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .list li {
    display: flex;
    justify-content: space-between;
    padding: 0.5rem 0;
    border-bottom: 1px solid #e2e8f0;
  }
  .count {
    font-weight: 600;
  }
  .empty {
    color: #64748b;
  }
  .loading {
    padding: 1rem 0;
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
