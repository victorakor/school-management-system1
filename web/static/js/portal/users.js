/**
 * users.js — Staff / user management module
 */
import { api } from '../shared/api.js';
import { showToast } from '../shared/utils.js';

const ROLES = ['PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','BURSAR','TEACHER'];

export async function initUsers(container, user) {
  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Staff</h1>
      <button id="add-user-btn" class="btn btn-primary">+ Add Staff Member</button>
    </div>
    <div class="flex gap-2 mb-4">
      <select id="role-filter" class="input text-sm py-2">
        <option value="">All Roles</option>
        ${ROLES.map(r => `<option value="${r}">${formatRole(r)}</option>`).join('')}
      </select>
    </div>
    <div id="users-list"></div>

    <!-- Add user modal -->
    <div id="add-user-modal" class="hidden fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/40 backdrop-blur-sm" id="modal-backdrop"></div>
      <div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-md p-6 z-10">
        <h2 class="font-display font-bold text-lg text-primary mb-4">Add Staff Member</h2>
        <div class="space-y-4">
          <div><label class="label">Full Name</label><input id="new-name" class="input w-full" placeholder="Full name" /></div>
          <div><label class="label">Email</label><input id="new-email" type="email" class="input w-full" placeholder="Email address" /></div>
          <div><label class="label">Phone</label><input id="new-phone" class="input w-full" placeholder="+234…" /></div>
          <div><label class="label">Role</label>
            <select id="new-role" class="input w-full">
              ${ROLES.map(r => `<option value="${r}">${formatRole(r)}</option>`).join('')}
            </select>
          </div>
          <div><label class="label">Division Scope</label>
            <select id="new-division" class="input w-full">
              <option value="ALL">All Divisions</option>
              <option value="NURSERY">Nursery</option>
              <option value="PRIMARY">Primary</option>
              <option value="SECONDARY">Secondary</option>
            </select>
          </div>
          <div><label class="label">Temporary Password</label><input id="new-password" type="password" class="input w-full" placeholder="Min. 8 characters" /></div>
        </div>
        <div class="flex gap-3 mt-6">
          <button id="cancel-add" class="btn btn-ghost flex-1">Cancel</button>
          <button id="confirm-add" class="btn btn-primary flex-1">Create Account</button>
        </div>
        <p id="add-error" class="text-danger text-xs mt-2 hidden"></p>
      </div>
    </div>
  `;

  document.getElementById('role-filter').addEventListener('change', loadUsers);
  document.getElementById('add-user-btn').addEventListener('click', () => document.getElementById('add-user-modal').classList.remove('hidden'));
  document.getElementById('cancel-add').addEventListener('click', () => document.getElementById('add-user-modal').classList.add('hidden'));
  document.getElementById('modal-backdrop').addEventListener('click', () => document.getElementById('add-user-modal').classList.add('hidden'));
  document.getElementById('confirm-add').addEventListener('click', createUser);
  await loadUsers();
}

async function loadUsers() {
  const list = document.getElementById('users-list');
  if (!list) return;
  const role = document.getElementById('role-filter')?.value || '';
  list.innerHTML = `<div class="card p-4 space-y-3">${Array(5).fill('<div class="skeleton h-12 rounded-lg"></div>').join('')}</div>`;

  try {
    const url = role ? `/api/users?role=${role}` : '/api/users';
    const data = await api.get(url);
    const users = data.users || data || [];

    list.innerHTML = users.length
      ? `<div class="card overflow-hidden">
          <table class="data-table w-full">
            <thead><tr><th>Name</th><th>Email</th><th>Role</th><th>Division</th><th>Status</th></tr></thead>
            <tbody>
              ${users.map(u => `
                <tr>
                  <td class="font-medium text-primary">${u.full_name}</td>
                  <td class="text-text-secondary text-sm">${u.email}</td>
                  <td><span class="badge badge-neutral text-xs">${formatRole(u.role)}</span></td>
                  <td class="text-text-secondary text-sm">${u.division_scope || '—'}</td>
                  <td><span class="badge ${u.is_active ? 'badge-success' : 'badge-warning'} text-xs">${u.is_active ? 'Active' : 'Inactive'}</span></td>
                </tr>`).join('')}
            </tbody>
          </table>
        </div>`
      : `<div class="card p-12 text-center text-text-secondary">No staff found.</div>`;
  } catch (err) {
    list.innerHTML = `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
  }
}

async function createUser() {
  const errEl = document.getElementById('add-error');
  errEl.classList.add('hidden');
  const body = {
    full_name: document.getElementById('new-name').value.trim(),
    email: document.getElementById('new-email').value.trim(),
    phone: document.getElementById('new-phone').value.trim(),
    role: document.getElementById('new-role').value,
    division_scope: document.getElementById('new-division').value,
    password: document.getElementById('new-password').value,
  };
  if (!body.full_name || !body.email || !body.password) {
    errEl.textContent = 'Name, email and password are required.';
    errEl.classList.remove('hidden');
    return;
  }
  try {
    await api.post('/api/users', body);
    document.getElementById('add-user-modal').classList.add('hidden');
    showToast('Staff member created successfully.', 'success');
    await loadUsers();
  } catch (err) {
    errEl.textContent = err.message || 'Failed to create user.';
    errEl.classList.remove('hidden');
  }
}

function formatRole(r) {
  const m = { PRINCIPAL:'Principal', VICE_PRINCIPAL:'Vice Principal', HEAD_TEACHER:'Head Teacher', ASST_HEAD_TEACHER:'Asst. Head Teacher', BURSAR:'Bursar', TEACHER:'Teacher' };
  return m[r] || r;
}
