/**
 * settings.js — School settings module
 *
 * Role capabilities per RBAC spec:
 *   OWNER     — full school identity, branding, grading config, templates
 *   ICT_ADMIN — system configuration only: email/SMS config, password reset
 *               info, backup status. Cannot see or edit school identity/finances.
 */
import { api } from '../shared/api.js';
import { showToast } from '../shared/utils.js';

export async function initSettings(container, user) {
  const role = user?.role || window.PORTAL_ROLE || '';

  if (role === 'ICT_ADMIN') {
    return initICTSettings(container);
  }

  if (role === 'OWNER') {
    return initOwnerSettings(container);
  }

  // Any other role that somehow lands here gets a hard block.
  container.innerHTML = `
    <div class="flex flex-col items-center justify-center h-64 gap-4">
      <div class="w-16 h-16 rounded-2xl bg-danger/10 flex items-center justify-center">
        <svg class="w-8 h-8 text-danger" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
            d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
        </svg>
      </div>
      <div class="text-center">
        <p class="font-semibold text-primary">Access Denied</p>
        <p class="text-sm text-text-secondary mt-1">You do not have permission to access Settings.</p>
      </div>
    </div>`;
}

// ─── Owner: full school settings ──────────────────────────────────────────────

async function initOwnerSettings(container) {
  container.innerHTML = `
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-primary">School Settings</h1>
      <p class="text-text-secondary text-sm mt-1">Manage your school's identity, branding, and configuration.</p>
    </div>
    <div id="settings-loading" class="card p-6 space-y-3">
      ${Array(6).fill('<div class="skeleton h-10 rounded-lg"></div>').join('')}
    </div>
    <form id="settings-form" class="hidden space-y-6">
      <div class="card p-6">
        <h2 class="font-semibold text-primary mb-4">School Identity</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="md:col-span-2"><label class="label">School Name</label><input id="s-name" class="input w-full" /></div>
          <div class="md:col-span-2"><label class="label">Motto</label><input id="s-motto" class="input w-full" /></div>
          <div class="md:col-span-2"><label class="label">Address</label><input id="s-address" class="input w-full" /></div>
          <div><label class="label">Phone</label><input id="s-phone" class="input w-full" /></div>
          <div><label class="label">Email</label><input id="s-email" type="email" class="input w-full" /></div>
          <div><label class="label">School Prefix (for Admission IDs)</label><input id="s-prefix" class="input w-full" maxlength="5" /></div>
          <div><label class="label">Year Established</label><input id="s-established-year" type="number" class="input w-full" placeholder="e.g. 2010" min="1900" max="${new Date().getFullYear()}" /></div>
          <div><label class="label">Brand Color</label>
            <div class="flex gap-2 items-center">
              <input id="s-color" type="color" class="h-10 w-16 rounded-lg border border-neutral-200 p-1 cursor-pointer" />
              <input id="s-color-hex" class="input flex-1 font-mono text-sm" placeholder="#0F2557" />
            </div>
          </div>
        </div>
      </div>

      <div class="card p-6">
        <h2 class="font-semibold text-primary mb-4">Branding Assets</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label class="label">School Logo</label>
            <p class="text-xs text-text-secondary mb-2">Shown on portal, documents, and emails.</p>
            <div id="logo-preview" class="w-20 h-20 rounded-xl border-2 border-dashed border-neutral-200 flex items-center justify-center mb-2">
              <svg class="w-8 h-8 text-neutral-300" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
            </div>
            <input id="s-logo-url" class="input w-full text-sm font-mono" placeholder="https://res.cloudinary.com/…" />
            <p class="text-xs text-text-secondary mt-1">Paste a Cloudinary URL after uploading.</p>
          </div>
        </div>
      </div>

      <div class="flex justify-end gap-3">
        <button type="button" id="cancel-settings" class="btn btn-ghost">Discard</button>
        <button type="submit" class="btn btn-primary">Save Settings</button>
      </div>
      <p id="settings-error" class="text-danger text-sm hidden"></p>
    </form>
  `;

  await loadOwnerSettings();

  document.getElementById('s-color').addEventListener('input', e => {
    document.getElementById('s-color-hex').value = e.target.value;
  });
  document.getElementById('s-color-hex').addEventListener('input', e => {
    if (/^#[0-9a-fA-F]{6}$/.test(e.target.value)) {
      document.getElementById('s-color').value = e.target.value;
    }
  });
  document.getElementById('settings-form').addEventListener('submit', saveOwnerSettings);
  document.getElementById('cancel-settings').addEventListener('click', loadOwnerSettings);
}

