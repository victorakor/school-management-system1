/**
 * admissions.js — Admissions portal module
 * Handles application listing, status updates, and appointment management.
 */
import { api } from '../shared/api.js';
import { formatDate, showToast } from '../shared/utils.js';

export async function initAdmissions(container) {
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
  await loadApplications();
}

async function loadApplications() {
  const status = document.getElementById('status-filter')?.value || '';
  const division = document.getElementById('division-filter')?.value || '';
  const list = document.getElementById('applications-list');
  list.innerHTML = skeletonRows(5);

  try {
    let url = '/api/admissions/applications?';
    if (status) url += `status=${status}&`;
    if (division) url += `division=${division}&`;
    const data = await api.get(url);

    if (!data.length) {
      list.innerHTML = `<div class="card p-12 text-center text-secondary">No applications found.</div>`;
      return;
    }

    list.innerHTML = data.map(app => `
      <div class="card p-4 flex items-center justify-between hover:shadow-md transition-shadow cursor-pointer"
           onclick="window._admissionsOpenApp('${app.id}')">
        <div class="flex items-center gap-4">
          <div class="w-10 h-10 rounded-full bg-accent/10 flex items-center justify-center text-accent font-bold">
            ${app.child_name.charAt(0)}
          </div>
          <div>
            <p class="font-semibold text-primary">${app.child_name}</p>
            <p class="text-xs text-secondary">${app.division} · Ref: <span class="font-mono">${app.ref_number}</span></p>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <span class="text-xs text-secondary">${formatDate(app.created_at)}</span>
          <span class="badge badge-${statusColor(app.status)}">${app.status.replace(/_/g, ' ')}</span>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-secondary">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
        </div>
      </div>
    `).join('');

    window._admissionsOpenApp = (id) => openApplicationModal(id);
  } catch (err) {
    list.innerHTML = `<div class="card p-6 text-danger text-sm">Failed to load applications.</div>`;
  }
}

async function openApplicationModal(id) {
  const modal = document.getElementById('application-modal');
  modal.className = 'fixed inset-0 z-50 flex items-center justify-center p-4';
  modal.innerHTML = `
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" onclick="window._admissionsCloseModal()"></div>
    <div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto z-10 p-6"
         style="animation: modalIn 300ms ease-out">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold text-primary">Application Details</h2>
        <button onclick="window._admissionsCloseModal()" class="text-secondary hover:text-primary">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
      <div id="modal-content" class="space-y-4">
        <div class="skeleton h-6 w-48 rounded"></div>
        <div class="skeleton h-4 w-full rounded"></div>
        <div class="skeleton h-4 w-3/4 rounded"></div>
      </div>
    </div>`;

  window._admissionsCloseModal = () => {
    modal.className = 'hidden';
    modal.innerHTML = '';
  };

  try {
    const app = await api.get(`/api/admissions/applications/${id}`);
    document.getElementById('modal-content').innerHTML = `
      <div class="grid grid-cols-2 gap-4">
        <div><p class="label">Child Name</p><p class="font-medium">${app.child_name}</p></div>
        <div><p class="label">Division</p><p class="font-medium">${app.division}</p></div>
        <div><p class="label">Date of Birth</p><p class="font-medium">${formatDate(app.child_dob)}</p></div>
        <div><p class="label">Gender</p><p class="font-medium">${app.child_gender}</p></div>
        <div><p class="label">Reference</p><p class="font-mono font-medium">${app.ref_number}</p></div>
        <div><p class="label">Status</p><span class="badge badge-${statusColor(app.status)}">${app.status.replace(/_/g, ' ')}</span></div>
        ${app.appointment_date ? `<div><p class="label">Appointment</p><p class="font-medium">${formatDate(app.appointment_date)} at ${app.appointment_time}</p></div>` : ''}
        ${app.medical_conditions ? `<div class="col-span-2"><p class="label">Medical Conditions</p><p class="text-sm">${app.medical_conditions}</p></div>` : ''}
      </div>
      <div class="flex gap-2 mt-6 pt-4 border-t border-neutral-100">
        ${app.status === 'PENDING' || app.status === 'UNDER_REVIEW' ? `
          <button class="btn btn-success flex-1" onclick="window._admissionsUpdateStatus('${app.id}', 'ACCEPTED')">Accept</button>
          <button class="btn btn-danger flex-1" onclick="window._admissionsUpdateStatus('${app.id}', 'DECLINED')">Decline</button>
          <button class="btn btn-secondary flex-1" onclick="window._admissionsUpdateStatus('${app.id}', 'UNDER_REVIEW')">Mark Under Review</button>
        ` : ''}
        ${app.passport_url ? `<a href="${app.passport_url}" target="_blank" class="btn btn-outline flex-1">View Passport</a>` : ''}
      </div>`;

    window._admissionsUpdateStatus = async (appId, status) => {
      try {
        await api.put(`/api/admissions/applications/${appId}/status`, { status });
        showToast(`Application ${status.toLowerCase().replace(/_/g, ' ')}`, 'success');
        window._admissionsCloseModal();
        loadApplications();
      } catch {
        showToast('Failed to update status', 'error');
      }
    };
  } catch {
    document.getElementById('modal-content').innerHTML = `<p class="text-danger">Failed to load application details.</p>`;
  }
}

function statusColor(status) {
  const map = { PENDING: 'warning', UNDER_REVIEW: 'warning', ACCEPTED: 'success', DECLINED: 'danger' };
  return map[status] || 'neutral';
}

function skeletonRows(n) {
  return Array(n).fill(0).map(() =>
    `<div class="card p-4 flex items-center gap-4">
      <div class="skeleton w-10 h-10 rounded-full"></div>
      <div class="flex-1 space-y-2">
        <div class="skeleton h-4 w-48 rounded"></div>
        <div class="skeleton h-3 w-32 rounded"></div>
      </div>
    </div>`
  ).join('');
}
