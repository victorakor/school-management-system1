/**
 * admissions.js — Admissions portal module
 * REPLACE: web/static/js/portal/admissions.js
 *
 * Changes from original:
 *   - Owner gets an "Override Decision" button per application card
 *   - Override modal forces ACCEPTED or DECLINED with mandatory justification reason
 *   - Original approve/decline/under-review workflow preserved for all other eligible roles
 */
import { api } from '../shared/api.js';
import { formatDate, showToast } from '../shared/utils.js';

const CAN_APPROVE = ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER'];
const CAN_PROCESS = [...CAN_APPROVE, 'ADMISSIONS_OFFICER'];

let _user = null;

export async function initAdmissions(container, user) {
  _user = user;
  if (user?.role === 'PARENT') return initParentAdmissions(container);

  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Admissions</h1>
      <div class="flex gap-2">
        <select id="status-filter" class="input text-sm py-2">
          <option value="">All Statuses</option>
          <option value="PENDING">Pending</option>
          <option value="UNDER_REVIEW">Under Review</option>
          <option value="ACCEPTED">Accepted</option>
          <option value="DECLINED">Declined</option>
        </select>
        <select id="division-filter" class="input text-sm py-2">
          <option value="">All Divisions</option>
          <option value="NURSERY">Nursery</option>
          <option value="PRIMARY">Primary</option>
          <option value="SECONDARY">Secondary</option>
        </select>
      </div>
    </div>
    <div id="applications-list" class="space-y-3"></div>
    <div id="application-modal" class="hidden"></div>
  `;

  document.getElementById('status-filter').addEventListener('change', loadApplications);
  document.getElementById('division-filter').addEventListener('change', loadApplications);

  if (user?.role === 'OWNER') mountOverrideAdmissionModal();

  await loadApplications();
}

async function loadApplications() {
  const list     = document.getElementById('applications-list');
  if (!list) return;
  const status   = document.getElementById('status-filter')?.value || '';
  const division = document.getElementById('division-filter')?.value || '';
  list.innerHTML = skeletonRows(4);

  const isOwner    = _user?.role === 'OWNER';
  const canApprove = CAN_APPROVE.includes(_user?.role);
  const canProcess = CAN_PROCESS.includes(_user?.role);

  try {
    const params = new URLSearchParams({ ...(status && { status }), ...(division && { division }) });
    const data   = await api.get(`/api/admissions/applications?${params}`);
    const apps   = data.applications || data || [];

    if (!apps.length) {
      list.innerHTML = `<div class="card p-12 text-center text-text-secondary">No applications found.</div>`;
      return;
    }

    list.innerHTML = apps.map(app => `
      <div class="card p-5">
        <div class="flex items-start justify-between gap-4 flex-wrap">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-3 mb-2 flex-wrap">
              <h3 class="font-semibold text-primary">${app.child_name}</h3>
              <span class="badge ${statusBadge(app.status)} text-xs">${app.status.replace('_',' ')}</span>
              <span class="badge badge-neutral text-xs">${app.division}</span>
            </div>
            <p class="text-sm text-text-secondary mb-1">Ref: <span class="font-mono">${app.ref_number}</span></p>
            <p class="text-xs text-text-secondary">Parent: ${app.parent_name} · Applied: ${formatDate(new Date(app.created_at))}</p>
            ${app.decline_reason ? `<p class="text-xs text-danger mt-1">Declined: ${app.decline_reason}</p>` : ''}
          </div>
          <div class="flex gap-2 flex-wrap shrink-0">
            ${canProcess && app.status === 'PENDING' ? `
              <button class="btn btn-sm btn-ghost mark-review-btn" data-id="${app.id}">Mark Under Review</button>` : ''}
            ${canApprove && ['PENDING','UNDER_REVIEW'].includes(app.status) ? `
              <button class="btn btn-sm btn-primary approve-btn" data-id="${app.id}" data-name="${app.child_name}">Approve</button>
              <button class="btn btn-sm btn-danger-ghost decline-btn" data-id="${app.id}" data-name="${app.child_name}">Decline</button>` : ''}
            ${isOwner ? `
              <button class="btn btn-sm btn-outline override-admission-btn"
                data-id="${app.id}" data-name="${app.child_name}" data-status="${app.status}">Override</button>` : ''}
          </div>
        </div>
      </div>`).join('');

    // Bind action buttons
    list.querySelectorAll('.mark-review-btn').forEach(btn =>
      btn.addEventListener('click', () => updateStatus(btn.dataset.id, 'UNDER_REVIEW')));
    list.querySelectorAll('.approve-btn').forEach(btn =>
      btn.addEventListener('click', () => updateStatus(btn.dataset.id, 'ACCEPTED', btn.dataset.name)));
    list.querySelectorAll('.decline-btn').forEach(btn =>
      btn.addEventListener('click', () => promptDecline(btn.dataset.id, btn.dataset.name)));
    list.querySelectorAll('.override-admission-btn').forEach(btn =>
      btn.addEventListener('click', () => openOverrideAdmissionModal(btn.dataset.id, btn.dataset.name, btn.dataset.status)));

  } catch (err) {
    list.innerHTML = `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
  }
}

