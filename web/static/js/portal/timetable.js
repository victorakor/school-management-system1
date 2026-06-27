/**
 * timetable.js — SortableJS timetable builder + upload interface
 */
import { api } from '../shared/api.js';
import { showToast } from '../shared/utils.js';

const DAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday'];
const PERIODS = 8;

export async function initTimetable(container, user) {
  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Timetable</h1>
      <div class="flex gap-2">
        <button id="tab-builder" class="btn btn-primary btn-sm">Builder</button>
        <button id="tab-upload" class="btn btn-outline btn-sm">Upload File</button>
      </div>
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-4 mb-6">
      <select id="tt-session" class="input"><option value="">Session</option></select>
      <select id="tt-term" class="input"><option value="">Term</option></select>
      <select id="tt-division" class="input">
        <option value="">Division</option>
        <option value="NURSERY">Nursery</option>
        <option value="PRIMARY">Primary</option>
        <option value="SECONDARY">Secondary</option>
      </select>
      <select id="tt-class" class="input"><option value="">Class (optional)</option></select>
    </div>
    <div id="tt-builder-panel">
      <div id="tt-grid-container" class="card overflow-hidden"></div>
      <div class="mt-4 flex gap-3">
        <button id="tt-save-btn" class="btn btn-primary">Save Timetable</button>
        <button id="tt-clear-btn" class="btn btn-outline">Clear Grid</button>
      </div>
    </div>
    <div id="tt-upload-panel" class="hidden card p-6">
      <h3 class="font-semibold text-primary mb-4">Upload Timetable File</h3>
      <div id="tt-dropzone" class="border-2 border-dashed border-neutral-200 rounded-xl p-12 text-center cursor-pointer hover:border-accent transition-colors">
        <svg class="mx-auto mb-3 text-secondary" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
        </svg>
        <p class="text-secondary text-sm">Drop a PDF or XLSX file here, or <span class="text-accent font-medium">click to browse</span></p>
        <p class="text-xs text-secondary mt-1">Max 20MB</p>
        <input type="file" id="tt-file-input" class="hidden" accept=".pdf,.xlsx">
      </div>
      <div id="tt-upload-preview" class="hidden mt-4"></div>
      <button id="tt-upload-btn" class="btn btn-primary mt-4 hidden">Upload Timetable</button>
    </div>
    <div id="tt-history" class="mt-6"></div>
  `;

  await loadFilters();
  setupTabs();
  setupDropzone();
  buildGrid();

  document.getElementById('tt-session').addEventListener('change', onSessionChange);
  document.getElementById('tt-term').addEventListener('change', loadHistory);
  document.getElementById('tt-division').addEventListener('change', loadHistory);
  document.getElementById('tt-class').addEventListener('change', loadHistory);
  document.getElementById('tt-save-btn').addEventListener('click', saveTimetable);
  document.getElementById('tt-clear-btn').addEventListener('click', buildGrid);
}

async function loadFilters() {
  try {
    const [sessions, classes] = await Promise.all([
      api.get('/api/sessions'),
      api.get('/api/classes'),
    ]);
    const sessionSel = document.getElementById('tt-session');
    sessions.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.id;
      opt.textContent = s.name;
      if (s.is_active) opt.selected = true;
      sessionSel.appendChild(opt);
    });
    const classSel = document.getElementById('tt-class');
    classes.forEach(c => {
      const opt = document.createElement('option');
      opt.value = c.id;
      opt.textContent = c.name;
      classSel.appendChild(opt);
    });
    if (sessionSel.value) await onSessionChange();
  } catch {}
}

async function onSessionChange() {
  const sessionID = document.getElementById('tt-session').value;
  if (!sessionID) return;
  try {
    const terms = await api.get(`/api/sessions/${sessionID}/terms`);
    const sel = document.getElementById('tt-term');
    sel.innerHTML = '<option value="">Term</option>';
    terms.forEach(t => {
      const opt = document.createElement('option');
      opt.value = t.id;
      opt.textContent = t.name;
      sel.appendChild(opt);
    });
  } catch {}
}

function setupTabs() {
  document.getElementById('tab-builder').addEventListener('click', () => {
    document.getElementById('tt-builder-panel').classList.remove('hidden');
    document.getElementById('tt-upload-panel').classList.add('hidden');
    document.getElementById('tab-builder').className = 'btn btn-primary btn-sm';
    document.getElementById('tab-upload').className = 'btn btn-outline btn-sm';
  });
  document.getElementById('tab-upload').addEventListener('click', () => {
    document.getElementById('tt-builder-panel').classList.add('hidden');
    document.getElementById('tt-upload-panel').classList.remove('hidden');
    document.getElementById('tab-builder').className = 'btn btn-outline btn-sm';
    document.getElementById('tab-upload').className = 'btn btn-primary btn-sm';
  });
}

function buildGrid(existingData = null) {
  const container = document.getElementById('tt-grid-container');
  const colWidth = `${100 / (DAYS.length + 1)}%`;

  let html = `<div class="overflow-x-auto"><table class="w-full text-sm border-collapse">
    <thead>
      <tr class="bg-primary text-white">
        <th class="p-3 text-left font-medium" style="width:80px">Period</th>
        ${DAYS.map(d => `<th class="p-3 text-center font-medium">${d}</th>`).join('')}
      </tr>
    </thead>
    <tbody>`;

  for (let p = 1; p <= PERIODS; p++) {
    html += `<tr class="${p % 2 === 0 ? 'bg-neutral-50' : 'bg-white'}">
      <td class="p-3 font-medium text-secondary border-r border-neutral-100">P${p}</td>
      ${DAYS.map(day => {
        const existing = existingData?.find(r => r.period === p)?.days?.[day];
        return `<td class="p-1 border border-neutral-100">
          <div class="tt-cell rounded-lg p-2 min-h-[60px] bg-white hover:bg-accent/5 transition-colors cursor-pointer"
               data-period="${p}" data-day="${day}"
               onclick="window._ttEditCell(this)">
            ${existing ? `
              <p class="text-xs font-semibold text-primary">${existing.subject_name || ''}</p>
              <p class="text-xs text-secondary">${existing.teacher_name || ''}</p>
            ` : `<p class="text-xs text-neutral-300 text-center mt-2">+ Add</p>`}
          </div>
        </td>`;
      }).join('')}
    </tr>`;
  }

  html += `</tbody></table></div>`;
  container.innerHTML = html;

  window._ttEditCell = (cell) => openCellEditor(cell);
}

function openCellEditor(cell) {
  const period = cell.dataset.period;
  const day = cell.dataset.day;

  const modal = document.createElement('div');
  modal.className = 'fixed inset-0 z-50 flex items-center justify-center p-4';
  modal.innerHTML = `
    <div class="fixed inset-0 bg-black/40" onclick="this.parentElement.remove()"></div>
    <div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-sm z-10 p-6">
      <h3 class="font-semibold text-primary mb-4">Period ${period} — ${day}</h3>
      <div class="space-y-3">
        <div>
          <label class="label block mb-1">Subject</label>
          <input type="text" id="cell-subject" class="input w-full" placeholder="Subject name">
        </div>
        <div>
          <label class="label block mb-1">Teacher</label>
          <input type="text" id="cell-teacher" class="input w-full" placeholder="Teacher name">
          <input type="hidden" id="cell-teacher-id">
        </div>
      </div>
      <div class="flex gap-2 mt-4">
        <button class="btn btn-primary flex-1" onclick="window._ttSaveCell(this, '${period}', '${day}')">Save</button>
        <button class="btn btn-outline flex-1" onclick="this.closest('.fixed').remove()">Cancel</button>
        <button class="btn btn-danger btn-sm" onclick="window._ttClearCell('${period}', '${day}'); this.closest('.fixed').remove()">Clear</button>
      </div>
    </div>`;
  document.body.appendChild(modal);

  window._ttSaveCell = (btn, p, d) => {
    const subject = document.getElementById('cell-subject').value;
    const teacher = document.getElementById('cell-teacher').value;
    const targetCell = document.querySelector(`.tt-cell[data-period="${p}"][data-day="${d}"]`);
    if (targetCell && subject) {
      targetCell.innerHTML = `
        <p class="text-xs font-semibold text-primary">${subject}</p>
        <p class="text-xs text-secondary">${teacher}</p>`;
      targetCell.dataset.subjectName = subject;
      targetCell.dataset.teacherName = teacher;
    }
    btn.closest('.fixed').remove();
  };

  window._ttClearCell = (p, d) => {
    const targetCell = document.querySelector(`.tt-cell[data-period="${p}"][data-day="${d}"]`);
    if (targetCell) {
      targetCell.innerHTML = `<p class="text-xs text-neutral-300 text-center mt-2">+ Add</p>`;
      delete targetCell.dataset.subjectName;
      delete targetCell.dataset.teacherName;
    }
  };
}

async function saveTimetable() {
  const sessionID = document.getElementById('tt-session').value;
  const termID = document.getElementById('tt-term').value;
  const divisionID = document.getElementById('tt-division').value;
  const classID = document.getElementById('tt-class').value;

  if (!sessionID || !termID || !divisionID) {
    showToast('Please select session, term, and division', 'error');
    return;
  }

  // Collect grid data
  const data = [];
  for (let p = 1; p <= PERIODS; p++) {
    const row = { period: p, days: {} };
    DAYS.forEach(day => {
      const cell = document.querySelector(`.tt-cell[data-period="${p}"][data-day="${day}"]`);
      if (cell?.dataset.subjectName) {
        row.days[day] = {
          subject_name: cell.dataset.subjectName,
          teacher_name: cell.dataset.teacherName || '',
        };
      }
    });
    data.push(row);
  }

  try {
    await api.post('/api/timetables/builder', {
      session_id: sessionID,
      term_id: termID,
      division_id: divisionID,
      class_id: classID || undefined,
      data,
    });
    showToast('Timetable saved successfully', 'success');
    loadHistory();
  } catch (err) {
    showToast(err.message || 'Failed to save timetable', 'error');
  }
}

function setupDropzone() {
  const dropzone = document.getElementById('tt-dropzone');
  const fileInput = document.getElementById('tt-file-input');

  dropzone.addEventListener('click', () => fileInput.click());
  dropzone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropzone.classList.add('border-accent', 'bg-accent/5');
  });
  dropzone.addEventListener('dragleave', () => {
    dropzone.classList.remove('border-accent', 'bg-accent/5');
  });
  dropzone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropzone.classList.remove('border-accent', 'bg-accent/5');
    const file = e.dataTransfer.files[0];
    if (file) handleFileSelect(file);
  });
  fileInput.addEventListener('change', () => {
    if (fileInput.files[0]) handleFileSelect(fileInput.files[0]);
  });
}

function handleFileSelect(file) {
  const preview = document.getElementById('tt-upload-preview');
  const uploadBtn = document.getElementById('tt-upload-btn');
  preview.innerHTML = `
    <div class="flex items-center gap-3 p-3 bg-neutral-50 rounded-lg">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-accent">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      </svg>
      <div>
        <p class="text-sm font-medium">${file.name}</p>
        <p class="text-xs text-secondary">${(file.size / 1024 / 1024).toFixed(2)} MB</p>
      </div>
    </div>`;
  preview.classList.remove('hidden');
  uploadBtn.classList.remove('hidden');
  uploadBtn.onclick = () => uploadFile(file);
}

async function uploadFile(file) {
  const sessionID = document.getElementById('tt-session').value;
  const termID = document.getElementById('tt-term').value;
  const divisionID = document.getElementById('tt-division').value;

  if (!sessionID || !termID || !divisionID) {
    showToast('Please select session, term, and division first', 'error');
    return;
  }

  try {
    // Get Cloudinary signature
    const sig = await api.post('/api/upload/sign', { folder: 'timetables', asset_type: 'raw' });
    const formData = new FormData();
    formData.append('file', file);
    formData.append('api_key', sig.api_key);
    formData.append('timestamp', sig.timestamp);
    formData.append('signature', sig.signature);
    formData.append('folder', sig.folder);

    const res = await fetch(`https://api.cloudinary.com/v1_1/${sig.cloud_name}/raw/upload`, {
      method: 'POST',
      body: formData,
    });
    const cloudData = await res.json();

    const fileType = file.name.endsWith('.xlsx') ? 'xlsx' : 'pdf';
    await api.post('/api/timetables/upload', {
      session_id: sessionID,
      term_id: termID,
      division_id: divisionID,
      file_url: cloudData.secure_url,
      file_type: fileType,
    });

    showToast('Timetable uploaded successfully', 'success');
    loadHistory();
  } catch (err) {
    showToast(err.message || 'Upload failed', 'error');
  }
}

async function loadHistory() {
  const sessionID = document.getElementById('tt-session').value;
  const termID = document.getElementById('tt-term').value;
  const divisionID = document.getElementById('tt-division').value;
  const historyEl = document.getElementById('tt-history');

  if (!sessionID || !termID || !divisionID) return;

  try {
    const history = await api.get(`/api/timetables/history?division_id=${divisionID}&session_id=${sessionID}&term_id=${termID}`);
    if (!history.length) {
      historyEl.innerHTML = '';
      return;
    }
    historyEl.innerHTML = `
      <h3 class="font-semibold text-primary mb-3">Version History</h3>
      <div class="space-y-2">
        ${history.map(tt => `
          <div class="card p-3 flex items-center justify-between">
            <div>
              <span class="badge badge-${tt.is_current ? 'success' : 'neutral'} mr-2">v${tt.version}</span>
              <span class="text-sm">${tt.type.replace(/_/g, ' ')}</span>
              <span class="text-xs text-secondary ml-2">${tt.effective_from ? new Date(tt.effective_from).toLocaleDateString() : ''}</span>
            </div>
            ${tt.is_current ? '<span class="text-xs text-success font-medium">Current</span>' : ''}
          </div>`).join('')}
      </div>`;
  } catch {}
}
