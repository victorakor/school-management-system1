/**
 * dashboard.js — Portal dashboard module
 * Renders role-specific stat cards, quick panels, and charts.
 */
import { api } from '../shared/api.js';
import { formatDate, formatCurrency, createElement, showSkeleton, hideSkeleton } from '../shared/utils.js';

const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

// ─── Entry Point ──────────────────────────────────────────────────────────────

export async function initDashboard(container, user) {
  container.innerHTML = `
    <div class="dashboard-header mb-6">
      <h1 class="text-2xl font-bold text-primary">Welcome back, ${user.full_name.split(' ')[0]} 👋</h1>
      <p class="text-secondary text-sm mt-1">${formatDate(new Date())} · ${user.role.replace(/_/g, ' ')}</p>
    </div>
    <div id="stat-cards" class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8"></div>
    <div id="dashboard-panels" class="grid grid-cols-1 lg:grid-cols-2 gap-6"></div>
  `;

  showSkeleton(document.getElementById('stat-cards'), 4, 'h-28');

  try {
    const stats = await api.get('/api/dashboard/stats');
    renderStatCards(document.getElementById('stat-cards'), stats, user.role);
    await renderPanels(document.getElementById('dashboard-panels'), user);
  } catch (err) {
    document.getElementById('stat-cards').innerHTML =
      `<p class="col-span-4 text-danger text-sm">Failed to load dashboard data.</p>`;
  }
}

// ─── Stat Cards ───────────────────────────────────────────────────────────────

function renderStatCards(container, stats, role) {
  container.innerHTML = '';
  const cards = getStatCardsForRole(stats, role);

  cards.forEach((card, i) => {
    const el = document.createElement('div');
    el.className = 'card p-5 flex flex-col gap-2 cursor-default';
    el.style.opacity = '0';
    el.style.transform = 'translateY(16px)';
    el.innerHTML = `
      <div class="flex items-center justify-between">
        <span class="text-secondary text-xs font-medium uppercase tracking-wide">${card.label}</span>
        <span class="w-8 h-8 rounded-lg flex items-center justify-center" style="background:${card.color}20">
          ${card.icon}
        </span>
      </div>
      <div class="text-3xl font-bold text-primary" data-count="${card.value}">0</div>
      ${card.sub ? `<p class="text-xs text-secondary">${card.sub}</p>` : ''}
    `;
    container.appendChild(el);

    // Animate in
    const delay = reduced ? 0 : i * 80;
    setTimeout(() => {
      el.style.transition = 'opacity 300ms ease-out, transform 300ms ease-out';
      el.style.opacity = '1';
      el.style.transform = 'translateY(0)';
      animateCount(el.querySelector('[data-count]'), card.value);
    }, delay);

    // Hover lift
    el.addEventListener('mouseenter', () => {
      if (!reduced) el.style.transform = 'translateY(-4px)';
    });
    el.addEventListener('mouseleave', () => {
      el.style.transform = 'translateY(0)';
    });
  });
}

function getStatCardsForRole(stats, role) {
  const base = [
    { label: 'Total Students', value: stats.total_students || 0, color: '#2563EB', icon: iconSVG('users'), sub: 'Enrolled this session' },
    { label: 'Active Staff', value: stats.total_staff || 0, color: '#10B981', icon: iconSVG('briefcase'), sub: 'All roles' },
    { label: 'Applications', value: stats.pending_applications || 0, color: '#F59E0B', icon: iconSVG('file-text'), sub: 'Pending review' },
    { label: 'Quizzes Active', value: stats.active_quizzes || 0, color: '#0F2557', icon: iconSVG('zap'), sub: 'Currently running' },
  ];

  if (role === 'BURSAR') {
    return [
      { label: 'Collected Today', value: stats.collected_today || 0, color: '#10B981', icon: iconSVG('trending-up'), sub: 'NGN', format: 'currency' },
      { label: 'This Term', value: stats.collected_term || 0, color: '#2563EB', icon: iconSVG('bar-chart-2'), sub: 'Total collected', format: 'currency' },
      { label: 'Outstanding', value: stats.outstanding_balance || 0, color: '#F43F5E', icon: iconSVG('alert-circle'), sub: 'Total owed', format: 'currency' },
      { label: 'Debtors', value: stats.debtor_count || 0, color: '#F59E0B', icon: iconSVG('users'), sub: 'Students with balance' },
    ];
  }

  return base;
}

