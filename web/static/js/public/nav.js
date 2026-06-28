/**
 * nav.js — Nav bar: opaque on white pages, frosted glass on scroll for dark-hero pages.
 */

import { prefersReducedMotion } from '../shared/utils.js';

const nav       = document.getElementById('site-nav');
const mobileBtn = document.getElementById('mobile-menu-btn');
const drawer    = document.getElementById('mobile-drawer');
const closeBtn  = document.getElementById('close-drawer-btn');
const backdrop  = document.getElementById('drawer-backdrop');
const drawerItems = document.querySelectorAll('.drawer-item');

// Homepage has a full-screen dark hero — all other pages are white.
// We detect it by checking for the hero section with bg-primary.
const hasDarkHero = !!document.querySelector('section.bg-primary');

// ─── Nav state helpers ────────────────────────────────────────────────────────

function setScrolled() {
  nav.style.background    = 'rgba(255,255,255,0.95)';
  nav.style.backdropFilter = 'blur(16px)';
  nav.style.WebkitBackdropFilter = 'blur(16px)';
  nav.style.borderBottom  = '1px solid rgba(0,0,0,0.08)';
  nav.style.boxShadow     = '0 4px 24px rgba(0,0,0,0.06)';
  nav.querySelectorAll('.nav-link').forEach(l => { l.style.color = '#64748B'; });
  const logo = nav.querySelector('span.font-display');
  if (logo) logo.style.color = '#0F2557';
  const portal = nav.querySelector('a[href="/login"]');
  if (portal) portal.style.color = '#64748B';
}

function setTransparent() {
  nav.style.background    = 'transparent';
  nav.style.backdropFilter = 'none';
  nav.style.WebkitBackdropFilter = 'none';
  nav.style.borderBottom  = 'none';
  nav.style.boxShadow     = 'none';
  nav.querySelectorAll('.nav-link').forEach(l => { l.style.color = 'rgba(255,255,255,0.8)'; });
  const logo = nav.querySelector('span.font-display');
  if (logo) logo.style.color = 'white';
  const portal = nav.querySelector('a[href="/login"]');
  if (portal) portal.style.color = 'rgba(255,255,255,0.8)';
}

// ─── Scroll handler ───────────────────────────────────────────────────────────

let ticking = false;

function updateNav() {
  if (!hasDarkHero || window.scrollY > 40) {
    setScrolled();
  } else {
    setTransparent();
  }
  ticking = false;
}

window.addEventListener('scroll', () => {
  if (!ticking) { requestAnimationFrame(updateNav); ticking = true; }
}, { passive: true });

updateNav(); // set correct state immediately on load

// ─── Active link ──────────────────────────────────────────────────────────────

const currentPath = window.location.pathname;
document.querySelectorAll('.nav-link').forEach(link => {
  const href = link.getAttribute('href');
  if (href === currentPath || (href !== '/' && currentPath.startsWith(href))) {
    link.style.fontWeight = '600';
  }
});

// ─── Mobile drawer ────────────────────────────────────────────────────────────

function openDrawer() {
  drawer.style.display = 'block';
  drawer.setAttribute('aria-hidden', 'false');
  mobileBtn.setAttribute('aria-expanded', 'true');
  document.body.style.overflow = 'hidden';

  const bd = drawer.querySelector('#drawer-backdrop');
  if (bd) { bd.style.opacity = '0'; bd.style.transition = 'opacity 300ms ease'; requestAnimationFrame(() => { bd.style.opacity = '1'; }); }

  if (!prefersReducedMotion) {
    drawerItems.forEach((item, i) => {
      item.style.transition = `opacity 300ms ease ${i * 60}ms, transform 300ms ease ${i * 60}ms`;
      item.style.transform = 'translateX(-20px)'; item.style.opacity = '0';
      requestAnimationFrame(() => { item.style.opacity = '1'; item.style.transform = 'translateX(0)'; });
    });
  } else {
    drawerItems.forEach(item => { item.style.opacity = '1'; item.style.transform = 'none'; });
  }
  setTimeout(() => drawer.querySelector('a')?.focus(), 100);
}

function closeDrawer() {
  drawer.setAttribute('aria-hidden', 'true');
  mobileBtn.setAttribute('aria-expanded', 'false');
  document.body.style.overflow = '';
  const bd = drawer.querySelector('#drawer-backdrop');
  if (bd) bd.style.opacity = '0';
  setTimeout(() => { drawer.style.display = 'none'; }, 300);
}

mobileBtn?.addEventListener('click', openDrawer);
closeBtn?.addEventListener('click', closeDrawer);
backdrop?.addEventListener('click', closeDrawer);
document.addEventListener('keydown', e => {
  if (e.key === 'Escape' && drawer?.style.display !== 'none') { closeDrawer(); mobileBtn?.focus(); }
});

// ─── Hero entrance animations (homepage only) ─────────────────────────────────

if (!prefersReducedMotion && hasDarkHero) {
  [
    { sel: '.hero-badge',  delay: 200 },
    { sel: '.hero-title',  delay: 400 },
    { sel: '.hero-motto',  delay: 550 },
    { sel: '.hero-desc',   delay: 650 },
    { sel: '.hero-ctas',   delay: 800 },
    { sel: '.hero-scroll', delay: 1000 },
  ].forEach(({ sel, delay }) => {
    const el = document.querySelector(sel);
    if (!el) return;
    el.style.transform  = 'translateY(24px)';
    el.style.transition = `opacity 600ms ease ${delay}ms, transform 600ms ease ${delay}ms`;
    setTimeout(() => { el.style.opacity = '1'; el.style.transform = 'translateY(0)'; }, 50);
  });
}
