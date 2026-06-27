/**
 * api.js — Fetch wrapper with auth headers, CSRF token, and error handling.
 * All portal and public JS modules import from this file.
 */

/** Read the CSRF token from the meta tag injected by Go. */
function getCSRFToken() {
  return document.querySelector('meta[name="csrf-token"]')?.content ?? '';
}

/**
 * Core fetch wrapper.
 * @param {string} url
 * @param {RequestInit} options
 * @returns {Promise<any>} Parsed JSON response data
 * @throws {Error} With message from server error response
 */
async function request(url, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    'X-CSRF-Token': getCSRFToken(),
    ...options.headers,
  };

  const res = await fetch(url, {
    credentials: 'include', // send HTTP-only cookies
    ...options,
    headers,
  });

  // Handle 401 — redirect to login
  if (res.status === 401) {
    window.location.href = '/login';
    throw new Error('Session expired');
  }

  let body;
  try {
    body = await res.json();
  } catch {
    body = {};
  }

  if (!res.ok) {
    const message = body?.error || body?.message || `Request failed (${res.status})`;
    const err = new Error(message);
    err.status = res.status;
    err.errors = body?.errors; // validation errors map
    throw err;
  }

  return body?.data !== undefined ? body.data : body;
}

export const api = {
  /** GET request */
  get(url, params = {}) {
    const qs = new URLSearchParams(params).toString();
    return request(qs ? `${url}?${qs}` : url, { method: 'GET' });
  },

  /** POST request with JSON body */
  post(url, body) {
    return request(url, { method: 'POST', body: JSON.stringify(body) });
  },

  /** PUT request with JSON body */
  put(url, body) {
    return request(url, { method: 'PUT', body: JSON.stringify(body) });
  },

  /** DELETE request */
  delete(url) {
    return request(url, { method: 'DELETE' });
  },
};

export default api;
