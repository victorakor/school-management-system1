/**
 * students.js — Student management module
 */
import { api } from '../shared/api.js';
import { formatDate, showToast } from '../shared/utils.js';

export async function initStudents(container, user) {
  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Students</h1>
      <div class="flex gap-2">
        <input id="student-search" type="search" placeholder="Search students…" class="input text-sm py-2 w-56" />
        <select id="division-filter" class="input text-sm py-2">
          <option value="">All Divisions</option>
          <option value="NURSERY">Nursery</option>
          <option value="PRIMARY">Primary</option>
          <option value="SECONDARY">Secondary</option>
        </select>
      </div>
    </div>
    <div id="students-list" class="space-y-3"></div>
  `;

  document.getElementById('student-search').addEventListener('input', debounce(loadStudents, 300));
  document.getElementById('division-filter').addEventListener('change', loadStudents);
  await loadStudents();
}

async function loadStudents() {
  const list = document.getElementById('students-list');
  if (!list) return;
  const search = document.getElementById('student-search')?.value || '';
  const division = document.getElementById('division-filter')?.value || '';

  list.innerHTML = skeletonRows(6);
  try {
    let url = '/api/students?';
    if (search) url += `search=${encodeURIComponent(search)}&`;
    if (division) url += `division=${division}&`;
    const data = await api.get(url);
    const students = data.students || data || [];

    if (!students.length) {
      list.innerHTML = `<div class="card p-12 text-center text-text-secondary">No students found.</div>`;
      return;
    }

    list.innerHTML = `
      <div class="card overflow-hidden">
        <table class="data-table w-full">
          <thead><tr>
            <th>Name</th><th>Admission ID</th><th>Class</th><th>Division</th><th>Status</th>
          </tr></thead>
          <tbody>
            ${students.map(s => `
              <tr>
                <td class="font-medium text-primary">${s.full_name}</td>
                <td class="font-mono text-xs text-text-secondary">${s.admission_id || '—'}</td>
                <td>${s.class_name || '—'}</td>
                <td><span class="badge badge-neutral">${s.division || '—'}</span></td>
                <td><span class="badge ${s.is_active ? 'badge-success' : 'badge-warning'}">${s.is_active ? 'Active' : 'Inactive'}</span></td>
              </tr>`).join('')}
          </tbody>
        </table>
      </div>`;
  } catch (err) {
    list.innerHTML = `<div class="card p-6 text-danger text-sm">Failed to load students: ${err.message}</div>`;
  }
}

function skeletonRows(n) {
  return `<div class="card p-4 space-y-3">${Array(n).fill('<div class="skeleton h-8 rounded-lg"></div>').join('')}</div>`;
}

function debounce(fn, ms) {
  let t;
  return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); };
}