async function loadOwnerSettings() {
  const loadingEl = document.getElementById('settings-loading');
  const formEl = document.getElementById('settings-form');
  try {
    const data = await api.get('/api/settings/school');
    const s = data.school || data;
    if (loadingEl) loadingEl.classList.add('hidden');
    if (formEl) formEl.classList.remove('hidden');
    setValue('s-name', s.name);
    setValue('s-motto', s.motto);
    setValue('s-address', s.address);
    setValue('s-phone', s.phone);
    setValue('s-email', s.email);
    setValue('s-prefix', s.prefix);
    setValue('s-logo-url', s.logo_url);
    if (s.established_year) setValue('s-established-year', s.established_year);
    if (s.primary_color) {
      document.getElementById('s-color').value = s.primary_color;
      document.getElementById('s-color-hex').value = s.primary_color;
    }
    if (s.logo_url) {
      const preview = document.getElementById('logo-preview');
      if (preview) preview.innerHTML = `<img src="${s.logo_url}" class="w-full h-full object-contain rounded-xl p-1" alt="Logo" />`;
    }
  } catch (err) {
    if (loadingEl) loadingEl.innerHTML = `<p class="text-danger text-sm">Failed to load settings: ${err.message}</p>`;
  }
}

async function saveOwnerSettings(e) {
  e.preventDefault();
  const errEl = document.getElementById('settings-error');
  errEl.classList.add('hidden');
  const establishedYear = parseInt(document.getElementById('s-established-year').value.trim()) || 0;
  const body = {
    name:          document.getElementById('s-name').value.trim(),
    motto:         document.getElementById('s-motto').value.trim(),
    address:       document.getElementById('s-address').value.trim(),
    phone:         document.getElementById('s-phone').value.trim(),
    email:         document.getElementById('s-email').value.trim(),
    prefix:        document.getElementById('s-prefix').value.trim().toUpperCase(),
    primary_color: document.getElementById('s-color-hex').value.trim(),
    logo_url:      document.getElementById('s-logo-url').value.trim(),
    ...(establishedYear > 1900 && { established_year: establishedYear }),
  };
  try {
    await api.put('/api/settings/school', body);
    showToast('Settings saved successfully.', 'success');
  } catch (err) {
    errEl.textContent = err.message || 'Failed to save settings.';
    errEl.classList.remove('hidden');
  }
}

// ─── ICT Admin: system-only settings ─────────────────────────────────────────
// Per RBAC spec: ICT_ADMIN can manage passwords, backups, email/SMS config,
// and system health — but CANNOT view or edit school identity or finances.

