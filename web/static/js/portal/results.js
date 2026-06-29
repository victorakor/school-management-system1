/**
 * results.js — Results viewer and publication interface
 */
import { api } from '../shared/api.js';
import { formatDate, showToast } from '../shared/utils.js';

export async function initResults(container, user) {
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
      opt.value = s.id;
      opt.textContent = s.name;
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
  sel.innerHTML = `<option value="">Select ${id.replace('res-', '')}</option>`;
  items.forEach(item => {
    const opt = document.createElement('option');
    opt.value = item.id;
    opt.textContent = item[labelKey];
    if (item.id === current) opt.selected = true;
    sel.appendChild(opt);
  });
}

async function loadResults() {
  const sessionID = document.getElementById('res-session').value;
  const termID = document.getElementById('res-term').value;
  const classID = document.getElementById('res-class').value;
  const container = document.getElementById('results-container');

  if (!sessionID || !termID || !classID) {
    container.innerHTML = '';
    return;
  }

  container.innerHTML = `<div class="card p-6"><div class="skeleton h-64 rounded"></div></div>`;

  try {
    const results = await api.get(`/api/results?session_id=${sessionID}&term_id=${termID}&class_id=${classID}`);

    if (!results.length) {
      container.innerHTML = `<div class="card p-12 text-center text-secondary">No results found. Ensure all scores are approved and calculation has been triggered.</div>`;
      return;
    }

    const allPublished = results.every(r => r.is_published);
    const publishBtn = document.getElementById('publish-btn');
    if (publishBtn) {
      publishBtn.classList.toggle('hidden', allPublished);
      publishBtn.onclick = () => publishResults(sessionID, termID, classID);
    }

    container.innerHTML = `
      <div class="card overflow-hidden">
        <div class="p-4 border-b border-neutral-100 flex items-center justify-between">
          <h3 class="font-semibold text-primary">Class Results — ${results.length} students</h3>
          <div class="flex gap-2">
            <button class="btn btn-outline btn-sm" onclick="window._calculateResults('${sessionID}','${termID}','${classID}')">
              Recalculate
            </button>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="data-table w-full">
            <thead>
              <tr>
                <th class="text-left">Student</th>
                <th>Total</th>
                <th>Average</th>
                <th>Position</th>
                <th>Subjects Passed</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              ${results.map(r => `
                <tr>
                  <td class="font-medium">${r.student_name || r.student_id}</td>
                  <td>${r.total_marks}</td>
                  <td>${r.average?.toFixed(1)}%</td>
                  <td>${r.class_position ? ordinal(r.class_position) : '--'}</td>
                  <td>${r.subjects_passed}/${r.subjects_passed + r.subjects_failed}</td>
                  <td><span class="badge badge-${r.is_published ? 'success' : 'warning'}">${r.is_published ? 'Published' : 'Draft'}</span></td>
                  <td>
                    <button class="btn btn-outline btn-sm" onclick="window._viewResult('${r.id}')">View</button>
                    ${r.doc_ref ? `<a href="/verify?ref=${r.doc_ref}" target="_blank" class="btn btn-outline btn-sm ml-1">PDF</a>` : ''}
                  </td>
                </tr>`).join('')}
            </tbody>
          </table>
        </div>
      </div>`;

    window._calculateResults = async (sID, tID, cID) => {
      try {
        await api.post('/api/results/calculate', { session_id: sID, term_id: tID, class_id: cID });
        showToast('Results calculated', 'success');
        loadResults();
      } catch (err) {
        showToast(err.message || 'Calculation failed', 'error');
      }
    };

    window._viewResult = async (resultID) => {
      try {
        const result = await api.get(`/api/results/${resultID}`);
        showResultModal(result);
      } catch {
        showToast('Failed to load result', 'error');
      }
    };
  } catch (err) {
    container.innerHTML = `<div class="card p-6 text-danger text-sm">Failed to load results.</div>`;
  }
}

async function publishResults(sessionID, termID, classID) {
  if (!confirm('Publish results? Students and parents will be able to view them.')) return;
  try {
    await api.post('/api/results/publish', { session_id: sessionID, term_id: termID, class_id: classID });
    showToast('Results published successfully', 'success');
    loadResults();
  } catch (err) {
    showToast(err.message || 'Failed to publish', 'error');
  }
}

function showResultModal(result) {
  const modal = document.createElement('div');
  modal.className = 'fixed inset-0 z-50 flex items-center justify-center p-4';
  modal.innerHTML = `
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" onclick="this.parentElement.remove()"></div>
    <div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-3xl max-h-[90vh] overflow-y-auto z-10 p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold text-primary">Result Details</h2>
        <button onclick="this.closest('.fixed').remove()" class="text-secondary hover:text-primary">✕</button>
      </div>
      <div class="grid grid-cols-3 gap-4 mb-6 p-4 bg-neutral-50 rounded-xl">
        <div><p class="label">Average</p><p class="text-2xl font-bold text-primary">${result.average?.toFixed(1)}%</p></div>
        <div><p class="label">Position</p><p class="text-2xl font-bold text-primary">${result.class_position ? ordinal(result.class_position) : '--'}</p></div>
        <div><p class="label">Attendance</p><p class="text-2xl font-bold text-primary">${result.attendance_present}/${result.attendance_total}</p></div>
      </div>
      <table class="data-table w-full text-sm">
        <thead>
          <tr>
            <th class="text-left">Subject</th>
            <th>CA</th>
            <th>Exam</th>
            <th>Total</th>
            <th>Grade</th>
            <th>Position</th>
          </tr>
        </thead>
        <tbody>
          ${(result.subjects || []).map(s => `
            <tr>
              <td>${s.subject_name || s.subject_id}</td>
              <td>${s.ca_total}</td>
              <td>${s.exam_score}</td>
              <td class="font-semibold">${s.total}</td>
              <td><span class="badge badge-${gradeColor(s.grade)}">${s.grade}</span></td>
              <td>${s.subject_position ? ordinal(s.subject_position) : '--'}</td>
            </tr>`).join('')}
        </tbody>
      </table>
      ${result.class_teacher_remark ? `<div class="mt-4 p-3 bg-neutral-50 rounded-lg"><p class="label">Class Teacher Remark</p><p class="text-sm">${result.class_teacher_remark}</p></div>` : ''}
    </div>`;
  document.body.appendChild(modal);
}

function canPublish(role) {
  // EXAM_OFFICER has PermPublishResults via ManageResults; include explicitly.
  return ['OWNER', 'PRINCIPAL', 'VICE_PRINCIPAL', 'HEAD_TEACHER', 'ASST_HEAD_TEACHER', 'EXAM_OFFICER'].includes(role);
}

function ordinal(n) {
  const s = ['th', 'st', 'nd', 'rd'];
  const v = n % 100;
  return n + (s[(v - 20) % 10] || s[v] || s[0]);
}

function gradeColor(grade) {
  if (!grade) return 'neutral';
  if (grade === 'A') return 'success';
  if (grade === 'B' || grade === 'C') return 'warning';
  return 'danger';
}