function animateCount(el, target) {
  if (reduced || typeof target !== 'number') {
    el.textContent = target;
    return;
  }
  const duration = 1000;
  const start = performance.now();
  const update = (now) => {
    const progress = Math.min((now - start) / duration, 1);
    const eased = 1 - Math.pow(1 - progress, 3);
    el.textContent = Math.floor(eased * target).toLocaleString();
    if (progress < 1) requestAnimationFrame(update);
  };
  requestAnimationFrame(update);
}

// ─── Dashboard Panels ─────────────────────────────────────────────────────────

async function renderPanels(container, user) {
  container.innerHTML = '';

  if (['OWNER', 'PRINCIPAL', 'VICE_PRINCIPAL', 'HEAD_TEACHER', 'ASST_HEAD_TEACHER'].includes(user.role)) {
    await Promise.all([
      renderRecentApplicationsPanel(container),
      renderPendingScoresPanel(container),
    ]);
  } else if (user.role === 'TEACHER') {
    await Promise.all([
      renderTodayTimetablePanel(container),
      renderPendingActionsPanel(container),
    ]);
  } else if (user.role === 'BURSAR') {
    await Promise.all([
      renderRecentPaymentsPanel(container),
      renderDebtorPanel(container),
    ]);
  } else if (user.role === 'STUDENT' || user.role === 'PUPIL') {
    await Promise.all([
      renderTodayTimetablePanel(container),
      renderLatestResultPanel(container),
    ]);
  } else if (user.role === 'PARENT') {
    await renderParentPanel(container);
  }
}

async function renderRecentApplicationsPanel(container) {
  const panel = createPanel('Recent Applications', 'file-text');
  container.appendChild(panel);
  const body = panel.querySelector('.panel-body');
  try {
    const data = await api.get('/api/admissions/applications?limit=5');
    if (!data.length) {
      body.innerHTML = emptyState('No recent applications');
      return;
    }
    body.innerHTML = `<ul class="divide-y divide-neutral-100">
      ${data.map(app => `
        <li class="py-3 flex items-center justify-between">
          <div>
            <p class="font-medium text-sm text-primary">${app.child_name}</p>
            <p class="text-xs text-secondary">${app.division} · ${formatDate(app.created_at)}</p>
          </div>
          <span class="badge badge-${statusColor(app.status)}">${app.status.replace(/_/g, ' ')}</span>
        </li>`).join('')}
    </ul>`;
  } catch {
    body.innerHTML = `<p class="text-danger text-sm">Failed to load applications.</p>`;
  }
}

async function renderPendingScoresPanel(container) {
  const panel = createPanel('Pending Score Approvals', 'check-square');
  container.appendChild(panel);
  const body = panel.querySelector('.panel-body');
  try {
    const data = await api.get('/api/scores?status=SUBMITTED&limit=5');
    if (!data.length) {
      body.innerHTML = emptyState('No pending approvals');
      return;
    }
    body.innerHTML = `<ul class="divide-y divide-neutral-100">
      ${data.map(s => `
        <li class="py-3 flex items-center justify-between">
          <div>
            <p class="font-medium text-sm text-primary">${s.subject_name}</p>
            <p class="text-xs text-secondary">${s.class_name} · ${s.teacher_name}</p>
          </div>
          <button class="btn btn-sm btn-primary" onclick="window.portalNavigate('/portal/scores')">Review</button>
        </li>`).join('')}
    </ul>`;
  } catch {
    body.innerHTML = emptyState('No pending approvals');
  }
}

