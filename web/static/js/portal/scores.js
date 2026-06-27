/**
 * scores.js — Score entry and approval interface
 */
import { api } from '../shared/api.js';
import { showToast, formatDate } from '../shared/utils.js';

export async function initScores(container, user) {
  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Score Entry</h1>
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-4 mb-6">
      <select id="score-session" class="input"><option value="">Select Session</option></select>
      <select id="score-term" class="input"><option value="">Select Term</option></select>
      <select id="score-class" class="input"><option value="">Select Class</option></select>
      <select id="score-subject" class="input"><option value="">Select Subject</option></select>
    </div>
    <div id="score-sheet-container"></div>
  `;

  await loadSessions();
  document.getElementById('score-session').addEventListener('change', onSessionChange);
  document.getElementById('score-term').addEventListener('change', loadScoreSheet);
  document.getElementById('score-class').addEventListener('change', loadScoreSheet);
  document.getElementById('score-subject').addEventListener('change', loadScoreSheet);
}

async function loadSessions() {
  try {
    const sessions = await api.get('/api/sessions');
    const sel = document.getElementById('score-session');
    sessions.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.id;
      opt.textContent = s.name;
      if (s.is_active) opt.selected = true;
      sel.appendChild(opt);
    });
    if (sel.value) await onSessionChange();
  } catch {}
}

async function onSessionChange() {
  const sessionID = document.getElementById('score-session').value;
  if (!sessionID) return;
  try {
    const [terms, classes, subjects] = await Promise.all([
      api.get(`/api/sessions/${sessionID}/terms`),
      api.get('/api/classes'),
      api.get('/api/subjects'),
    ]);
    populateSelect('score-term', terms, 'name');
    populateSelect('score-class', classes, 'name');
    populateSelect('score-subject', subjects, 'name');
  } catch {}
}

function populateSelect(id, items, labelKey) {
  const sel = document.getElementById(id);
  const current = sel.value;
  sel.innerHTML = `<option value="">Select ${id.replace('score-', '')}</option>`;
  items.forEach(item => {
    const opt = document.createElement('option');
    opt.value = item.id;
    opt.textContent = item[labelKey];
    if (item.id === current) opt.selected = true;
    sel.appendChild(opt);
  });
}

async function loadScoreSheet() {
  const sessionID = document.getElementById('score-session').value;
  const termID = document.getElementById('score-term').value;
  const classID = document.getElementById('score-class').value;
  const subjectID = document.getElementById('score-subject').value;
  const container = document.getElementById('score-sheet-container');

  if (!sessionID || !termID || !classID || !subjectID) {
    container.innerHTML = '';
    return;
  }

  container.innerHTML = `<div class="card p-6"><div class="skeleton h-64 rounded"></div></div>`;

  try {
    const [sheet, structure] = await Promise.all([
      api.get(`/api/scores/sheet?session_id=${sessionID}&term_id=${termID}&class_id=${classID}&subject_id=${subjectID}`),
      api.get(`/api/scores/structure?session_id=${sessionID}&term_id=${termID}&class_id=${classID}&subject_id=${subjectID}`),
    ]);

    renderScoreSheet(container, sheet, structure, { sessionID, termID, classID, subjectID });
  } catch (err) {
    container.innerHTML = `<div class="card p-6 text-danger text-sm">Failed to load score sheet. ${err.message || ''}</div>`;
  }
}

function renderScoreSheet(container, sheet, structure, params) {
  const components = structure?.components || [];
  const status = sheet.status || 'DRAFT';
  const isLocked = status === 'SUBMITTED' || status === 'APPROVED';

  container.innerHTML = `
    <div class="card overflow-hidden">
      <div class="p-4 border-b border-neutral-100 flex items-center justify-between">
        <div>
          <h3 class="font-semibold text-primary">Score Sheet</h3>
          <p class="text-xs text-secondary mt-0.5">Status: <span class="badge badge-${statusColor(status)}">${status}</span></p>
        </div>
        <div class="flex gap-2">
          ${!isLocked ? `
            <button id="save-draft-btn" class="btn btn-outline btn-sm">Save Draft</button>
            <button id="submit-scores-btn" class="btn btn-primary btn-sm">Submit for Approval</button>
          ` : ''}
          ${status === 'SUBMITTED' ? `
            <button id="approve-btn" class="btn btn-success btn-sm">Approve</button>
            <button id="reject-btn" class="btn btn-danger btn-sm">Reject</button>
          ` : ''}
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="data-table w-full">
          <thead>
            <tr>
              <th class="text-left">Student</th>
              ${components.map(c => `<th>${c.name}<br><span class="text-xs font-normal text-secondary">/${c.max_marks}</span></th>`).join('')}
              <th>Exam<br><span class="text-xs font-normal text-secondary">/${structure?.exam_marks || 70}</span></th>
              <th>Total</th>
            </tr>
          </thead>
          <tbody id="score-tbody">
            ${(sheet.students || []).map(student => `
              <tr data-student-id="${student.id}">
                <td class="font-medium">${student.full_name}</td>
                ${components.map((c, ci) => `
                  <td>
                    <input type="number" class="score-input w-16 text-center border rounded px-1 py-0.5 text-sm"
                           data-component="${ci}" data-max="${c.max_marks}"
                           value="${student.scores?.components?.[ci]?.score || ''}"
                           min="0" max="${c.max_marks}" ${isLocked ? 'disabled' : ''}
                           oninput="window._validateScore(this)">
                  </td>`).join('')}
                <td>
                  <input type="number" class="score-input exam-score w-16 text-center border rounded px-1 py-0.5 text-sm"
                         data-max="${structure?.exam_marks || 70}"
                         value="${student.scores?.exam_score || ''}"
                         min="0" max="${structure?.exam_marks || 70}" ${isLocked ? 'disabled' : ''}
                         oninput="window._validateScore(this)">
                </td>
                <td class="font-semibold total-cell" id="total-${student.id}">
                  ${student.scores?.total || '--'}
                </td>
              </tr>`).join('')}
          </tbody>
        </table>
      </div>
    </div>`;

  // Live total calculation
  window._validateScore = (input) => {
    const max = parseInt(input.dataset.max);
    const val = parseInt(input.value);
    if (val > max) {
      input.classList.add('border-danger', 'text-danger');
      input.value = max;
    } else {
      input.classList.remove('border-danger', 'text-danger');
    }
    updateRowTotal(input.closest('tr'));
  };

  function updateRowTotal(row) {
    const inputs = row.querySelectorAll('.score-input');
    let total = 0;
    inputs.forEach(inp => { total += parseInt(inp.value) || 0; });
    const studentID = row.dataset.studentId;
    const totalCell = document.getElementById(`total-${studentID}`);
    if (totalCell) totalCell.textContent = total;
  }

  // Save draft
  document.getElementById('save-draft-btn')?.addEventListener('click', () => saveScores('DRAFT', params));
  document.getElementById('submit-scores-btn')?.addEventListener('click', () => saveScores('SUBMITTED', params));
  document.getElementById('approve-btn')?.addEventListener('click', () => approveScores(sheet.id));
  document.getElementById('reject-btn')?.addEventListener('click', () => rejectScores(sheet.id));
}

async function saveScores(status, params) {
  const rows = document.querySelectorAll('#score-tbody tr');
  const entries = [];
  rows.forEach(row => {
    const studentID = row.dataset.studentId;
    const compInputs = row.querySelectorAll('[data-component]');
    const examInput = row.querySelector('.exam-score');
    const components = [];
    compInputs.forEach((inp, i) => {
      components.push({ index: i, score: parseInt(inp.value) || 0 });
    });
    entries.push({
      student_id: studentID,
      components,
      exam_score: parseInt(examInput?.value) || 0,
    });
  });

  try {
    await api.post('/api/scores/save', {
      ...params,
      status,
      entries,
    });
    showToast(status === 'DRAFT' ? 'Draft saved' : 'Scores submitted for approval', 'success');
    if (status === 'SUBMITTED') loadScoreSheet();
  } catch (err) {
    showToast(err.message || 'Failed to save scores', 'error');
  }
}

async function approveScores(sheetID) {
  try {
    await api.put(`/api/scores/${sheetID}/approve`, {});
    showToast('Scores approved', 'success');
    loadScoreSheet();
  } catch {
    showToast('Failed to approve scores', 'error');
  }
}

async function rejectScores(sheetID) {
  const reason = prompt('Enter rejection reason:');
  if (!reason) return;
  try {
    await api.put(`/api/scores/${sheetID}/reject`, { reason });
    showToast('Scores rejected', 'success');
    loadScoreSheet();
  } catch {
    showToast('Failed to reject scores', 'error');
  }
}

function statusColor(s) {
  const m = { DRAFT: 'neutral', SUBMITTED: 'warning', APPROVED: 'success', REJECTED: 'danger' };
  return m[s] || 'neutral';
}
