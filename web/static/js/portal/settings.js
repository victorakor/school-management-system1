/**
 * settings.js — School settings module
 * REPLACE: web/static/js/portal/settings.js
 *
 * Changes from original:
 *   - Added "Academic Calendar" section for Owner:
 *       • Open / Close individual terms
 *       • Close a session
 *       • Publish the academic calendar
 */
import { api } from '../shared/api.js';
import { formatDate, showToast } from '../shared/utils.js';

export async function initSettings(container, user) {
  const role = user?.role || window.PORTAL_ROLE || '';
  if (role === 'ICT_ADMIN') return initICTSettings(container);
  if (role === 'OWNER')     return initOwnerSettings(container);

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

// ─── Owner Settings ───────────────────────────────────────────────────────────

async function initOwnerSettings(container) {
  container.innerHTML = `
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-primary">School Settings</h1>
      <p class="text-text-secondary text-sm mt-1">Manage school identity, branding, academic calendar, and configuration.</p>
    </div>

    <!-- Sub-tabs -->
    <div class="flex gap-1 mb-6 border-b border-neutral-100">
      <button class="settings-tab px-4 py-2 text-sm font-medium border-b-2 transition-colors" data-tab="identity">School Identity</button>
      <button class="settings-tab px-4 py-2 text-sm font-medium border-b-2 transition-colors" data-tab="calendar">Academic Calendar</button>
    </div>

    <div id="settings-tab-content"></div>
  `;

  container.querySelectorAll('.settings-tab').forEach(btn => {
    btn.addEventListener('click', () => switchSettingsTab(btn.dataset.tab));
  });

  switchSettingsTab('identity');
}

function switchSettingsTab(tab) {
  document.querySelectorAll('.settings-tab').forEach(b => {
    const active = b.dataset.tab === tab;
    b.classList.toggle('border-primary', active);
    b.classList.toggle('text-primary', active);
    b.classList.toggle('border-transparent', !active);
    b.classList.toggle('text-text-secondary', !active);
  });
  const content = document.getElementById('settings-tab-content');
  if (!content) return;
  if (tab === 'identity') renderIdentityTab(content);
  if (tab === 'calendar') renderCalendarTab(content);
}

// ─── Identity Tab (original school settings form) ────────────────────────────

function renderIdentityTab(content) {
  content.innerHTML = `
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
            <div id="logo-preview" class="w-20 h-20 rounded-xl border-2 border-dashed border-neutral-200 flex items-center justify-center mb-2">
              <svg class="w-8 h-8 text-neutral-300" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
            </div>
            <input id="s-logo-url" class="input w-full text-sm font-mono" placeholder="https://res.cloudinary.com/…" />
          </div>
          <div>
            <label class="label">School Stamp / Seal</label>
            <div id="stamp-preview" class="w-20 h-20 rounded-xl border-2 border-dashed border-neutral-200 flex items-center justify-center mb-2">
              <svg class="w-8 h-8 text-neutral-300" fill="none" stroke="currentColor" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/></svg>
            </div>
            <input id="s-stamp-url" class="input w-full text-sm font-mono" placeholder="https://res.cloudinary.com/…" />
          </div>
          <div>
            <label class="label">Principal's Signature</label>
            <input id="s-sig-principal" class="input w-full text-sm font-mono" placeholder="https://res.cloudinary.com/…" />
          </div>
          <div>
            <label class="label">Head Teacher's Signature</label>
            <input id="s-sig-headteacher" class="input w-full text-sm font-mono" placeholder="https://res.cloudinary.com/…" />
          </div>
        </div>
      </div>

      <div class="card p-6">
        <h2 class="font-semibold text-primary mb-4">Document & Finance Settings</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div><label class="label">Bursar Name (for receipts)</label><input id="s-bursar-name" class="input w-full" placeholder="e.g. Mrs. Adaeze Obi" /></div>
          <div><label class="label">Max Video Upload (MB)</label><input id="s-max-video" type="number" class="input w-full" min="10" max="500" placeholder="50" /></div>
          <div class="md:col-span-2"><label class="label">Admission Documents Required (one per line)</label>
            <textarea id="s-admission-docs" class="input w-full" rows="3" placeholder="Birth certificate&#10;Passport photograph&#10;Previous school report"></textarea>
          </div>
          <div class="md:col-span-2"><label class="label">School Directions / How to Find Us</label>
            <textarea id="s-directions" class="input w-full" rows="3"></textarea>
          </div>
          <div class="md:col-span-2 flex items-center gap-3 p-3 rounded-xl bg-neutral-50">
            <input id="s-watermark" type="checkbox" class="w-4 h-4 rounded accent-primary" />
            <div>
              <label for="s-watermark" class="text-sm font-medium text-primary cursor-pointer">Enable watermark on result cards</label>
              <p class="text-xs text-text-secondary">Adds a faint school logo watermark to PDF result cards.</p>
            </div>
          </div>
        </div>
      </div>

      <div class="flex justify-end gap-3">
        <button type="button" id="cancel-settings" class="btn btn-ghost">Discard</button>
        <button type="submit" class="btn btn-primary">Save Settings</button>
      </div>
      <p id="settings-error" class="text-danger text-sm hidden"></p>
    </form>`;

  loadOwnerSettings();

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
  const formEl    = document.getElementById('settings-form');
  try {
    const data = await api.get('/api/settings/school');
    const s    = data.school || data;
    loadingEl?.classList.add('hidden');
    formEl?.classList.remove('hidden');
    setValue('s-name', s.name); setValue('s-motto', s.motto);
    setValue('s-address', s.address); setValue('s-phone', s.phone);
    setValue('s-email', s.email); setValue('s-prefix', s.prefix);
    setValue('s-logo-url', s.logo_url); setValue('s-stamp-url', s.stamp_url);
    setValue('s-sig-principal', s.signature_principal_url);
    setValue('s-sig-headteacher', s.signature_headteacher_url);
    setValue('s-bursar-name', s.bursar_name);
    setValue('s-admission-docs', s.admission_documents_list);
    setValue('s-directions', s.school_directions);
    if (s.max_video_upload_mb) setValue('s-max-video', s.max_video_upload_mb);
    const watermarkEl = document.getElementById('s-watermark');
    if (watermarkEl) watermarkEl.checked = !!s.watermark_enabled;
    if (s.established_year) setValue('s-established-year', s.established_year);
    if (s.primary_color) {
      document.getElementById('s-color').value = s.primary_color;
      document.getElementById('s-color-hex').value = s.primary_color;
    }
    if (s.logo_url) {
      const preview = document.getElementById('logo-preview');
      if (preview) preview.innerHTML = `<img src="${s.logo_url}" class="w-full h-full object-contain rounded-xl p-1" alt="Logo" />`;
    }
    if (s.stamp_url) {
      const preview = document.getElementById('stamp-preview');
      if (preview) preview.innerHTML = `<img src="${s.stamp_url}" class="w-full h-full object-contain rounded-xl p-1" alt="Stamp" />`;
    }
  } catch (err) {
    loadingEl?.innerHTML = `<p class="text-danger text-sm">Failed to load settings: ${err.message}</p>`;
  }
}

async function saveOwnerSettings(e) {
  e.preventDefault();
  const errEl = document.getElementById('settings-error');
  errEl.classList.add('hidden');
  const establishedYear = parseInt(document.getElementById('s-established-year').value.trim()) || 0;
  const maxVideo        = parseInt(document.getElementById('s-max-video')?.value.trim()) || 0;
  const body = {
    name: document.getElementById('s-name').value.trim(),
    motto: document.getElementById('s-motto').value.trim(),
    address: document.getElementById('s-address').value.trim(),
    phone: document.getElementById('s-phone').value.trim(),
    email: document.getElementById('s-email').value.trim(),
    prefix: document.getElementById('s-prefix').value.trim().toUpperCase(),
    primary_color: document.getElementById('s-color-hex').value.trim(),
    logo_url: document.getElementById('s-logo-url').value.trim(),
    stamp_url: document.getElementById('s-stamp-url')?.value.trim() || '',
    signature_principal_url: document.getElementById('s-sig-principal')?.value.trim() || '',
    signature_headteacher_url: document.getElementById('s-sig-headteacher')?.value.trim() || '',
    bursar_name: document.getElementById('s-bursar-name')?.value.trim() || '',
    admission_documents_list: document.getElementById('s-admission-docs')?.value.trim() || '',
    school_directions: document.getElementById('s-directions')?.value.trim() || '',
    watermark_enabled: document.getElementById('s-watermark')?.checked ?? false,
    ...(establishedYear > 1900 && { established_year: establishedYear }),
    ...(maxVideo > 0 && { max_video_upload_mb: maxVideo }),
  };
  try {
    await api.put('/api/settings/school', body);
    showToast('Settings saved successfully.', 'success');
  } catch (err) {
    errEl.textContent = err.message || 'Failed to save settings.';
    errEl.classList.remove('hidden');
  }
}

// ─── Academic Calendar Tab ────────────────────────────────────────────────────

async function renderCalendarTab(content) {
  content.innerHTML = `
    <div class="mb-4 flex items-center justify-between">
      <div>
        <h2 class="font-semibold text-primary">Academic Calendar</h2>
        <p class="text-text-secondary text-sm mt-0.5">Open and close terms, close sessions, and publish the calendar.</p>
      </div>
      <select id="cal-session-sel" class="input text-sm py-1.5 w-52">
        <option value="">Select Session</option>
      </select>
    </div>
    <div id="calendar-content">
      <div class="card p-4 space-y-2">${Array(4).fill('<div class="skeleton h-14 rounded-lg"></div>').join('')}</div>
    </div>`;

  try {
    const sessions = await api.get('/api/sessions');
    const sel = document.getElementById('cal-session-sel');
    sessions.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.id; opt.textContent = s.name;
      if (s.is_active) opt.selected = true;
      sel.appendChild(opt);
    });
    sel.addEventListener('change', () => loadCalendarSession(sel.value));
    if (sel.value) loadCalendarSession(sel.value);
    else document.getElementById('calendar-content').innerHTML =
      `<div class="card p-8 text-center text-text-secondary">Select a session above to manage its terms.</div>`;
  } catch (err) {
    document.getElementById('calendar-content').innerHTML =
      `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
  }
}

async function loadCalendarSession(sessionId) {
  if (!sessionId) return;
  const content = document.getElementById('calendar-content');
  content.innerHTML = `<div class="card p-4 space-y-2">${Array(3).fill('<div class="skeleton h-14 rounded-lg"></div>').join('')}</div>`;

  try {
    const [sessionRes, terms] = await Promise.all([
      api.get('/api/sessions').then(list => list.find(s => s.id === sessionId)),
      api.get(`/api/sessions/${sessionId}/terms`),
    ]);
    const session = sessionRes || {};

    content.innerHTML = `
      <!-- Session actions -->
      <div class="card p-5 mb-4">
        <div class="flex items-center justify-between flex-wrap gap-3">
          <div>
            <h3 class="font-semibold text-primary">${session.name || 'Session'}</h3>
            <p class="text-xs text-text-secondary mt-0.5">
              Status: <span class="font-medium ${session.is_active ? 'text-success' : 'text-text-secondary'}">${session.is_active ? 'Active' : 'Closed'}</span>
            </p>
          </div>
          <div class="flex gap-2 flex-wrap">
            <button id="publish-cal-btn" class="btn btn-primary btn-sm">
              Publish Calendar
            </button>
            ${session.is_active ? `<button id="close-session-btn" class="btn btn-danger-ghost btn-sm">Close Session</button>` : ''}
          </div>
        </div>
      </div>

      <!-- Terms -->
      <div class="space-y-3" id="terms-list">
        ${terms.length ? terms.map(t => renderTermCard(t, sessionId)).join('') :
          `<div class="card p-8 text-center text-text-secondary">No terms found for this session.</div>`}
      </div>`;

    document.getElementById('publish-cal-btn')?.addEventListener('click', () => publishCalendar(sessionId, session.name));
    document.getElementById('close-session-btn')?.addEventListener('click', () => closeSession(sessionId, session.name));

    wireTermButtons(sessionId);
  } catch (err) {
    content.innerHTML = `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
  }
}