async function updateStatus(appId, status, name = '') {
  const label = status === 'ACCEPTED' ? 'approve' : 'mark as under review';
  if (status === 'ACCEPTED' && !confirm(`Approve application for ${name}?`)) return;
  try {
    await api.put(`/api/admissions/applications/${appId}`, { status });
    showToast(`Application ${status.toLowerCase().replace('_',' ')}.`, 'success');
    await loadApplications();
  } catch (err) {
    showToast(err.message || `Failed to ${label}.`, 'error');
  }
}

async function promptDecline(appId, name) {
  const reason = prompt(`Reason for declining ${name}'s application (required):`);
  if (!reason?.trim()) return;
  try {
    await api.put(`/api/admissions/applications/${appId}`, { status: 'DECLINED', decline_reason: reason.trim() });
    showToast('Application declined.', 'success');
    await loadApplications();
  } catch (err) {
    showToast(err.message || 'Failed to decline application.', 'error');
  }
}

// ─── Override Admission Modal ─────────────────────────────────────────────────

function mountOverrideAdmissionModal() {
  if (document.getElementById('override-admission-modal')) return;
  const modal = document.createElement('div');
  modal.id = 'override-admission-modal';
  modal.style.cssText = 'display:none;position:fixed;inset:0;z-index:200;align-items:center;justify-content:center;';
  modal.innerHTML = `
    <div id="oa-backdrop" style="position:absolute;inset:0;background:rgba(0,0,0,0.45);backdrop-filter:blur(4px);"></div>
    <div style="position:relative;background:#fff;border-radius:1.25rem;box-shadow:0 25px 60px rgba(0,0,0,0.2);width:100%;max-width:460px;padding:1.5rem;z-index:201;margin:1rem;">
      <h2 style="font-size:1.125rem;font-weight:700;color:var(--color-primary);margin-bottom:0.25rem;">Override Admission Decision</h2>
      <p id="oa-subtitle" style="font-size:0.8rem;color:var(--color-text-secondary);margin-bottom:1rem;"></p>
      <div class="p-3 rounded-xl mb-4" style="background:#fef9c3;border:1px solid #fde68a;">
        <p style="font-size:0.78rem;color:#92400e;">⚠️ This bypasses the normal workflow and is permanently audit-logged.</p>
      </div>
      <div style="display:grid;gap:0.75rem;">
        <div>
          <label class="label">New Decision <span style="color:var(--color-danger)">*</span></label>
          <select id="oa-status" class="input w-full">
            <option value="ACCEPTED">Accept</option>
            <option value="DECLINED">Decline</option>
          </select>
        </div>
        <div id="oa-decline-reason-row" style="display:none;">
          <label class="label">Decline Reason <span style="color:var(--color-danger)">*</span></label>
          <input id="oa-decline-reason" class="input w-full" placeholder="Reason for declining" />
        </div>
        <div>
          <label class="label">Override Justification <span style="color:var(--color-danger)">*</span></label>
          <input id="oa-reason" class="input w-full" placeholder="Why are you overriding this decision?" />
        </div>
      </div>
      <p id="oa-error" style="color:var(--color-danger);font-size:0.75rem;margin-top:0.5rem;display:none;"></p>
      <div style="display:flex;gap:0.75rem;margin-top:1.25rem;">
        <button id="cancel-oa-btn" class="btn btn-ghost" style="flex:1;">Cancel</button>
        <button id="confirm-oa-btn" class="btn btn-danger" style="flex:1;">Confirm Override</button>
      </div>
    </div>`;
  document.body.appendChild(modal);
  modal.querySelector('#oa-backdrop').addEventListener('click', () => { modal.style.display = 'none'; });
  modal.querySelector('#cancel-oa-btn').addEventListener('click', () => { modal.style.display = 'none'; });
  modal.querySelector('#oa-status').addEventListener('change', e => {
    document.getElementById('oa-decline-reason-row').style.display =
      e.target.value === 'DECLINED' ? 'block' : 'none';
  });
}