async function initICTSettings(container) {
  container.innerHTML = `
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-primary">System Settings</h1>
      <p class="text-text-secondary text-sm mt-1">Manage technical configuration, email/SMS delivery, and system health.</p>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">

      <!-- Email / SMS Configuration -->
      <div class="card p-6">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
          <h2 class="font-semibold text-primary">Email / SMS Configuration</h2>
        </div>
        <div class="space-y-3">
          <div><label class="label">SMTP Host</label><input id="ict-smtp-host" class="input w-full font-mono text-sm" placeholder="smtp.sendgrid.net" /></div>
          <div><label class="label">SMTP Port</label><input id="ict-smtp-port" type="number" class="input w-full font-mono text-sm" placeholder="587" /></div>
          <div><label class="label">From Email</label><input id="ict-smtp-from" type="email" class="input w-full font-mono text-sm" placeholder="noreply@leaps.edu.ng" /></div>
          <div><label class="label">SMS Provider API Key</label><input id="ict-sms-key" type="password" class="input w-full font-mono text-sm" placeholder="sk_live_…" /></div>
        </div>
        <div class="mt-4 flex gap-2">
          <button id="ict-test-email" class="btn btn-ghost text-sm">Send Test Email</button>
          <button id="ict-save-comms" class="btn btn-primary text-sm">Save</button>
        </div>
        <p id="comms-msg" class="text-sm mt-2 hidden"></p>
      </div>

      <!-- Backup & Storage -->
      <div class="card p-6">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2 1 3 3 3h10c2 0 3-1 3-3V9l-5-5H7c-2 0-3 1-3 3zm7 8v-6m0 0l-2 2m2-2l2 2"/></svg>
          <h2 class="font-semibold text-primary">Backup &amp; Storage</h2>
        </div>
        <div id="backup-status" class="space-y-2 mb-4">
          <div class="skeleton h-6 rounded"></div>
          <div class="skeleton h-6 rounded w-3/4"></div>
        </div>
        <div class="flex gap-2 flex-wrap">
          <button id="ict-trigger-backup" class="btn btn-primary text-sm">Trigger Backup Now</button>
          <button id="ict-restore" class="btn btn-ghost text-sm text-danger">Restore from Backup…</button>
        </div>
        <p id="backup-msg" class="text-sm mt-2 hidden"></p>
      </div>

      <!-- System Health -->
      <div class="card p-6 lg:col-span-2">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>
          <h2 class="font-semibold text-primary">System Health</h2>
        </div>
        <div id="health-grid" class="grid grid-cols-2 md:grid-cols-4 gap-4">
          ${['Database', 'Email Queue', 'Storage', 'Worker'].map(svc => `
            <div class="rounded-xl border border-neutral-100 p-4 text-center">
              <p class="text-xs text-text-secondary mb-1">${svc}</p>
              <div class="inline-flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-neutral-200 health-dot" data-service="${svc.toLowerCase().replace(' ','-')}"></span>
                <span class="text-sm font-medium health-label text-text-secondary">Checking…</span>
              </div>
            </div>`).join('')}
        </div>
      </div>

      <!-- Password Reset Tools -->
      <div class="card p-6 lg:col-span-2">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/></svg>
          <h2 class="font-semibold text-primary">Password Reset &amp; Account Unlock</h2>
        </div>
        <p class="text-sm text-text-secondary mb-4">Search for any user account to send a password reset email or unlock a locked account. You cannot view or change passwords directly.</p>
        <div class="flex gap-2 mb-4">
          <input id="ict-user-search" class="input flex-1" placeholder="Search by name or email…" />
          <button id="ict-user-search-btn" class="btn btn-primary">Search</button>
        </div>
        <div id="ict-user-results" class="space-y-2"></div>
        <p id="user-search-msg" class="text-sm mt-2 hidden"></p>
      </div>

    </div>
  `;

  // Load backup status and system health
  await Promise.all([loadBackupStatus(), loadHealthStatus()]);
  wireICTEvents();
}

async function loadBackupStatus() {
  const el = document.getElementById('backup-status');
  try {
    const data = await api.get('/api/admin/backup/status').catch(() => null);
    if (!data) {
      el.innerHTML = `<p class="text-sm text-text-secondary">Backup status unavailable.</p>`;
      return;
    }
    el.innerHTML = `
      <div class="flex justify-between text-sm">
        <span class="text-text-secondary">Last backup</span>
        <span class="font-medium text-primary">${data.last_backup_at ? new Date(data.last_backup_at).toLocaleString() : 'Never'}</span>
      </div>
      <div class="flex justify-between text-sm">
        <span class="text-text-secondary">Storage used</span>
        <span class="font-medium text-primary">${data.storage_used_mb ? data.storage_used_mb + ' MB' : '—'}</span>
      </div>
      <div class="flex justify-between text-sm">
        <span class="text-text-secondary">Next scheduled</span>
        <span class="font-medium text-primary">${data.next_backup_at ? new Date(data.next_backup_at).toLocaleString() : '—'}</span>
      </div>`;
  } catch {
    el.innerHTML = `<p class="text-sm text-text-secondary">Unable to load backup status.</p>`;
  }
}

