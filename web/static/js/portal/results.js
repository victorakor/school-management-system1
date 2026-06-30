/**
 * results.js — Results viewer and publication interface
 * REPLACE: web/static/js/portal/results.js
 *
 * Changes from original:
 *   - Owner gets an "Override Result" button per result row that opens a modal
 *     to edit class teacher remark, admin remark, and mandatory override reason
 */
import { api } from '../shared/api.js';
import { formatDate, showToast } from '../shared/utils.js';

export async function initResults(container, user) {
  const isOwner = user?.role === 'OWNER';

  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Results</h1>
      ${canPublish(user.role) ? `<button id="publish-btn" class="btn btn-primary hidden">Publish Results</button>` : ''}
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
      <select id="res-session" class="input"><option value="">Select Session</option></select>
      <select id="res-term" class="input"><option value="">Select Term</option></select>
      <select id="res-class" class="input"><option value="">Select Class</option></select>
    </div>
    <div id="results-container"></div>
  `;

  await loadResultFilters();
  document.getElementById('res-session').addEventListener('change', onSessionChange);
  document.getElementById('res-term').addEventListener('change', loadResults);
  document.getElementById('res-class').addEventListener('change', loadResults);

  if (isOwner) mountOverrideModal();
}

async function loadResultFilters() {
  try {
    const [sessions, classes] = await Promise.all([
      api.get('/api/sessions'),
      api.get('/api/classes'),
    ]);
    const sessionSel = document.getElementById('res-session');
    sessions.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.id; opt.textContent = s.name;
      if (s.is_active) opt.selected = true;
      sessionSel.appendChild(opt);
    });
    populateSelect('res-class', classes, 'name');
    if (sessionSel.value) await onSessionChange();
  } catch {}
}

async function onSessionChange() {
  const sessionID = document.getElementById('res-session').value;
  if (!sessionID) return;
  try {
    const terms = await api.get(`/api/sessions/${sessionID}/terms`);
    populateSelect('res-term', terms, 'name');
  } catch {}
}

function populateSelect(id, items, labelKey) {
  const sel = document.getElementById(id);
  const current = sel.value;
  sel.innerHTML = `<option value="">Select ${id.replace('res-','')}</option>`;
  items.forEach(item => {
    const opt = document.createElement('option');
    opt.value = item.id; opt.textContent = item[labelKey];
    if (item.id === current) opt.selected = true;
    sel.appendChild(opt);
  });
}

async function loadResults() {
  const sessionID = document.getElementById('res-session').value;
  const termID    = document.getElementById('res-term').value;
  const classID   = document.getElementById('res-class').value;
  const container = document.getElementById('results-container');
  if (!sessionID || !termID || !classID) { container.innerHTML = ''; return; }

  container.innerHTML = `<div class="card p-6"><div class="skeleton h-64 rounded"></div></div>`;

  const isOwner = window.PORTAL_ROLE === 'OWNER';

  try {
    const data = await api.get(`/api/results?session_id=${sessionID}&term_id=${termID}&class_id=${classID}`);
    const results = data.results || data || [];

    if (!results.length) {
      container.innerHTML = `<div class="card p-12 text-center text-text-secondary">No results found for this selection.</div>`;
      return;
    }

    const publishBtn = document.getElementById('publish-btn');
    if (publishBtn) publishBtn.classList.remove('hidden');

    container.innerHTML = `
      <div class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="data-table w-full text-sm">
            <thead>
              <tr>
                <th>Student</th>
                <th class="text-right">Total</th>
                <th class="text-right">Average</th>
                <th>Grade</th>
                <th>Remark</th>
                <th>Status</th>
                ${isOwner ? '<th></th>' : ''}
              </tr>
            </thead>
            <tbody>
              ${results.map(r => `
                <tr>
                  <td class="font-medium text-primary">${r.student_name || r.student?.full_name || '—'}</td>
                  <td class="text-right">${r.total_score ?? '—'}</td>
                  <td class="text-right">${r.average != null ? Number(r.average).toFixed(1) : '—'}</td>
                  <td><span class="badge badge-neutral text-xs">${r.grade || '—'}</span></td>
                  <td class="text-text-secondary text-xs max-w-[180px] truncate">${r.class_teacher_remark || r.admin_remark || '—'}</td>
                  <td><span class="badge ${r.is_published ? 'badge-success' : 'badge-warning'} text-xs">${r.is_published ? 'Published' : 'Draft'}</span></td>
                  ${isOwner ? `
                  <td>
                    <button class="btn btn-sm btn-ghost override-result-btn"
                      data-id="${r.id}"
                      data-student="${r.student_name || r.student?.full_name || ''}"
                      data-teacher-remark="${encodeURIComponent(r.class_teacher_remark || '')}"
                      data-admin-remark="${encodeURIComponent(r.admin_remark || '')}">
                      Override
                    </button>
                  </td>` : ''}
                </tr>`).join('')}
            </tbody>
          </table>
        </div>
      </div>`;

    if (isOwner) {
      container.querySelectorAll('.override-result-btn').forEach(btn => {
        btn.addEventListener('click', () => openOverrideModal(
          btn.dataset.id,
          btn.dataset.student,
          decodeURIComponent(btn.dataset.teacherRemark),
          decodeURIComponent(btn.dataset.adminRemark),
        ));
      });
    }

    if (publishBtn) {
      publishBtn.onclick = async () => {
        try {
          await api.post(`/api/results/publish`, { session_id: sessionID, term_id: termID, class_id: classID });
          showToast('Results published.', 'success');
          loadResults();
        } catch (err) {
          showToast(err.message || 'Failed to publish results.', 'error');
        }
      };
    }
  } catch (err) {
    container.innerHTML = `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
  }
}

