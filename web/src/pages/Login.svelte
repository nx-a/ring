<script>
  import { api, saveToken, saveRole } from '../api.js';

  let { onLogin } = $props();

  let login = $state('');
  let password = $state('');
  let isRegister = $state(false);
  let role = $state('viewer');
  let error = $state('');
  let loading = $state(false);

  async function submit(e) {
    e.preventDefault();
    error = '';
    loading = true;
    try {
      const res = isRegister
        ? await api.register(login, password, role)
        : await api.login(login, password);
      saveToken(res.token);
      saveRole(res.role || 'viewer');
      onLogin();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-box">
  <h1>{isRegister ? 'Register' : 'Login'}</h1>
  {#if error}
    <div class="error">{error}</div>
  {/if}
  <form onsubmit={submit}>
    <label>
      Login
      <input type="text" bind:value={login} required />
    </label>
    <label>
      Password
      <input type="password" bind:value={password} required />
    </label>
    {#if isRegister}
      <label>
        Role
        <select bind:value={role}>
          <option value="admin">Admin</option>
          <option value="viewer">Viewer (logs only)</option>
        </select>
      </label>
    {/if}
    <button type="submit" disabled={loading}>
      {loading ? '...' : isRegister ? 'Register' : 'Login'}
    </button>
  </form>
  <button class="toggle" type="button" onclick={() => isRegister = !isRegister}>
    {isRegister ? 'Already have an account? Login' : 'Create account'}
  </button>
</div>

<style>
  .login-box {
    max-width: 360px;
    margin: 10vh auto;
    background: white;
    padding: 2rem;
    border-radius: 12px;
    box-shadow: 0 4px 24px rgba(0,0,0,0.08);
  }
  h1 {
    margin-top: 0;
    font-size: 1.5rem;
  }
  form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    font-size: 0.9rem;
    color: #475569;
  }
  input, select {
    padding: 0.6rem 0.75rem;
    border: 1px solid #cbd5e1;
    border-radius: 6px;
    font-size: 1rem;
  }
  button[type="submit"] {
    background: #2563eb;
    color: white;
    border: none;
    padding: 0.75rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 1rem;
  }
  button[type="submit"]:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
  .toggle {
    margin-top: 1rem;
    background: transparent;
    border: none;
    color: #2563eb;
    cursor: pointer;
    text-decoration: underline;
  }
  .error {
    background: #fee2e2;
    color: #991b1b;
    padding: 0.75rem;
    border-radius: 6px;
    margin-bottom: 1rem;
    font-size: 0.9rem;
  }
</style>
