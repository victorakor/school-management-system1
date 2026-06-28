/**
 * utils.js — Shared utility functions for date formatting, DOM helpers, and string operations.
 */

// ─── Date Formatting ──────────────────────────────────────────────────────────

/** Format a date string or Date object to "2 January 2025" */
export function formatDate(date) {
  if (!date) return '';
  const d = new Date(date);
  return d.toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric' });
}

/** Format a date string or Date object to "2 Jan 2025, 3:04 PM" */
export function formatDateTime(date) {
  if (!date) return '';
  const d = new Date(date);
  return d.toLocaleDateString('en-GB', {
    day: 'numeric', month: 'short', year: 'numeric',
    hour: 'numeric', minute: '2-digit', hour12: true,
  });
}

/** Format seconds remaining as "MM:SS" */
export function formatCountdown(seconds) {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}

/** Return relative time string: "2 minutes ago", "just now", etc. */
export function timeAgo(date) {
  const seconds = Math.floor((Date.now() - new Date(date)) / 1000);
  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} hr ago`;
  return formatDate(date);
}

// ─── DOM Helpers ──────────────────────────────────────────────────────────────

/** Query selector shorthand */
export const $ = (sel, ctx = document) => ctx.querySelector(sel);
export const $$ = (sel, ctx = document) => [...ctx.querySelectorAll(sel)];

/** Show an element (removes 'hidden' class) */
export function show(el) {
  if (el) el.classList.remove('hidden');
}

/** Hide an element (adds 'hidden' class) */
export function hide(el) {
  if (el) el.classList.add('hidden');
}

/** Toggle visibility */
export function toggle(el) {
  if (el) el.classList.toggle('hidden');
}

/** Create an element with optional classes and text */
export function createElement(tag, classes = '', text = '') {
  const el = document.createElement(tag);
  if (classes) el.className = classes;
  if (text) el.textContent = text;
  return el;
}

/** Inject HTML into an element safely */
export function setHTML(el, html) {
  if (el) el.innerHTML = html;
}

// ─── String Helpers ───────────────────────────────────────────────────────────

/** Truncate a string to maxLen characters */
export function truncate(str, maxLen = 100) {
  if (!str || str.length <= maxLen) return str;
  return str.slice(0, maxLen - 3) + '…';
}

/** Convert snake_case or UPPER_CASE to Title Case */
export function titleCase(str) {
  if (!str) return '';
  return str.toLowerCase().replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

/** Format a number with commas: 1234567 → "1,234,567" */
export function formatNumber(n) {
  return Number(n).toLocaleString('en-NG');
}

/** Format currency: 5000 → "₦5,000.00" */
export function formatCurrency(amount) {
  return new Intl.NumberFormat('en-NG', { style: 'currency', currency: 'NGN' }).format(amount);
}

// ─── Form Helpers ─────────────────────────────────────────────────────────────

/** Collect all form field values as a plain object */
export function formData(form) {
  const data = {};
  new FormData(form).forEach((value, key) => {
    data[key] = value;
  });
  return data;
}

/** Display validation errors on form fields */
export function showFormErrors(errors, form) {
  // Clear previous errors
  form.querySelectorAll('.form-error').forEach(el => {
    el.textContent = '';
    el.classList.add('hidden');
  });
  form.querySelectorAll('input, select, textarea').forEach(el => {
    el.classList.remove('border-danger');
  });

  if (!errors) return;

  Object.entries(errors).forEach(([field, message]) => {
    const input = form.querySelector(`[name="${field}"]`);
    if (input) {
      input.classList.add('border-danger');
      const errorEl = input.closest('.form-group')?.querySelector('.form-error');
      if (errorEl) {
        errorEl.textContent = message;
        errorEl.classList.remove('hidden');
      }
    }
  });
}

/** Clear all form errors */
export function clearFormErrors(form) {
  form.querySelectorAll('.form-error').forEach(el => {
    el.textContent = '';
    el.classList.add('hidden');
  });
  form.querySelectorAll('.border-danger').forEach(el => {
    el.classList.remove('border-danger');
  });
}

// ─── Animation Helpers ────────────────────────────────────────────────────────

/** Check if user prefers reduced motion */
export const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

/** Fade in an element */
export function fadeIn(el, duration = 300) {
  if (!el) return;
  if (prefersReducedMotion) {
    el.style.opacity = '1';
    return;
  }
  el.style.opacity = '0';
  el.style.transition = `opacity ${duration}ms ease`;
  requestAnimationFrame(() => {
    el.style.opacity = '1';
  });
}

/** Stagger animate a list of elements */
export function staggerIn(elements, delay = 60, duration = 300) {
  if (prefersReducedMotion) {
    elements.forEach(el => { el.style.opacity = '1'; el.style.transform = 'none'; });
    return;
  }
  elements.forEach((el, i) => {
    el.style.opacity = '0';
    el.style.transform = 'translateY(16px)';
    el.style.transition = `opacity ${duration}ms ease, transform ${duration}ms ease`;
    setTimeout(() => {
      el.style.opacity = '1';
      el.style.transform = 'translateY(0)';
    }, i * delay);
  });
}

// ─── Intersection Observer Helper ─────────────────────────────────────────────

/** Observe elements and call callback when they enter the viewport */
export function onVisible(elements, callback, options = {}) {
  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        callback(entry.target);
        observer.unobserve(entry.target);
      }
    });
  }, { threshold: 0.1, ...options });

  // Accept NodeList, Array, or single element — filter out nulls
  const els = Array.from(
    Array.isArray(elements) || elements instanceof NodeList ? elements : [elements]
  ).filter(Boolean);
  els.forEach(el => observer.observe(el));
  return observer;
}

// ─── Toast Notifications ──────────────────────────────────────────────────────

export function showToast(message, type = 'info') {
  const colors = {
    success: 'bg-success text-white',
    danger:  'bg-danger text-white',
    warning: 'bg-warning text-white',
    info:    'bg-accent text-white',
  };
  const toast = document.createElement('div');
  toast.className = `fixed bottom-6 right-6 z-[200] px-5 py-3 rounded-xl shadow-lg text-sm font-medium ${colors[type] || colors.info} transition-all duration-300`;
  toast.textContent = message;
  document.body.appendChild(toast);
  // Animate in
  requestAnimationFrame(() => { toast.style.opacity = '1'; toast.style.transform = 'translateY(0)'; });
  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateY(8px)';
    setTimeout(() => toast.remove(), 300);
  }, 3500);
}

// ─── Skeleton Helpers ─────────────────────────────────────────────────────────

export function showSkeleton(container, count = 4, heightClass = 'h-28') {
  if (!container) return;
  container.innerHTML = Array(count).fill(
    `<div class="skeleton ${heightClass} rounded-2xl"></div>`
  ).join('');
}

export function hideSkeleton(container) {
  if (!container) container.innerHTML = '';
}
