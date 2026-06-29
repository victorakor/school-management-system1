/**
 * users.js — Staff / user management module
 * Updated to include all 15 roles from the LEAPS RBAC specification.
 */
import { api } from '../shared/api.js';
import { showToast } from '../shared/utils.js';

// All roles that can be created via the staff management screen.
// Students/Pupils are created via admissions. Parents self-register.
// OWNER is seeded at setup and never created here.
const STAFF_ROLES = [
  'PRINCIPAL',
  'VICE_PRINCIPAL',
  'HEAD_TEACHER',
  'ASST_HEAD_TEACHER',
  'EXAM_OFFICER',
  'ADMISSIONS_OFFICER',
  'BURSAR',
  'TEACHER',
  'CLASS_TEACHER',
  'BLOG_MANAGER',
  'ICT_ADMIN',
];

// Roles the filter dropdown shows (includes non-staff so the owner can
// search for any user type by role).
const ALL_FILTERABLE_ROLES = [
  ...STAFF_ROLES,
  'STUDENT',
  'PUPIL',
  'PARENT',
];

export async function initUsers(container, user) {
  // Only OWNER can create staff accounts. PRINCIPAL/HEAD/ICT_ADMIN can view list.
  const canCreate = user && user.role === 'OWNER';

  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Staff</h1>
      ${canCreate ? '<button id="add-user-btn" class="btn btn-primary">+ Add Staff Member</button>' : ''}
    </div>
    <div class="flex gap-2 mb-4">
      <select id="role-filter" class="input text-sm py-2">
        <option value="">All Roles</option>
        ${ALL_FILTERABLE_ROLES.map(r => `<option value="${r}">${formatRole(r)}</option>`).join('')}
      </select>
    </div>
    <div id="users-list"></div>
  `;

  // Only wire up the add-user modal for OWNER
  if (canCreate) {
    let modal = document.getElementById('add-user-modal');
  if (!modal) {
    modal = document.createElement('div');
    modal.id = 'add-user-modal';
    modal.style.cssText = 'display:none;position:fixed;inset:0;z-index:200;align-items:center;justify-content:center;';
    modal.innerHTML = `
      <div id="modal-backdrop" style="position:absolute;inset:0;background:rgba(0,0,0,0.45);backdrop-filter:blur(4px);"></div>
      <div style="position:relative;background:#fff;border-radius:1.25rem;box-shadow:0 25px 60px rgba(0,0,0,0.2);width:100%;max-width:480px;padding:1.5rem;z-index:201;margin:1rem;">
        <h2 style="font-size:1.125rem;font-weight:700;color:var(--color-primary);margin-bottom:1rem;">Add Staff Member</h2>
        <div style="display:grid;gap:0.75rem;">
          <div><label class="label">Full Name</label><input id="new-name" class="input w-full" placeholder="Full name" /></div>
          <div><label class="label">Email</label><input id="new-email" type="email" class="input w-full" placeholder="Email address" /></div>
          <div><label class="label">Phone</label><input id="new-phone" class="input w-full" placeholder="+234…" /></div>
          <div><label class="label">Role</label>
            <select id="new-role" class="input w-full">
              ${STAFF_ROLES.map(r => `<option value="${r}">${formatRole(r)}</option>`).join('')}
            </select>
          </div>
          <div id="division-row"><label class="label">Division Scope</label>
            <select id="new-division" class="input w-full">
              <option value="ALL">All Divisions</option>
              <option value="NURSERY">Nursery</option>
              <option value="PRIMARY">Primary</option>
              <option value="SECONDARY">Secondary</option>
            </select>
            <p id="division-hint" style="font-size:0.72rem;color:var(--color-text-secondary);margin-top:0.25rem;"></p>
          </div>
          <div><label class="label">Temporary Password</label><input id="new-password" type="password" class="input w-full" placeholder="Min. 8 characters" /></div>
        </div>
        <div style="display:flex;gap:0.75rem;margin-top:1.25rem;">
          <button id="cancel-add-btn" class="btn btn-ghost" style="flex:1;">Cancel</button>
          <button id="confirm-add-btn" class="btn btn-primary" style="flex:1;">Create Account</button>
        </div>
        <p id="add-error" style="color:var(--color-danger);font-size:0.75rem;margin-top:0.5rem;display:none;"></p>
      </div>
    `;
    document.body.appendChild(modal);
    }

    const openModal  = () => { modal.style.display = 'flex'; updateDivisionHint(); };
    const closeModal = () => { modal.style.display = 'none'; clearForm(); };

    document.getElementById('add-user-btn').addEventListener('click', openModal);
    modal.querySelector('#cancel-add-btn').addEventListener('click', closeModal);
    modal.querySelector('#modal-backdrop').addEventListener('click', closeModal);
    modal.querySelector('#confirm-add-btn').addEventListener('click', () => createUser(closeModal));

    // Update division hint when role changes
    modal.querySelector('#new-role').addEventListener('change', updateDivisionHint);

    document.addEventListener('keydown', function escHandler(e) {
      if (e.key === 'Escape' && modal.style.display === 'flex') closeModal();
    });
  } // end canCreate

  document.getElementById('role-filter').addEventListener('change', loadUsers);
  await loadUsers();
}

// Show a contextual hint about which division scope makes sense for the role.
function updateDivisionHint() {
  const role = document.getElementById('new-role')?.value;
  const hint = document.getElementById('division-hint');
  if (!hint) return;
  const hints = {
    PRINCIPAL:          'Secondary only — set to SECONDARY',
    VICE_PRINCIPAL:     'Secondary only — set to SECONDARY',
    HEAD_TEACHER:       'Nursery/Primary — set to PRIMARY or NURSERY',
    ASST_HEAD_TEACHER:  'Nursery/Primary — set to PRIMARY or NURSERY',
    BURSAR:             'Set to the division this bursar covers',
    EXAM_OFFICER:       'Usually ALL unless scoped to one division',
    ADMISSIONS_OFFICER: 'Usually ALL to handle all divisions',
    BLOG_MANAGER:       'Leave as ALL',
    ICT_ADMIN:          'Leave as ALL',
    TEACHER:            'Set to the division this teacher teaches in',
    CLASS_TEACHER:      'Set to the division this class teacher is in',
  };
  hint.textContent = hints[role] || '';
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

async function createUser(closeModal) {
  const errEl = document.getElementById('add-error');
  errEl.style.display = 'none';

  const body = {
    full_name:      document.getElementById('new-name').value.trim(),
    email:          document.getElementById('new-email').value.trim(),
    phone:          document.getElementById('new-phone').value.trim(),
    role:           document.getElementById('new-role').value,
    division_scope: document.getElementById('new-division').value,
    password:       document.getElementById('new-password').value,
  };

  if (!body.full_name || !body.email || !body.password) {
    errEl.textContent = 'Name, email and password are required.';
    errEl.style.display = 'block';
    return;
  }

  try {
    await api.post('/api/users', body);
    closeModal();
    showToast('Staff member created successfully.', 'success');
    await loadUsers();
  } catch (err) {
    errEl.textContent = err.message || 'Failed to create user.';
    errEl.style.display = 'block';
  }
}

function clearForm() {
  ['new-name','new-email','new-phone','new-password'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.value = '';
  });
  const errEl = document.getElementById('add-error');
  if (errEl) errEl.style.display = 'none';
  const hint = document.getElementById('division-hint');
  if (hint) hint.textContent = '';
}

function formatRole(r) {
  const m = {
    OWNER:              'School Owner',
    PRINCIPAL:          'Principal',
    VICE_PRINCIPAL:     'Vice Principal',
    HEAD_TEACHER:       'Head Teacher',
    ASST_HEAD_TEACHER:  'Asst. Head Teacher',
    EXAM_OFFICER:       'Exam Officer',
    ADMISSIONS_OFFICER: 'Admissions Officer',
    BURSAR:             'Bursar',
    TEACHER:            'Teacher',
    CLASS_TEACHER:      'Class Teacher',
    STUDENT:            'Student',
    PUPIL:              'Pupil',
    PARENT:             'Parent',
    BLOG_MANAGER:       'Blog / Media Manager',
    ICT_ADMIN:          'ICT Administrator',
  };
  return m[r] || r;
}