/**
 * notifications.js — Notification panel + polling for the portal.
 * Polls GET /api/notifications/unread-count every 30 seconds.
 */

import { api } from './api.js';
import { $, $$, formatDateTime, timeAgo } from './utils.js';

const POLL_INTERVAL = 30_000; // 30 seconds

let pollTimer = null;

/** Initialise the notification system. Call once on portal load. */
export function initNotifications() {
  const bell = $('#notification-bell');
  const badge = $('#notification-badge');
  const panel = $('#notification-panel');
  const closeBtn = $('#close-notifications');
  const markAllBtn = $('#mark-all-read');

  if (!bell) return;

  // Toggle panel on bell click
  bell.addEventListener('click', async () => {
    const isOpen = !panel.classList.contains('hidden');
    if (isOpen) {
      closePanel();
    } else {
      await openPanel();
    }
  });

  if (closeBtn) {
    closeBtn.addEventListener('click', closePanel);
  }

  if (markAllBtn) {
    markAllBtn.addEventListener('click', async () => {
      try {
        await api.put('/api/notifications/read-all');
        await loadNotifications();
        updateBadge(0);
      } catch {}
    });
  }

  // Close panel on backdrop click
  document.addEventListener('click', (e) => {
    if (panel && !panel.classList.contains('hidden') &&
        !panel.contains(e.target) && !bell.contains(e.target)) {
      closePanel();
    }
  });

  // Start polling
  startPolling(badge);
}

async function openPanel() {
  const panel = $('#notification-panel');
  if (!panel) return;

  panel.classList.remove('hidden');
  panel.style.opacity = '0';
  panel.style.transform = 'translateY(-8px)';
  panel.style.transition = 'opacity 200ms ease, transform 200ms ease';

  requestAnimationFrame(() => {
    panel.style.opacity = '1';
    panel.style.transform = 'translateY(0)';
  });

  await loadNotifications();
}

function closePanel() {
  const panel = $('#notification-panel');
  if (!panel) return;
  panel.classList.add('hidden');
}

async function loadNotifications() {
  const list = $('#notification-list');
  if (!list) return;

  list.innerHTML = `
    <div class="flex items-center justify-center py-8">
      <svg class="w-5 h-5 animate-spin text-accent" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
    </div>`;

  try {
    const data = await api.get('/api/notifications', { page: 1 });
    const notifications = Array.isArray(data) ? data : (data?.data ?? []);

    if (notifications.length === 0) {
      list.innerHTML = `
        <div class="text-center py-10 px-4">
          <div class="w-12 h-12 bg-neutral-100 rounded-full flex items-center justify-center mx-auto mb-3">
            <svg class="w-6 h-6 text-text-secondary" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/></svg>
          </div>
          <p class="text-text-secondary text-sm">No notifications yet</p>
        </div>`;
      return;
    }

    list.innerHTML = notifications.map(n => `
      <button
        class="w-full text-left px-4 py-3.5 hover:bg-background transition-colors border-b border-neutral-50 last:border-0 ${!n.is_read ? 'bg-accent/5 border-l-2 border-l-accent' : ''}"
        data-id="${n.id}"
        onclick="markRead('${n.id}', this)"
      >
        <div class="flex items-start gap-3">
          <div class="flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center mt-0.5 ${getNotifColor(n.type)}">
            ${getNotifIcon(n.type)}
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-text-primary truncate">${n.title}</p>
            <p class="text-xs text-text-secondary mt-0.5 line-clamp-2">${n.body}</p>
            <p class="text-xs text-text-secondary/60 mt-1">${timeAgo(n.created_at)}</p>
          </div>
          ${!n.is_read ? '<span class="flex-shrink-0 w-2 h-2 bg-accent rounded-full mt-2" aria-label="Unread"></span>' : ''}
        </div>
      </button>`).join('');

    // Count unread
    const unread = notifications.filter(n => !n.is_read).length;
    updateBadge(unread);

  } catch {
    list.innerHTML = `<p class="text-center text-danger text-sm py-6">Failed to load notifications</p>`;
  }
}

function startPolling(badge) {
  const poll = async () => {
    try {
      const data = await api.get('/api/notifications/unread-count');
      updateBadge(data?.count ?? 0);
    } catch {}
  };

  poll();
  pollTimer = setInterval(poll, POLL_INTERVAL);
}

function updateBadge(count) {
  const badge = $('#notification-badge');
  if (!badge) return;

  if (count > 0) {
    badge.textContent = count > 99 ? '99+' : String(count);
    badge.classList.remove('hidden');
  } else {
    badge.classList.add('hidden');
  }
}

function getNotifColor(type) {
  const map = {
    ADMISSION: 'bg-accent/10 text-accent',
    RESULT: 'bg-success/10 text-success',
    QUIZ: 'bg-gold/10 text-gold',
    PAYMENT: 'bg-emerald-100 text-emerald-600',
    TIMETABLE: 'bg-purple-100 text-purple-600',
    SCORE: 'bg-blue-100 text-blue-600',
    PROMOTION: 'bg-indigo-100 text-indigo-600',
  };
  return map[type] || 'bg-neutral-100 text-text-secondary';
}

function getNotifIcon(type) {
  const icons = {
    ADMISSION: '<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>',
    RESULT: '<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>',
    QUIZ: '<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>',
    PAYMENT: '<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z"/></svg>',
  };
  return icons[type] || '<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"/></svg>';
}

// Expose markRead globally for inline onclick handlers
window.markRead = async (id, btn) => {
  try {
    await api.put(`/api/notifications/${id}/read`);
    btn.classList.remove('bg-accent/5', 'border-l-2', 'border-l-accent');
    btn.querySelector('[aria-label="Unread"]')?.remove();
  } catch {}
};