function renderTermCard(term, sessionId) {
  return `
    <div class="card p-4 flex items-center justify-between gap-4 flex-wrap" id="term-card-${term.id}">
      <div>
        <p class="font-medium text-primary">${term.name}</p>
        <p class="text-xs text-text-secondary mt-0.5">
          ${term.start_date ? formatDate(new Date(term.start_date)) : '—'} →
          ${term.end_date   ? formatDate(new Date(term.end_date))   : '—'}
          ${term.next_resumption_date ? ` · Resumption: ${formatDate(new Date(term.next_resumption_date))}` : ''}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <span class="badge ${term.is_active ? 'badge-success' : 'badge-neutral'} text-xs">
          ${term.is_active ? 'Open' : 'Closed'}
        </span>
        <button class="btn btn-sm ${term.is_active ? 'btn-danger-ghost' : 'btn-ghost'} toggle-term-btn"
          data-term-id="${term.id}"
          data-session-id="${sessionId}"
          data-is-active="${term.is_active}"
          data-name="${term.name}">
          ${term.is_active ? 'Close Term' : 'Open Term'}
        </button>
      </div>
    </div>`;
}

function wireTermButtons(sessionId) {
  document.querySelectorAll('.toggle-term-btn').forEach(btn => {
    btn.addEventListener('click', () => toggleTerm(
      btn.dataset.termId,
      btn.dataset.sessionId,
      btn.dataset.isActive === 'true',
      btn.dataset.name,
    ));
  });
}