async function loadHealthStatus() {
  const dots = document.querySelectorAll('.health-dot');
  const labels = document.querySelectorAll('.health-label');
  try {
    const data = await api.get('/api/admin/health').catch(() => null);
    const services = ['database', 'email-queue', 'storage', 'worker'];
    services.forEach((svc, i) => {
      const ok = data?.services?.[svc] !== false;
      if (dots[i]) {
        dots[i].className = `w-2 h-2 rounded-full ${ok ? 'bg-success' : 'bg-danger'} health-dot`;
      }
      if (labels[i]) {
        labels[i].textContent = ok ? 'OK' : 'Error';
        labels[i].className = `text-sm font-medium ${ok ? 'text-success' : 'text-danger'} health-label`;
      }
    });
  } catch {
    dots.forEach((d, i) => {
      d.className = 'w-2 h-2 rounded-full bg-neutral-300 health-dot';
      if (labels[i]) { labels[i].textContent = 'Unknown'; }
    });
  }
}

function wireICTEvents() {
  document.getElementById('ict-trigger-backup')?.addEventListener('click', async () => {
    const msg = document.getElementById('backup-msg');
    try {
      await api.post('/api/admin/backup/trigger', {});
      msg.textContent = 'Backup triggered successfully.';
      msg.className = 'text-sm mt-2 text-success';
      msg.classList.remove('hidden');
    } catch (err) {
      msg.textContent = err.message || 'Failed to trigger backup.';
      msg.className = 'text-sm mt-2 text-danger';
      msg.classList.remove('hidden');
    }
  });

  document.getElementById('ict-test-email')?.addEventListener('click', async () => {
    const msg = document.getElementById('comms-msg');
    try {
      await api.post('/api/admin/email/test', {});
      msg.textContent = 'Test email sent — check your inbox.';
      msg.className = 'text-sm mt-2 text-success';
      msg.classList.remove('hidden');
    } catch (err) {
      msg.textContent = err.message || 'Failed to send test email.';
      msg.className = 'text-sm mt-2 text-danger';
      msg.classList.remove('hidden');
    }
  });

  document.getElementById('ict-user-search-btn')?.addEventListener('click', async () => {
    const q = document.getElementById('ict-user-search')?.value.trim();
    const resultsEl = document.getElementById('ict-user-results');
    const msg = document.getElementById('user-search-msg');
    if (!q) return;
    resultsEl.innerHTML = '<div class="skeleton h-12 rounded-xl"></div>';
    msg.classList.add('hidden');
    try {
      const data = await api.get(`/api/users?search=${encodeURIComponent(q)}&limit=10`);
      const users = data.users || data || [];
      if (!users.length) {
        resultsEl.innerHTML = '<p class="text-sm text-text-secondary">No users found.</p>';
        return;
      }
      resultsEl.innerHTML = users.map(u => `
        <div class="card p-4 flex items-center justify-between gap-4">
          <div>
            <p class="font-medium text-sm text-primary">${u.full_name}</p>
            <p class="text-xs text-text-secondary">${u.email} · ${u.role.replace(/_/g,' ')}</p>
          </div>
          <div class="flex gap-2">
            <button class="btn btn-ghost text-sm reset-pw-btn" data-id="${u.id}" data-email="${u.email}">Reset Password</button>
            ${u.is_locked ? `<button class="btn btn-ghost text-sm text-success unlock-btn" data-id="${u.id}">Unlock</button>` : ''}
          </div>
        </div>`).join('');

      resultsEl.querySelectorAll('.reset-pw-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
          try {
            await api.post(`/api/users/${btn.dataset.id}/reset-password`, {});
            showToast(`Password reset email sent to ${btn.dataset.email}.`, 'success');
          } catch (err) {
            showToast(err.message || 'Failed to send reset email.', 'error');
          }
        });
      });

      resultsEl.querySelectorAll('.unlock-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
          try {
            await api.post(`/api/users/${btn.dataset.id}/unlock`, {});
            showToast('Account unlocked.', 'success');
            btn.remove();
          } catch (err) {
            showToast(err.message || 'Failed to unlock account.', 'error');
          }
        });
      });
    } catch (err) {
      resultsEl.innerHTML = '';
      msg.textContent = err.message || 'Search failed.';
      msg.className = 'text-sm mt-2 text-danger';
      msg.classList.remove('hidden');
    }
  });
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

function setValue(id, val) {
  const el = document.getElementById(id);
  if (el) el.value = val || '';
}
