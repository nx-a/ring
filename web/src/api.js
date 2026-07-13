const API_URL = import.meta.env.DEV ? '/api' : '';

export function getToken() {
  return localStorage.getItem('ring_token');
}

export function getRole() {
  return localStorage.getItem('ring_role') || 'viewer';
}

function request(method, path, body) {
  const headers = {
    'Content-Type': 'application/json',
  };
  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const opts = {
    method,
    headers,
  };
  if (body !== undefined) {
    opts.body = JSON.stringify(body);
  }
  return fetch(`${API_URL}${path}`, opts).then(async res => {
    if (res.status === 401) {
      localStorage.removeItem('ring_token');
      localStorage.removeItem('ring_role');
      window.dispatchEvent(new Event('logout'));
      throw new Error('Unauthorized');
    }
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || `HTTP ${res.status}`);
    }
    if (res.status === 204 || res.headers.get('content-length') === '0') {
      return null;
    }
    return res.json();
  });
}

export const api = {
  login: (login, password) => request('POST', '/auth/in', { login, password }),
  register: (login, password, role) => request('POST', '/auth/reg', { login, password, role }),
  getBuckets: () => request('GET', '/bucket'),
  createBucket: (bucket) => request('POST', '/bucket', bucket),
  updateRetention: (id, timeLife) => request('PUT', `/bucket/${id}/retention`, { timeLife }),
  deleteBucket: (id) => request('DELETE', `/bucket/${id}`),
  getPoints: (bucketId) => request('GET', `/point/by/bucket/${bucketId}`),
  createPoint: (point) => request('POST', '/point', point),
  deletePoint: (id) => request('DELETE', `/point/${id}`),
  getTokens: (bucketId) => request('GET', `/token/by/bucket/${bucketId}`),
  createToken: (token) => request('POST', '/token', token),
  deleteToken: (id) => request('DELETE', `/token/${id}`),
  getLogs: (params) => {
    const query = new URLSearchParams();
    if (params.bucket) query.set('bucket', params.bucket);
    if (params.ext?.length) query.set('ext', params.ext);
    if (params.timeStart) query.set('timeStart', params.timeStart);
    if (params.timeEnd) query.set('timeEnd', params.timeEnd);
    if (params.level?.length) query.set('level', params.level);
    if (params.data?.length) query.set('data', params.data);
    if (params.limit) query.set('limit', String(params.limit));
    if (params.offset) query.set('offset', String(params.offset));
    return request('GET', `/data?${query.toString()}`);
  },
  exportUrl: (params, format) => {
    const query = new URLSearchParams();
    query.set('format', format);
    if (params.bucket) query.set('bucket', params.bucket);
    if (params.ext?.length) query.set('ext', params.ext);
    if (params.timeStart) query.set('timeStart', params.timeStart);
    if (params.timeEnd) query.set('timeEnd', params.timeEnd);
    if (params.level?.length) query.set('level', params.level);
    if (params.data?.length) query.set('data', params.data);
    return `${API_URL}/data/export?${query.toString()}`;
  },
  getStatus: () => request('GET', '/status'),
  getMetrics: () => request('GET', '/metrics'),
};

export function saveToken(token) {
  localStorage.setItem('ring_token', token);
}

export function saveRole(role) {
  localStorage.setItem('ring_role', role);
}

export function clearToken() {
  localStorage.removeItem('ring_token');
  localStorage.removeItem('ring_role');
}

export function createEventSource() {
  const base = import.meta.env.DEV ? '/api' : '';
  return new EventSource(`${base}/logs/stream?token=${encodeURIComponent(getToken())}`);
}