async function toggleTerm(termId, sessionId, isCurrentlyActive, termName) {
  const action = isCurrentlyActive ? 'close' : 'open';
  if (!confirm(`${isCurrentlyActive ? 'Close' : 'Open'} "${termName}"? ${isCurrentlyActive ? 'This will lock score entry for this term.' : ''}`)) return;
  try {
    await api.put(`/api/sessions/${sessionId}/terms/${termId}/status`, { is_active: !isCurrentlyActive });
    showToast(`"${termName}" ${action}d successfully.`, 'success');
    loadCalendarSession(sessionId);
  } catch (err) {
    showToast(err.message || `Failed to ${action} term.`, 'error');
  }
}

async function closeSession(sessionId, sessionName) {
  if (!confirm(`Close session "${sessionName}"? This will not activate another session automatically.`)) return;
  try {
    await api.put(`/api/sessions/${sessionId}/close`, {});
    showToast(`Session "${sessionName}" closed.`, 'success');
    loadCalendarSession(sessionId);
  } catch (err) {
    showToast(err.message || 'Failed to close session.', 'error');
  }
}

async function publishCalendar(sessionId, sessionName) {
  if (!confirm(`Publish the academic calendar for "${sessionName}"? This will be visible to all staff.`)) return;
  try {
    await api.post(`/api/sessions/${sessionId}/calendar/publish`, {});
    showToast('Academic calendar published.', 'success');
  } catch (err) {
    showToast(err.message || 'Failed to publish calendar.', 'error');
  }
}