async function renderTodayTimetablePanel(container) {
  const panel = createPanel("Today's Timetable", 'calendar');
  container.appendChild(panel);
  const body = panel.querySelector('.panel-body');
  const now = new Date();
  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
  const today = days[now.getDay()];
  body.innerHTML = `<p class="text-secondary text-sm mb-3">${today}</p>
    <div class="space-y-2" id="timetable-slots">
      ${[...Array(6)].map((_, i) => `
        <div class="flex items-center gap-3 p-2 rounded-lg bg-neutral-50">
          <span class="text-xs text-secondary w-16">Period ${i + 1}</span>
          <span class="text-sm text-secondary italic">No class</span>
        </div>`).join('')}
    </div>`;
}

async function renderLatestResultPanel(container) {
  const panel = createPanel('Latest Results', 'award');
  container.appendChild(panel);
  const body = panel.querySelector('.panel-body');
  try {
    const data = await api.get('/api/results?limit=1');
    if (!data.length) {
      body.innerHTML = emptyState('No results published yet');
      return;
    }
    const r = data[0];
    body.innerHTML = `
      <div class="text-center py-4">
        <div class="text-4xl font-bold text-primary">${r.average?.toFixed(1) || '--'}%</div>
        <p class="text-secondary text-sm mt-1">Overall Average</p>
        <p class="text-xs text-secondary mt-1">${r.class_position ? utils.ordinal(r.class_position) + ' in class' : ''}</p>
        <a href="/portal/results" class="btn btn-primary mt-4 inline-block">View Full Report</a>
      </div>`;
  } catch {
    body.innerHTML = emptyState('No results available');
  }
}

async function renderRecentPaymentsPanel(container) {
  const panel = createPanel('Recent Payments', 'credit-card');
  container.appendChild(panel);
  const body = panel.querySelector('.panel-body');
  body.innerHTML = emptyState('Payment history will appear here');
}

async function renderDebtorPanel(container) {
  const panel = createPanel('Outstanding Balances', 'alert-circle');
  container.appendChild(panel);
  const body = panel.querySelector('.panel-body');
  body.innerHTML = emptyState('Debtor list will appear here');
}

async function renderPendingActionsPanel(container) {
  const panel = createPanel('Pending Actions', 'bell');
  container.appendChild(panel);
  const body = panel.querySelector('.panel-body');
  body.innerHTML = emptyState('No pending actions');
}

async function renderParentPanel(container) {
  const panel = createPanel('My Children', 'users');
  panel.style.gridColumn = '1 / -1';
  container.appendChild(panel);
  const body = panel.querySelector('.panel-body');
  body.innerHTML = emptyState('Child information will appear here');
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function createPanel(title, icon) {
  const el = document.createElement('div');
  el.className = 'card p-5';
  el.innerHTML = `
    <div class="flex items-center gap-2 mb-4">
      <span class="text-accent">${iconSVG(icon)}</span>
      <h3 class="font-semibold text-primary">${title}</h3>
    </div>
    <div class="panel-body"></div>
  `;
  return el;
}

function emptyState(msg) {
  return `<div class="text-center py-8 text-secondary text-sm">${msg}</div>`;
}

function statusColor(status) {
  const map = { PENDING: 'warning', UNDER_REVIEW: 'warning', ACCEPTED: 'success', DECLINED: 'danger' };
  return map[status] || 'neutral';
}

function iconSVG(name) {
  const icons = {
    users: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
    briefcase: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>`,
    'file-text': `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>`,
    zap: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>`,
    'trending-up': `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>`,
    'bar-chart-2': `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>`,
    'alert-circle': `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>`,
    'check-square': `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>`,
    calendar: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>`,
    award: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="8" r="6"/><path d="M15.477 12.89L17 22l-5-3-5 3 1.523-9.11"/></svg>`,
    'credit-card': `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="1" y="4" width="22" height="16" rx="2"/><line x1="1" y1="10" x2="23" y2="10"/></svg>`,
    bell: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>`,
  };
  return icons[name] || '';
}