function openOverrideAdmissionModal(appId, childName, currentStatus) {
  const modal = document.getElementById('override-admission-modal');
  if (!modal) return;
  document.getElementById('oa-subtitle').textContent = `Application for: ${childName} (currently: ${currentStatus.replace('_',' ')})`;
  document.getElementById('oa-status').value = 'ACCEPTED';
  document.getElementById('oa-decline-reason-row').style.display = 'none';
  document.getElementById('oa-decline-reason').value = '';
  document.getElementById('oa-reason').value = '';
  document.getElementById('oa-error').style.display = 'none';
  modal.style.display = 'flex';

  const oldBtn = document.getElementById('confirm-oa-btn');
  const newBtn = oldBtn.cloneNode(true);
  oldBtn.parentNode.replaceChild(newBtn, oldBtn);
  newBtn.addEventListener('click', async () => {
    const status        = document.getElementById('oa-status').value;
    const declineReason = document.getElementById('oa-decline-reason').value.trim();
    const reason        = document.getElementById('oa-reason').value.trim();
    const errEl         = document.getElementById('oa-error');
    errEl.style.display = 'none';
    if (!reason) {
      errEl.textContent = 'Override justification is required.';
      errEl.style.display = 'block';
      return;
    }
    if (status === 'DECLINED' && !declineReason) {
      errEl.textContent = 'Decline reason is required when declining.';
      errEl.style.display = 'block';
      return;
    }
    try {
      await api.put(`/api/admissions/applications/${appId}/override`, {
        status, decline_reason: declineReason, reason,
      });
      modal.style.display = 'none';
      showToast('Admission decision overridden and logged.', 'success');
      await loadApplications();
    } catch (err) {
      errEl.textContent = err.message || 'Failed to override decision.';
      errEl.style.display = 'block';
    }
  });
}

// ─── Parent View (unchanged) ──────────────────────────────────────────────────

async function initParentAdmissions(container) {
  container.innerHTML = `
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-primary">My Applications</h1>
      <p class="text-text-secondary text-sm mt-1">Track the status of your child's admission application.</p>
    </div>
    <div id="parent-applications-list" class="space-y-3"></div>`;

  const list = document.getElementById('parent-applications-list');
  list.innerHTML = skeletonRows(3);
  try {
    const data = await api.get('/api/admissions/my-applications');
    const apps = data.applications || data || [];
    if (!apps.length) {
      list.innerHTML = `<div class="card p-12 text-center text-text-secondary">No applications found. Visit the <a href="/admissions" class="text-accent underline">Admissions page</a> to apply.</div>`;
      return;
    }
    list.innerHTML = apps.map(app => `
      <div class="card p-5">
        <div class="flex items-center justify-between mb-3">
          <h3 class="font-semibold text-primary">${app.child_name}</h3>
          <span class="badge ${statusBadge(app.status)} text-xs">${app.status.replace('_',' ')}</span>
        </div>
        <p class="text-xs text-text-secondary">Ref: <span class="font-mono">${app.ref_number}</span> · Division: ${app.division}</p>
        <p class="text-xs text-text-secondary mt-1">Applied: ${formatDate(new Date(app.created_at))}</p>
      </div>`).join('');
  } catch (err) {
    list.innerHTML = `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
  }
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function statusBadge(s) {
  return { ACCEPTED:'badge-success', DECLINED:'badge-danger', UNDER_REVIEW:'badge-warning', PENDING:'badge-neutral' }[s] || 'badge-neutral';
}

function skeletonRows(n) {
  return Array(n).fill('<div class="skeleton h-24 rounded-2xl"></div>').join('');
}