// ─── ICT Admin Settings (unchanged from original) ────────────────────────────

async function initICTSettings(container) {
  container.innerHTML = `
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-primary">System Settings</h1>
      <p class="text-text-secondary text-sm mt-1">Manage technical configuration, email/SMS delivery, and system health.</p>
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
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
      <div class="card p-6">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2 1 3 3 3h10c2 0 3-1 3-3V9l-5-5H7c-2 0-3 1-3 3z"/></svg>
          <h2 class="font-semibold text-primary">Backup &amp; Storage</h2>
        </div>
        <div id="backup-status" class="space-y-2 mb-4">
          <div class="skeleton h-6 rounded"></div><div class="skeleton h-6 rounded w-3/4"></div>
        </div>
        <div class="flex gap-2 flex-wrap">
          <button id="ict-trigger-backup" class="btn btn-primary text-sm">Trigger Backup Now</button>
          <button id="ict-restore" class="btn btn-ghost text-sm text-danger">Restore from Backup…</button>
        </div>
        <p id="backup-msg" class="text-sm mt-2 hidden"></p>
      </div>
      <div class="card p-6 lg:col-span-2">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>
          <h2 class="font-semibold text-primary">System Health</h2>
        </div>
        <div id="health-grid" class="grid grid-cols-2 md:grid-cols-4 gap-4">
          ${['Database','Email Queue','Storage','Worker'].map(svc => `
            <div class="rounded-xl border border-neutral-100 p-4 text-center">
              <p class="text-xs text-text-secondary mb-1">${svc}</p>
              <div class="inline-flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-neutral-200 health-dot" data-service="${svc.toLowerCase().replace(' ','-')}"></span>
                <span class="text-sm font-medium health-label text-text-secondary">Checking…</span>
              </div>
            </div>`).join('')}
        </div>
      </div>
      <div class="card p-6 lg:col-span-2">
        <div class="flex items-center gap-2 mb-4">
          <svg class="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/></svg>
          <h2 class="font-semibold text-primary">Password Reset &amp; Account Unlock</h2>
        </div>
        <p class="text-sm text-text-secondary mb-4">Search for any user to send a password reset email or unlock a locked account.</p>
        <div class="flex gap-2 mb-4">
          <input id="ict-user-search" class="input flex-1" placeholder="Search by name or email…" />
          <button id="ict-user-search-btn" class="btn btn-primary">Search</button>
        </div>
        <div id="ict-user-results" class="space-y-2"></div>
        <p id="user-search-msg" class="text-sm mt-2 hidden"></p>
      </div>
    </div>`;

  await Promise.all([loadBackupStatus(), loadHealthStatus()]);
  wireICTEvents();
}

