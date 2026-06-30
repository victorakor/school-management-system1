/**
 * audit.js — Audit log viewer (Owner only)
 * Replaces: nothing (NEW FILE)
 * Place at: web/static/js/portal/audit.js
 */
import { api } from '../shared/api.js';
import { formatDate, showToast } from '../shared/utils.js';

let _state = { page: 1, total: 0, limit: 50, filters: {} };

export async function initAudit(container) {
  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-primary">Audit Logs</h1>
        <p class="text-text-secondary text-sm mt-1">A complete, tamper-proof record of every action taken in the system.</p>
      </div>
      <button id="export-audit-btn" class="btn btn-outline">
        <svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
        Export CSV
      </button>
    </div>

    <div class="card p-4 mb-4">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div>
          <label class="label text-xs">Action</label>
          <select id="filter-action" class="input text-sm py-1.5">
            <option value="">All Actions</option>
            <option value="login">Login</option>
            <option value="register">Register</option>
            <option value="reset_password">Reset Password</option>
            <option value="suspend_user">Suspend User</option>
            <option value="unlock_user">Unlock User</option>
            <option value="approve_score">Approve Score</option>
            <option value="reject_score">Reject Score</option>
            <option value="publish_results">Publish Results</option>
            <option value="override_result">Override Result</option>
            <option value="record_payment">Record Payment</option>
            <option value="override_admission">Override Admission</option>
            <option value="publish_quiz">Publish Quiz</option>
            <option value="close_quiz">Close Quiz</option>
            <option value="open_term">Open Term</option>
            <option value="close_term">Close Term</option>
            <option value="close_session">Close Session</option>
            <option value="publish_academic_calendar">Publish Calendar</option>
          </select>
        </div>
        <div>
          <label class="label text-xs">Entity Type</label>
          <select id="filter-entity" class="input text-sm py-1.5">
            <option value="">All Entities</option>
            <option value="user">User</option>
            <option value="score_entry">Score Entry</option>
            <option value="result">Result</option>
            <option value="fee_payment">Fee Payment</option>
            <option value="application">Application</option>
            <option value="quiz">Quiz</option>
            <option value="term">Term</option>
            <option value="academic_session">Session</option>
          </select>
        </div>
        <div>
          <label class="label text-xs">From Date</label>
          <input id="filter-from" type="date" class="input text-sm py-1.5" />
        </div>
        <div>
          <label class="label text-xs">To Date</label>
          <input id="filter-to" type="date" class="input text-sm py-1.5" />
        </div>
      </div>
      <div class="flex justify-end mt-3">
        <button id="apply-filters-btn" class="btn btn-primary btn-sm">Apply Filters</button>
        <button id="clear-filters-btn" class="btn btn-ghost btn-sm ml-2">Clear</button>
      </div>
    </div>

    <div id="audit-table-wrap" class="card overflow-hidden"></div>
    <div id="audit-pagination" class="flex items-center justify-between mt-4 text-sm text-text-secondary"></div>
  `;

  document.getElementById('export-audit-btn').addEventListener('click', exportCSV);
  document.getElementById('apply-filters-btn').addEventListener('click', () => { _state.page = 1; loadLogs(); });
  document.getElementById('clear-filters-btn').addEventListener('click', clearFilters);

  await loadLogs();
}

async function loadLogs() {
  const wrap = document.getElementById('audit-table-wrap');
  if (!wrap) return;

  wrap.innerHTML = `<div class="p-4 space-y-2">${Array(8).fill('<div class="skeleton h-10 rounded"></div>').join('')}</div>`;

  const params = new URLSearchParams({
    page: _state.page,
    limit: _state.limit,
    ...(document.getElementById('filter-action')?.value && { action: document.getElementById('filter-action').value }),
    ...(document.getElementById('filter-entity')?.value && { entity_type: document.getElementById('filter-entity').value }),
    ...(document.getElementById('filter-from')?.value && { from: document.getElementById('filter-from').value }),
    ...(document.getElementById('filter-to')?.value && { to: document.getElementById('filter-to').value }),
  });

  try {
    const data = await api.get(`/api/audit/logs?${params}`);
    const logs = data.logs || [];
    _state.total = data.total || 0;

    if (!logs.length) {
      wrap.innerHTML = `<div class="p-12 text-center text-text-secondary">No audit entries found for the selected filters.</div>`;
      renderPagination();
      return;
    }

    wrap.innerHTML = `
      <div class="overflow-x-auto">
        <table class="data-table w-full text-sm">
          <thead>
            <tr>
              <th class="whitespace-nowrap">Timestamp</th>
              <th>User</th>
              <th>Role</th>
              <th>Action</th>
              <th>Entity</th>
              <th>IP</th>
            </tr>
          </thead>
          <tbody>
            ${logs.map(l => `
              <tr>
                <td class="whitespace-nowrap font-mono text-xs text-text-secondary">${formatDate(new Date(l.created_at))}</td>
                <td class="font-medium text-primary">${l.user_name || '—'}</td>
                <td><span class="badge badge-neutral text-xs">${l.user_role || '—'}</span></td>
                <td><span class="badge ${actionBadge(l.action)} text-xs font-mono">${l.action}</span></td>
                <td class="text-text-secondary text-xs">${l.entity_type ? `${l.entity_type}${l.entity_id ? ` <span class="font-mono opacity-60">#${l.entity_id.slice(0,8)}</span>` : ''}` : '—'}</td>
                <td class="font-mono text-xs text-text-secondary">${l.ip_address || '—'}</td>
              </tr>`).join('')}
          </tbody>
        </table>
      </div>`;

    renderPagination();
  } catch (err) {
    wrap.innerHTML = `<div class="p-6 text-danger text-sm">${err.message}</div>`;
  }
}

function renderPagination() {
  const el = document.getElementById('audit-pagination');
  if (!el) return;
  const totalPages = Math.ceil(_state.total / _state.limit);
  const from = (_state.page - 1) * _state.limit + 1;
  const to = Math.min(_state.page * _state.limit, _state.total);
  el.innerHTML = `
    <span>${_state.total > 0 ? `Showing ${from}–${to} of ${_state.total} entries` : 'No entries'}</span>
    <div class="flex gap-2">
      <button id="prev-page" class="btn btn-ghost btn-sm" ${_state.page <= 1 ? 'disabled' : ''}>← Prev</button>
      <span class="px-3 py-1 text-primary font-medium">Page ${_state.page} / ${totalPages || 1}</span>
      <button id="next-page" class="btn btn-ghost btn-sm" ${_state.page >= totalPages ? 'disabled' : ''}>Next →</button>
    </div>`;
  document.getElementById('prev-page')?.addEventListener('click', () => { _state.page--; loadLogs(); });
  document.getElementById('next-page')?.addEventListener('click', () => { _state.page++; loadLogs(); });
}

function clearFilters() {
  ['filter-action', 'filter-entity', 'filter-from', 'filter-to'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.value = '';
  });
  _state.page = 1;
  loadLogs();
}

async function exportCSV() {
  const btn = document.getElementById('export-audit-btn');
  if (btn) btn.textContent = 'Exporting…';
  try {
    const res = await fetch('/api/audit/logs/export', {
      credentials: 'include',
      headers: { 'X-CSRF-Token': document.querySelector('meta[name="csrf-token"]')?.content || '' },
    });
    if (!res.ok) throw new Error('Export failed');
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit-logs-${new Date().toISOString().slice(0,10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
    showToast('Audit log exported.', 'success');
  } catch (err) {
    showToast(err.message || 'Export failed.', 'error');
  } finally {
    if (btn) btn.innerHTML = `<svg class="w-4 h-4 mr-2 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>Export CSV`;
  }
}

function actionBadge(action) {
  if (['override_result','override_admission','suspend_user'].includes(action)) return 'badge-danger';
  if (['publish_results','publish_quiz','approve_score'].includes(action)) return 'badge-success';
  if (['reset_password','unlock_user','close_quiz','close_term','close_session'].includes(action)) return 'badge-warning';
  return 'badge-neutral';
}
