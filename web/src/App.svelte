<script>
  import { onMount } from 'svelte';
  import Login from './pages/Login.svelte';
  import Management from './pages/Management.svelte';
  import Logs from './pages/Logs.svelte';
  import Live from './pages/Live.svelte';
  import Status from './pages/Status.svelte';
  import { getToken, getRole, clearToken } from './api.js';

  let page = $state('login');
  let token = $state(getToken());
  let role = $state(getRole());

  onMount(() => {
    const handler = () => {
      token = null;
      role = 'viewer';
      page = 'login';
    };
    window.addEventListener('logout', handler);
    if (token) page = role === 'admin' ? 'management' : 'logs';
    return () => window.removeEventListener('logout', handler);
  });

  function onLogin() {
    token = getToken();
    role = getRole();
    page = role === 'admin' ? 'management' : 'logs';
  }

  function logout() {
    clearToken();
    token = null;
    role = 'viewer';
    page = 'login';
  }

  function navigate(p) {
    page = p;
  }
</script>

<main>
  {#if token}
    <nav class="nav">
      <div class="brand">Ring Logs</div>
      <div class="links">
        {#if role === 'admin'}
          <button class="link" class:active={page === 'management'} onclick={() => navigate('management')}>Management</button>
        {/if}
        <button class="link" class:active={page === 'logs'} onclick={() => navigate('logs')}>Logs</button>
        <button class="link" class:active={page === 'live'} onclick={() => navigate('live')}>Live</button>
        <button class="link" class:active={page === 'status'} onclick={() => navigate('status')}>Status</button>
        <button class="link" onclick={logout}>Logout</button>
      </div>
    </nav>
  {/if}

  <div class="container">
    {#if page === 'login'}
      <Login onLogin={onLogin} />
    {:else if page === 'management'}
      <Management />
    {:else if page === 'logs'}
      <Logs />
    {:else if page === 'live'}
      <Live />
    {:else if page === 'status'}
      <Status />
    {/if}
  </div>
</main>

<style>
  :global(body) {
    margin: 0;
    font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #f5f7fa;
    color: #1a1a2e;
  }
  main {
    min-height: 100vh;
  }
  .nav {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1.5rem;
    background: #1a1a2e;
    color: white;
  }
  .brand {
    font-weight: 700;
    font-size: 1.25rem;
  }
  .links {
    display: flex;
    gap: 0.5rem;
  }
  .link {
    background: transparent;
    border: none;
    color: #cbd5e1;
    cursor: pointer;
    padding: 0.5rem 0.75rem;
    font-size: 0.95rem;
  }
  .link:hover, .link.active {
    color: white;
    text-decoration: underline;
  }
  .container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 1.5rem;
  }
</style>
