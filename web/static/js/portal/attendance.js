// Roles with PermManageAttendance — can mark attendance.
const CAN_MARK_ATTENDANCE = [
  'OWNER', 'PRINCIPAL', 'VICE_PRINCIPAL', 'HEAD_TEACHER', 'ASST_HEAD_TEACHER',
  'TEACHER', 'CLASS_TEACHER',
];

export async function initAttendance(container, user) {
  const canMark = user && CAN_MARK_ATTENDANCE.includes(user.role);

  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Attendance</h1>
    </div>
    <div class="grid grid-cols-1 ${canMark ? 'lg:grid-cols-2' : ''} gap-6">
      ${canMark ? `
      <div class="card p-6">
        <h2 class="font-semibold text-primary mb-4">Mark Attendance</h2>
        <div class="space-y-4">
          <div>
            <label class="label">Class</label>
            <select id="att-class" class="input w-full"><option value="">Select class…</option></select>
          </div>
          <div>
            <label class="label">Subject</label>
            <select id="att-subject" class="input w-full"><option value="">Select subject…</option></select>
          </div>
          <div>
            <label class="label">Date</label>
            <input id="att-date" type="date" class="input w-full" value="${new Date().toISOString().split('T')[0]}" />
          </div>
          <button id="load-att-sheet" class="btn btn-primary w-full">Load Students</button>
        </div>
        <div id="att-sheet" class="mt-4 space-y-2"></div>
      </div>` : ''}
      <div class="card p-6">
        <h2 class="font-semibold text-primary mb-4">Attendance Summary</h2>
        ${!canMark ? `
        <div class="space-y-3 mb-4">
          <select id="att-class" class="input w-full"><option value="">Select class…</option></select>
        </div>` : ''}
        <div id="att-summary" class="text-text-secondary text-sm">Select a class to view summary.</div>
      </div>
    </div>
  `;

  await loadClasses();
  if (canMark) {
    document.getElementById('load-att-sheet')?.addEventListener('click', loadAttendanceSheet);
  }
}

async function loadClasses() {
  try {
    const data = await api.get('/api/classes');
    const sel = document.getElementById('att-class');
    if (!sel) return;
    (data.classes || data || []).forEach(c => {
      const opt = document.createElement('option');
      opt.value = c.id;
      opt.textContent = c.name + (c.stream ? ` ${c.stream}` : '');
      sel.appendChild(opt);
    });
  } catch {}
}

async function loadAttendanceSheet() {
  const classId = document.getElementById('att-class')?.value;
  const date = document.getElementById('att-date')?.value;
  const sheet = document.getElementById('att-sheet');
  if (!classId || !sheet) return;

  sheet.innerHTML = '<div class="skeleton h-8 rounded-lg"></div>';
  try {
    const data = await api.get(`/api/attendance?class_id=${classId}&date=${date}`);
    const records = data.records || data || [];

    sheet.innerHTML = records.length
      ? records.map(r => `
          <div class="flex items-center justify-between p-3 rounded-xl border border-neutral-100 hover:bg-background transition-colors">
            <span class="text-sm font-medium">${r.student_name || r.student_id}</span>
            <div class="flex gap-2">
              ${['PRESENT','ABSENT','LATE'].map(s => `
                <button class="att-btn px-3 py-1 rounded-lg text-xs font-semibold border transition-colors ${r.status === s ? statusClass(s) : 'border-neutral-200 text-text-secondary hover:border-accent'}"
                  data-student="${r.student_id}" data-status="${s}">${s}</button>`).join('')}
            </div>
          </div>`).join('')
      : '<p class="text-text-secondary text-sm">No students found for this class.</p>';

    sheet.querySelectorAll('.att-btn').forEach(btn => {
      btn.addEventListener('click', () => markAttendance(btn, classId, date));
    });
  } catch (err) {
    sheet.innerHTML = `<p class="text-danger text-sm">${err.message}</p>`;
  }
}

async function markAttendance(btn, classId, date) {
  const studentId = btn.dataset.student;
  const status = btn.dataset.status;
  try {
    await api.post('/api/attendance', { student_id: studentId, class_id: classId, date, status });
    const row = btn.closest('div.flex');
    row.querySelectorAll('.att-btn').forEach(b => {
      b.className = `att-btn px-3 py-1 rounded-lg text-xs font-semibold border transition-colors ${b.dataset.status === status ? statusClass(status) : 'border-neutral-200 text-text-secondary hover:border-accent'}`;
    });
  } catch {}
}

function statusClass(s) {
  return s === 'PRESENT' ? 'bg-success/10 border-success text-success' : s === 'ABSENT' ? 'bg-danger/10 border-danger text-danger' : 'bg-warning/10 border-warning text-warning';
}
