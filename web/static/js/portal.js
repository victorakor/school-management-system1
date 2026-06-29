/**
 * portal.js — Portal shell: sidebar navigation, routing, role-based nav, and state.
 */

import { api } from './shared/api.js';
import { $, $$, prefersReducedMotion } from './shared/utils.js';
import { initNotifications } from './shared/notifications.js';

// ─── Portal State ─────────────────────────────────────────────────────────────

const state = {
  user: null,
  currentSection: null,
  sidebarCollapsed: false,
};

// ─── Init ─────────────────────────────────────────────────────────────────────

async function init() {
  try {
    state.user = await api.get('/api/users/me');
    renderSidebar();
    initNotifications();
    handleRouting();
    initSidebarToggle();
    initMobileNav();
  } catch (err) {
    // Redirect to login if not authenticated
    window.location.href = '/login';
  }
}

// ─── Sidebar Rendering ────────────────────────────────────────────────────────

const NAV_ITEMS = [
  { id: 'dashboard',   label: 'Dashboard',    icon: 'home',         roles: ['ALL'] },
  { id: 'admissions',  label: 'Admissions',   icon: 'file-text',    roles: ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','ADMISSIONS_OFFICER','PARENT'] },
  { id: 'students',    label: 'Students',     icon: 'users',        roles: ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','EXAM_OFFICER','TEACHER','CLASS_TEACHER'] },
  { id: 'attendance',  label: 'Attendance',   icon: 'check-square', roles: ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','TEACHER','CLASS_TEACHER'] },
  { id: 'scores',      label: 'Scores',       icon: 'edit-3',       roles: ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','EXAM_OFFICER','TEACHER','CLASS_TEACHER'] },
  { id: 'results',     label: 'Results',      icon: 'bar-chart-2',  roles: ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','EXAM_OFFICER','STUDENT','PUPIL','PARENT'] },
  { id: 'quizzes',     label: 'Quizzes',      icon: 'help-circle',  roles: ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','EXAM_OFFICER','TEACHER','CLASS_TEACHER','STUDENT','PUPIL','PARENT'] },
  { id: 'timetable',   label: 'Timetable',    icon: 'calendar',     roles: ['ALL'] },
  { id: 'finance',     label: 'Finance',      icon: 'dollar-sign',  roles: ['OWNER','BURSAR','PARENT'] },
  { id: 'feed',        label: 'Activities',   icon: 'image',        roles: ['ALL'] },
  { id: 'users',       label: 'Staff',        icon: 'user-check',   roles: ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','ICT_ADMIN'] },
  { id: 'settings',    label: 'Settings',     icon: 'settings',     roles: ['OWNER','ICT_ADMIN'] },
];

function renderSidebar() {
  const sidebar = $('#portal-sidebar');
  const navList = $('#sidebar-nav');
  const userNameEl = $('#sidebar-user-name');
  const userRoleEl = $('#sidebar-user-role');
  const userAvatarEl = $('#sidebar-user-avatar');

  if (!sidebar || !state.user) return;

  // User info
  if (userNameEl) userNameEl.textContent = state.user.full_name;
  if (userRoleEl) userRoleEl.textContent = formatRole(state.user.role);
  if (userAvatarEl) {
    if (state.user.avatar_url) {
      userAvatarEl.src = state.user.avatar_url;
    } else {
      userAvatarEl.alt = state.user.full_name;
      userAvatarEl.parentElement.innerHTML = `
        <div class="w-10 h-10 rounded-full bg-accent/20 flex items-center justify-center text-accent font-bold text-sm" aria-hidden="true">
          ${state.user.full_name.charAt(0).toUpperCase()}
        </div>`;
    }
  }

  // Nav items
  if (!navList) return;
  const role = state.user.role;
  const visibleItems = NAV_ITEMS.filter(item =>
    item.roles.includes('ALL') || item.roles.includes(role)
  );

  navList.innerHTML = visibleItems.map(item => `
    <li>
      <a
        href="/portal/${item.id}"
        class="sidebar-item"
        data-section="${item.id}"
        aria-label="${item.label}"
      >
        <span class="sidebar-icon flex-shrink-0" aria-hidden="true">${getLucideIcon(item.icon)}</span>
        <span class="sidebar-label">${item.label}</span>
      </a>
    </li>`).join('');

  // Highlight active item
  updateActiveNavItem();
}

// ─── Routing ──────────────────────────────────────────────────────────────────

function handleRouting() {
  // Handle browser back/forward
  window.addEventListener('popstate', () => {
    loadSection(getCurrentSection());
  });

  // Handle nav link clicks
  document.addEventListener('click', (e) => {
    const link = e.target.closest('[data-section]');
    if (!link) return;
    e.preventDefault();
    const section = link.dataset.section;
    navigateTo(section);
  });

  // Load initial section
  loadSection(getCurrentSection());
}

function getCurrentSection() {
  const path = window.location.pathname;
  const match = path.match(/\/portal\/([^/]+)/);
  return match ? match[1] : 'dashboard';
}

function navigateTo(section) {
  history.pushState({}, '', `/portal/${section}`);
  loadSection(section);
}

// Map of section id → the exported init function name in that module
const SECTION_INIT_FN = {
  dashboard:    'initDashboard',
  admissions:   'initAdmissions',
  students:     'initStudents',
  attendance:   'initAttendance',
  scores:       'initScores',
  results:      'initResults',
  quizzes:      null,   // handled inline below
  timetable:    'initTimetable',
  finance:      'initFinance',
  feed:         'initFeedAdmin',
  users:        'initUsers',
  settings:     'initSettings',
  notifications:'initNotificationsPage',
};

async function loadSection(section) {
  state.currentSection = section;
  updateActiveNavItem();

  // Update browser page title
  const titleEl = $('#portal-page-title');
  const navItem = NAV_ITEMS.find(n => n.id === section);
  if (titleEl && navItem) titleEl.textContent = navItem.label;

  const content = $('#portal-content');
  if (!content) return;

  // Page exit animation
  if (!prefersReducedMotion) {
    content.style.opacity = '0';
    content.style.transition = 'opacity 150ms ease-in';
  }

  try {
    const fnName = SECTION_INIT_FN[section];

    // Special case: quizzes section shows a list — the proctored quiz module
    // (quiz.js initQuiz) is launched separately when a student clicks "Start Quiz".
    if (section === 'quizzes') {
      await renderQuizzesList(content, state.user);
    } else if (fnName) {
      const module = await import(`./portal/${section}.js`);
      if (typeof module[fnName] === 'function') {
        await module[fnName](content, state.user);
      } else {
        content.innerHTML = renderComingSoon(section);
      }
    } else {
      content.innerHTML = renderComingSoon(section);
    }
  } catch (err) {
    console.error(`[portal] failed to load section "${section}":`, err);
    content.innerHTML = renderComingSoon(section);
  }

  // Page enter animation
  if (!prefersReducedMotion) {
    content.style.opacity = '0';
    content.style.transform = 'translateY(8px)';
    content.style.transition = 'opacity 150ms ease-out, transform 150ms ease-out';
    requestAnimationFrame(() => {
      content.style.opacity = '1';
      content.style.transform = 'translateY(0)';
    });
  } else {
    content.style.opacity = '1';
  }
}

async function renderQuizzesList(container, user) {
  const canManage = ['OWNER','PRINCIPAL','VICE_PRINCIPAL','HEAD_TEACHER','ASST_HEAD_TEACHER','EXAM_OFFICER','TEACHER','CLASS_TEACHER'].includes(user.role);
  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Quizzes</h1>
      ${canManage ? '<button id="create-quiz-btn" class="btn btn-primary">+ Create Quiz</button>' : ''}
    </div>
    <div class="flex gap-2 mb-4">
      <select id="quiz-status-filter" class="input text-sm py-2">
        <option value="">All Statuses</option>
        <option value="DRAFT">Draft</option>
        <option value="ACTIVE">Active</option>
        <option value="CLOSED">Closed</option>
      </select>
    </div>
    <div id="quizzes-list" class="space-y-3"></div>
  `;

  document.getElementById('quiz-status-filter').addEventListener('change', loadQuizzes);
  await loadQuizzes();

  async function loadQuizzes() {
    const list = document.getElementById('quizzes-list');
    if (!list) return;
    const status = document.getElementById('quiz-status-filter')?.value || '';
    list.innerHTML = `<div class="space-y-3">${Array(4).fill('<div class="skeleton h-20 rounded-2xl"></div>').join('')}</div>`;
    try {
      const url = status ? `/api/quizzes?status=${status}` : '/api/quizzes';
      const data = await api.get(url);
      const quizzes = data.quizzes || data || [];

      if (!quizzes.length) {
        list.innerHTML = `<div class="card p-12 text-center text-text-secondary">No quizzes found.</div>`;
        return;
      }

      list.innerHTML = quizzes.map(q => {
        const isActive = q.status === 'ACTIVE';
        const canStart = ['STUDENT','PUPIL'].includes(user.role) && isActive;
        return `
          <div class="card p-5 flex items-center justify-between gap-4">
            <div class="flex-1 min-w-0">
              <h3 class="font-semibold text-primary truncate">${q.title}</h3>
              <div class="flex gap-3 mt-1 flex-wrap">
                <span class="text-xs text-text-secondary">${q.question_count} questions · ${q.duration_minutes} min</span>
                <span class="badge ${isActive ? 'badge-success' : q.status === 'DRAFT' ? 'badge-warning' : 'badge-neutral'} text-xs">${q.status}</span>
              </div>
            </div>
            ${canStart ? `<button class="btn btn-primary text-sm start-quiz-btn" data-quiz-id="${q.id}">Start</button>` : ''}
            ${canManage && q.status === 'DRAFT' ? `<button class="btn btn-ghost text-sm">Edit</button>` : ''}
          </div>`;
      }).join('');

      list.querySelectorAll('.start-quiz-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
          const quizId = btn.dataset.quizId;
          // Dynamically load the full quiz module and launch it
          const { initQuiz } = await import('./portal/quiz.js');
          await initQuiz(quizId);
        });
      });
    } catch (err) {
      list.innerHTML = `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
    }
  }
}

function renderComingSoon(section) {
  return `
    <div class="flex flex-col items-center justify-center h-64 gap-4">
      <div class="w-16 h-16 rounded-2xl bg-accent/10 flex items-center justify-center">
        <svg class="w-8 h-8 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/></svg>
      </div>
      <div class="text-center">
        <p class="font-semibold text-primary">Module loading…</p>
        <p class="text-xs text-text-secondary mt-1 font-mono">${section}</p>
      </div>
    </div>`;
}

function updateActiveNavItem() {
  $$('[data-section]').forEach(link => {
    const isActive = link.dataset.section === state.currentSection;
    link.classList.toggle('active', isActive);
    link.setAttribute('aria-current', isActive ? 'page' : 'false');
  });
}

// ─── Sidebar Toggle ───────────────────────────────────────────────────────────

function initSidebarToggle() {
  const toggleBtn = $('#sidebar-toggle');
  const sidebar = $('#portal-sidebar');

  toggleBtn?.addEventListener('click', () => {
    state.sidebarCollapsed = !state.sidebarCollapsed;
    sidebar?.classList.toggle('sidebar-collapsed', state.sidebarCollapsed);
    toggleBtn.setAttribute('aria-expanded', String(!state.sidebarCollapsed));
    localStorage.setItem('sidebar-collapsed', state.sidebarCollapsed);
  });

  // Restore state
  const saved = localStorage.getItem('sidebar-collapsed') === 'true';
  if (saved) {
    state.sidebarCollapsed = true;
    sidebar?.classList.add('sidebar-collapsed');
  }
}

// ─── Mobile Bottom Nav ────────────────────────────────────────────────────────

function initMobileNav() {
  const mobileNav = $('#portal-mobile-nav');
  if (!mobileNav || !state.user) return;

  const topItems = NAV_ITEMS
    .filter(item => item.roles.includes('ALL') || item.roles.includes(state.user.role))
    .slice(0, 5);

  mobileNav.innerHTML = topItems.map(item => `
    <a href="/portal/${item.id}" class="flex flex-col items-center gap-1 py-2 px-3 text-white/60 hover:text-white transition-colors min-h-[44px] min-w-[44px]" data-section="${item.id}">
      <span class="w-5 h-5" aria-hidden="true">${getLucideIcon(item.icon)}</span>
      <span class="text-[10px] font-medium">${item.label}</span>
    </a>`).join('');
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatRole(role) {
  const map = {
    OWNER:               'School Owner',
    PRINCIPAL:           'Principal',
    VICE_PRINCIPAL:      'Vice Principal',
    HEAD_TEACHER:        'Head Teacher',
    ASST_HEAD_TEACHER:   'Asst. Head Teacher',
    EXAM_OFFICER:        'Exam Officer',
    ADMISSIONS_OFFICER:  'Admissions Officer',
    BURSAR:              'Bursar',
    TEACHER:             'Teacher',
    CLASS_TEACHER:       'Class Teacher',
    STUDENT:             'Student',
    PUPIL:               'Pupil',
    PARENT:              'Parent',
    BLOG_MANAGER:        'Blog / Media Manager',
    ICT_ADMIN:           'ICT Administrator',
  };
  return map[role] || role;
}

function getLucideIcon(name) {
  // Inline SVG icons (subset of Lucide)
  const icons = {
    'home': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>',
    'file-text': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>',
    'users': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>',
    'check-square': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>',
    'edit-3': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>',
    'bar-chart-2': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>',
    'help-circle': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>',
    'calendar': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>',
    'dollar-sign': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',
    'image': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>',
    'user-check': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><polyline points="17 11 19 13 23 9"/></svg>',
    'settings': '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',
  };
  return icons[name] || '';
}

// ─── Start ────────────────────────────────────────────────────────────────────
init();