// ─── Override Result Modal ────────────────────────────────────────────────────

function mountOverrideModal() {
  if (document.getElementById('override-result-modal')) return;
  const modal = document.createElement('div');
  modal.id = 'override-result-modal';
  modal.style.cssText = 'display:none;position:fixed;inset:0;z-index:200;align-items:center;justify-content:center;';
  modal.innerHTML = `
    <div id="override-backdrop" style="position:absolute;inset:0;background:rgba(0,0,0,0.45);backdrop-filter:blur(4px);"></div>
    <div style="position:relative;background:#fff;border-radius:1.25rem;box-shadow:0 25px 60px rgba(0,0,0,0.2);width:100%;max-width:480px;padding:1.5rem;z-index:201;margin:1rem;">
      <h2 style="font-size:1.125rem;font-weight:700;color:var(--color-primary);margin-bottom:0.25rem;">Override Result</h2>
      <p id="override-subtitle" style="font-size:0.8rem;color:var(--color-text-secondary);margin-bottom:1rem;"></p>
      <div class="p-3 rounded-xl mb-4" style="background:#fef9c3;border:1px solid #fde68a;">
        <p style="font-size:0.78rem;color:#92400e;">⚠️ This action is permanently audit-logged and cannot be undone. Only use when the recorded remark is factually incorrect.</p>
      </div>
      <div style="display:grid;gap:0.75rem;">
        <div>
          <label class="label">Class Teacher Remark</label>
          <input id="override-teacher-remark" class="input w-full" placeholder="Leave blank to keep existing" />
        </div>
        <div>
          <label class="label">Admin / Head Remark</label>
          <input id="override-admin-remark" class="input w-full" placeholder="Leave blank to keep existing" />
        </div>
        <div>
          <label class="label">Override Reason <span style="color:var(--color-danger)">*</span></label>
          <input id="override-reason" class="input w-full" placeholder="Required — explain why this is being overridden" />
        </div>
      </div>
      <p id="override-error" style="color:var(--color-danger);font-size:0.75rem;margin-top:0.5rem;display:none;"></p>
      <div style="display:flex;gap:0.75rem;margin-top:1.25rem;">
        <button id="cancel-override-btn" class="btn btn-ghost" style="flex:1;">Cancel</button>
        <button id="confirm-override-btn" class="btn btn-danger" style="flex:1;">Confirm Override</button>
      </div>
    </div>`;
  document.body.appendChild(modal);
  modal.querySelector('#override-backdrop').addEventListener('click', () => { modal.style.display = 'none'; });
  modal.querySelector('#cancel-override-btn').addEventListener('click', () => { modal.style.display = 'none'; });
}

function openOverrideModal(resultId, studentName, teacherRemark, adminRemark) {
  const modal = document.getElementById('override-result-modal');
  if (!modal) return;
  document.getElementById('override-subtitle').textContent = `Editing result for: ${studentName}`;
  document.getElementById('override-teacher-remark').value = teacherRemark;
  document.getElementById('override-admin-remark').value = adminRemark;
  document.getElementById('override-reason').value = '';
  document.getElementById('override-error').style.display = 'none';
  modal.style.display = 'flex';

  const oldBtn = document.getElementById('confirm-override-btn');
  const newBtn = oldBtn.cloneNode(true);
  oldBtn.parentNode.replaceChild(newBtn, oldBtn);
  newBtn.addEventListener('click', async () => {
    const reason = document.getElementById('override-reason').value.trim();
    const errEl  = document.getElementById('override-error');
    errEl.style.display = 'none';
    if (!reason) {
      errEl.textContent = 'Override reason is required.';
      errEl.style.display = 'block';
      return;
    }
    try {
      await api.put(`/api/results/${resultId}/override`, {
        class_teacher_remark: document.getElementById('override-teacher-remark').value.trim(),
        admin_remark:         document.getElementById('override-admin-remark').value.trim(),
        reason,
      });
      modal.style.display = 'none';
      showToast('Result overridden and logged.', 'success');
      loadResults();
    } catch (err) {
      errEl.textContent = err.message || 'Failed to override result.';
      errEl.style.display = 'block';
    }
  });
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function canPublish(role) {
  return ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','EXAM_OFFICER'].includes(role);
}