async function loadBackupStatus() {
  const el = document.getElementById('backup-status');
  try {
    const data = await api.get('/api/admin/backup/status').catch(() => null);
    if (!data) { el.innerHTML = `<p class="text-sm text-text-secondary">Backup status unavailable.</p>`; return; }
    el.innerHTML = `
      <div class="flex justify-between text-sm"><span class="text-text-secondary">Last backup</span><span class="font-medium text-primary">${data.last_backup_at ? new Date(data.last_backup_at).toLocaleString() : 'Never'}</span></div>
      <div class="flex justify-between text-sm"><span class="text-text-secondary">Storage used</span><span class="font-medium text-primary">${data.storage_used_mb ? data.storage_used_mb + ' MB' : '—'}</span></div>
      <div class="flex justify-between text-sm"><span class="text-text-secondary">Next scheduled</span><span class="font-medium text-primary">${data.next_backup_at ? new Date(data.next_backup_at).toLocaleString() : '—'}</span></div>`;
  } catch { el.innerHTML = `<p class="text-sm text-text-secondary">Unable to load backup status.</p>`; }
}

async function loadHealthStatus() {
  const dots = document.querySelectorAll('.health-dot');
  const labels = document.querySelectorAll('.health-label');
  try {
    const data = await api.get('/api/admin/health').catch(() => null);
    ['database','email-queue','storage','worker'].forEach((svc, i) => {
      const ok = data?.services?.[svc] !== false;
      if (dots[i]) dots[i].className = `w-2 h-2 rounded-full ${ok ? 'bg-success' : 'bg-danger'} health-dot`;
      if (labels[i]) { labels[i].textContent = ok ? 'OK' : 'Error'; labels[i].className = `text-sm font-medium ${ok ? 'text-success' : 'text-danger'} health-label`; }
    });
  } catch {
    dots.forEach((d, i) => { d.className = 'w-2 h-2 rounded-full bg-neutral-300 health-dot'; if (labels[i]) labels[i].textContent = 'Unknown'; });
  }
}

function wireICTEvents() {
  document.getElementById('ict-trigger-backup')?.addEventListener('click', async () => {
    const msg = document.getElementById('backup-msg');
    try {
      await api.post('/api/admin/backup/trigger', {});
      msg.textContent = 'Backup triggered successfully.'; msg.className = 'text-sm mt-2 text-success'; msg.classList.remove('hidden');
    } catch (err) { msg.textContent = err.message || 'Failed.'; msg.className = 'text-sm mt-2 text-danger'; msg.classList.remove('hidden'); }
  });
  document.getElementById('ict-test-email')?.addEventListener('click', async () => {
    const msg = document.getElementById('comms-msg');
    try {
      await api.post('/api/admin/email/test', {});
      msg.textContent = 'Test email sent.'; msg.className = 'text-sm mt-2 text-success'; msg.classList.remove('hidden');
    } catch (err) { msg.textContent = err.message || 'Failed.'; msg.className = 'text-sm mt-2 text-danger'; msg.classList.remove('hidden'); }
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
      if (!users.length) { resultsEl.innerHTML = '<p class="text-sm text-text-secondary">No users found.</p>'; return; }
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
          try { await api.post(`/api/users/${btn.dataset.id}/reset-password`, {}); showToast(`Password reset email sent to ${btn.dataset.email}.`, 'success'); }
          catch (err) { showToast(err.message || 'Failed.', 'error'); }
        });
      });
      resultsEl.querySelectorAll('.unlock-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
          try { await api.post(`/api/users/${btn.dataset.id}/unlock`, {}); showToast('Account unlocked.', 'success'); btn.remove(); }
          catch (err) { showToast(err.message || 'Failed.', 'error'); }
        });
      });
    } catch (err) {
      resultsEl.innerHTML = '';
      msg.textContent = err.message || 'Search failed.'; msg.className = 'text-sm mt-2 text-danger'; msg.classList.remove('hidden');
    }
  });
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function setValue(id, val) {
  const el = document.getElementById(id);
  if (el) el.value = val || '';
}
