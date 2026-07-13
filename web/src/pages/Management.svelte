<script>
  import { onMount } from 'svelte';
  import { api } from '../api.js';

  let buckets = $state([]);
  let points = $state([]);
  let tokens = $state([]);
  let selectedBucket = $state(null);
  let error = $state('');
  let message = $state('');

  let newBucket = $state({ systemName: '', timeLife: 240 });
  let newPoint = $state({ bucket: 0, ext: '', timeZone: '' });
  let newToken = $state({ bucket: 0, type: 1 });

  const tokenTypes = [
    { value: 1, label: 'Write' },
    { value: 2, label: 'Read' },
    { value: 3, label: 'Full' },
  ];

  onMount(loadBuckets);

  async function loadBuckets() {
    try {
      buckets = await api.getBuckets();
      error = '';
    } catch (e) {
      error = e.message;
    }
  }

  async function selectBucket(bucket) {
    selectedBucket = bucket;
    newPoint.bucket = bucket.bucketId;
    newToken.bucket = bucket.bucketId;
    await Promise.all([loadPoints(), loadTokens()]);
  }

  async function loadPoints() {
    if (!selectedBucket) return;
    try {
      points = await api.getPoints(selectedBucket.bucketId);
    } catch (e) {
      error = e.message;
    }
  }

  async function loadTokens() {
    if (!selectedBucket) return;
    try {
      tokens = await api.getTokens(selectedBucket.bucketId);
    } catch (e) {
      error = e.message;
    }
  }

  async function createBucket(e) {
    e.preventDefault();
    try {
      await api.createBucket({
        systemName: newBucket.systemName,
        timeLife: Number(newBucket.timeLife),
      });
      newBucket = { systemName: '', timeLife: 240 };
      message = 'Bucket created';
      await loadBuckets();
    } catch (e) {
      error = e.message;
    }
  }

  async function deleteBucket(id) {
    try {
      await api.deleteBucket(id);
      selectedBucket = null;
      points = [];
      tokens = [];
      message = 'Bucket deleted';
      await loadBuckets();
    } catch (e) {
      error = e.message;
    }
  }

  async function createPoint(e) {
    e.preventDefault();
    try {
      await api.createPoint({
        bucket: newPoint.bucket,
        ext: newPoint.ext.trim(),
        timeZone: newPoint.timeZone,
      });
      newPoint.ext = '';
      newPoint.timeZone = '';
      message = 'Point created';
      await loadPoints();
    } catch (e) {
      error = e.message;
    }
  }

  async function deletePoint(id) {
    try {
      await api.deletePoint(id);
      message = 'Point deleted';
      await loadPoints();
    } catch (e) {
      error = e.message;
    }
  }

  async function createToken(e) {
    e.preventDefault();
    try {
      const t = await api.createToken({
        bucket: newToken.bucket,
        type: Number(newToken.type),
      });
      newToken.type = 1;
      message = `Token created: ${t.token}`;
      await loadTokens();
    } catch (e) {
      error = e.message;
    }
  }

  async function updateRetention(e) {
    e.preventDefault();
    try {
      await api.updateRetention(selectedBucket.bucketId, Number(selectedBucket.timeLife));
      message = 'Retention updated';
      await loadBuckets();
    } catch (e) {
      error = e.message;
    }
  }

  async function deleteToken(id) {
    try {
      await api.deleteToken(id);
      message = 'Token deleted';
      await loadTokens();
    } catch (e) {
      error = e.message;
    }
  }
</script>

<h1>Management</h1>

{#if error}
  <div class="alert error">{error}</div>
{/if}
{#if message}
  <div class="alert success">{message}</div>
{/if}

<div class="grid">
  <section class="card">
    <h2>Buckets</h2>
    <form onsubmit={createBucket}>
      <label>
        Name
        <input type="text" bind:value={newBucket.systemName} required placeholder="my-app" />
      </label>
      <label>
        Time life (hours)
        <input type="number" bind:value={newBucket.timeLife} required />
      </label>
      <button type="submit">Create bucket</button>
    </form>
    <ul class="list">
      {#each buckets as b}
        <li class:selected={selectedBucket?.bucketId === b.bucketId}>
          <button class="select" onclick={() => selectBucket(b)}>
            {b.systemName}
          </button>
          <button class="danger" onclick={() => deleteBucket(b.bucketId)}>Delete</button>
        </li>
      {/each}
    </ul>
  </section>

  {#if selectedBucket}
    <section class="card">
      <h2>Settings for {selectedBucket.systemName}</h2>
      <form onsubmit={updateRetention}>
        <label>
          Retention time (hours)
          <input type="number" bind:value={selectedBucket.timeLife} required />
        </label>
        <button type="submit">Update retention</button>
      </form>
    </section>

    <section class="card">
      <h2>Points for {selectedBucket.systemName}</h2>
      <form onsubmit={createPoint}>
        <label>
          External ID
          <input type="text" bind:value={newPoint.ext} required />
        </label>
        <label>
          Time zone
          <input type="text" bind:value={newPoint.timeZone} placeholder="UTC" />
        </label>
        <button type="submit">Create point</button>
      </form>
      <ul class="list">
        {#each points as p}
          <li>
            <span>{p.externalId}</span>
            <button class="danger" onclick={() => deletePoint(p.pointId)}>Delete</button>
          </li>
        {/each}
      </ul>
    </section>

    <section class="card">
      <h2>Tokens for {selectedBucket.systemName}</h2>
      <form onsubmit={createToken}>
        <label>
          Type
          <select bind:value={newToken.type}>
            {#each tokenTypes as t}
              <option value={t.value}>{t.label}</option>
            {/each}
          </select>
        </label>
        <button type="submit">Create token</button>
      </form>
      <ul class="list">
        {#each tokens as t}
          <li>
            <span class="token">{t.token}</span>
            <span class="badge">{tokenTypes.find(x => x.value === t.type)?.label}</span>
            <button class="danger" onclick={() => deleteToken(t.tokenId)}>Delete</button>
          </li>
        {/each}
      </ul>
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
  .card h2 {
    margin-top: 0;
    font-size: 1.15rem;
  }
  form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 0.85rem;
    color: #475569;
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
    padding: 0.6rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9rem;
  }
  button.danger {
    background: #dc2626;
  }
  .list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .list li {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
    background: #f8fafc;
    border-radius: 6px;
  }
  .list li.selected {
    background: #dbeafe;
  }
  button.select {
    background: transparent;
    color: #1e293b;
    padding: 0;
    flex: 1;
    text-align: left;
  }
  .token {
    font-family: monospace;
    font-size: 0.8rem;
    word-break: break-all;
    flex: 1;
  }
  .badge {
    background: #e2e8f0;
    color: #475569;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
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
  .success {
    background: #dcfce7;
    color: #166534;
  }
</style>
