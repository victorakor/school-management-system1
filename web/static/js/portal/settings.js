/**
 * settings.js — School settings module (Owner only)
 */
import { api } from '../shared/api.js';
import { showToast } from '../shared/utils.js';

export async function initSettings(container, user) {
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
            <p class="text-xs text-text-secondary mt-1">Paste a Cloudinary URL after uploading via the upload button in your profile.</p>
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

  await loadSettings();

  document.getElementById('s-color').addEventListener('input', e => {
    document.getElementById('s-color-hex').value = e.target.value;
  });
  document.getElementById('s-color-hex').addEventListener('input', e => {
    if (/^#[0-9a-fA-F]{6}$/.test(e.target.value)) {
      document.getElementById('s-color').value = e.target.value;
    }
  });
  document.getElementById('settings-form').addEventListener('submit', saveSettings);
  document.getElementById('cancel-settings').addEventListener('click', loadSettings);
}

async function loadSettings() {
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

async function saveSettings(e) {
  e.preventDefault();
  const errEl = document.getElementById('settings-error');
  errEl.classList.add('hidden');

  const body = {
    name:          document.getElementById('s-name').value.trim(),
    motto:         document.getElementById('s-motto').value.trim(),
    address:       document.getElementById('s-address').value.trim(),
    phone:         document.getElementById('s-phone').value.trim(),
    email:         document.getElementById('s-email').value.trim(),
    prefix:        document.getElementById('s-prefix').value.trim().toUpperCase(),
    primary_color: document.getElementById('s-color-hex').value.trim(),
    logo_url:      document.getElementById('s-logo-url').value.trim(),
  };

  try {
    await api.put('/api/settings/school', body);
    showToast('Settings saved successfully.', 'success');
  } catch (err) {
    errEl.textContent = err.message || 'Failed to save settings.';
    errEl.classList.remove('hidden');
  }
}

function setValue(id, val) {
  const el = document.getElementById(id);
  if (el) el.value = val || '';
}
