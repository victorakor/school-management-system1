/**
 * notifications.js — Full notifications page for portal
 */
import { api } from '../shared/api.js';
import { formatDate } from '../shared/utils.js';

export async function initNotificationsPage(container) {
  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Notifications</h1>
      <button id="mark-all-read-btn" class="btn btn-outline btn-sm">Mark All Read</button>
    </div>
    <div id="notif-list" class="space-y-2"></div>
    <div id="notif-load-more" class="text-center mt-6 hidden">
      <button id="load-more-btn" class="btn btn-outline">Load More</button>
    </div>
  `;

  let page = 1;
  document.getElementById('mark-all-read-btn').addEventListener('click', async () => {
    await api.put('/api/notifications/read-all', {});
    loadNotifications(1, true);
  });
  document.getElementById('load-more-btn')?.addEventListener('click', () => {
    page++;
    loadNotifications(page, false);
  });

  await loadNotifications(1, true);

  async function loadNotifications(p, replace) {
    const list = document.getElementById('notif-list');
    if (replace) list.innerHTML = skeletonRows(5);

    try {
      const data = await api.get(`/api/notifications?page=${p}&limit=20`);
      if (replace) list.innerHTML = '';

      if (!data.length && p === 1) {
        list.innerHTML = `<div class="card p-12 text-center text-secondary">No notifications yet.</div>`;
        return;
      }

      data.forEach(notif => {
        const el = document.createElement('div');
        el.className = `card p-4 flex items-start gap-3 cursor-pointer transition-colors ${!notif.is_read ? 'border-l-4 border-accent bg-accent/5' : ''}`;
        el.innerHTML = `
          <div class="w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${!notif.is_read ? 'bg-accent text-white' : 'bg-neutral-100 text-secondary'}">
            ${notifIcon(notif.type)}
          </div>
          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm text-primary">${notif.title}</p>
            <p class="text-sm text-secondary mt-0.5 line-clamp-2">${notif.body}</p>
            <p class="text-xs text-secondary mt-1">${formatDate(notif.created_at)}</p>
          </div>
          ${!notif.is_read ? `<div class="w-2 h-2 rounded-full bg-accent flex-shrink-0 mt-2"></div>` : ''}
        `;
        el.addEventListener('click', async () => {
          if (!notif.is_read) {
            await api.put(`/api/notifications/${notif.id}/read`, {});
            el.classList.remove('border-l-4', 'border-accent', 'bg-accent/5');
            el.querySelector('.w-2.h-2')?.remove();
          }
        });
        list.appendChild(el);
      });

      const loadMoreEl = document.getElementById('notif-load-more');
      if (data.length === 20) {
        loadMoreEl.classList.remove('hidden');
      } else {
        loadMoreEl.classList.add('hidden');
      }
    } catch {
      if (replace) list.innerHTML = `<div class="card p-6 text-danger text-sm">Failed to load notifications.</div>`;
    }
  }
}

function notifIcon(type) {
  const icons = {
    ADMISSION: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/></svg>`,
    RESULT: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="8" r="6"/><path d="M15.477 12.89L17 22l-5-3-5 3 1.523-9.11"/></svg>`,
    PAYMENT: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="1" y="4" width="22" height="16" rx="2"/><line x1="1" y1="10" x2="23" y2="10"/></svg>`,
    QUIZ: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>`,
    ANNOUNCEMENT: `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 17H2a3 3 0 0 0 3-3V9a7 7 0 0 1 14 0v5a3 3 0 0 0 3 3zm-8.27 4a2 2 0 0 1-3.46 0"/></svg>`,
  };
  return icons[type] || icons.ANNOUNCEMENT;
}

function skeletonRows(n) {
  return Array(n).fill(0).map(() =>
    `<div class="card p-4 flex items-start gap-3">
      <div class="skeleton w-8 h-8 rounded-full flex-shrink-0"></div>
      <div class="flex-1 space-y-2">
        <div class="skeleton h-4 w-48 rounded"></div>
        <div class="skeleton h-3 w-full rounded"></div>
      </div>
    </div>`
  ).join('');
}
