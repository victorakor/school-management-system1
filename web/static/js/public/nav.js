/**
 * nav.js — Frosted glass nav on scroll, mobile drawer with GSAP stagger animations.
 */

import { prefersReducedMotion } from '../shared/utils.js';

const nav = document.getElementById('site-nav');
const mobileBtn = document.getElementById('mobile-menu-btn');
const drawer = document.getElementById('mobile-drawer');
const closeBtn = document.getElementById('close-drawer-btn');
const backdrop = document.getElementById('drawer-backdrop');
const drawerItems = document.querySelectorAll('.drawer-item');

// ─── Detect if page has a dark hero (homepage) ────────────────────────────────
// If the first section is dark (bg-primary), start nav transparent.
// On all other pages (white background), start nav opaque immediately.
const hasDarkHero = !!document.querySelector('section.bg-primary, section[class*="bg-primary"]');

function setNavScrolled() {
  nav.style.background = 'rgba(255,255,255,0.95)';
  nav.style.backdropFilter = 'blur(16px)';
  nav.style.WebkitBackdropFilter = 'blur(16px)';
  nav.style.borderBottom = '1px solid rgba(0,0,0,0.08)';
  nav.style.boxShadow = '0 4px 24px rgba(0,0,0,0.06)';
  nav.querySelectorAll('.nav-link').forEach(link => {
    link.style.color = 'var(--color-text-secondary, #64748B)';
  });
  // Logo text
  const logoText = nav.querySelector('span.font-display');
  if (logoText) logoText.style.color = 'var(--color-primary, #0F2557)';
  // Portal/Apply buttons
  const portalLink = nav.querySelector('a[href="/login"]');
  if (portalLink) portalLink.style.color = 'var(--color-text-secondary, #64748B)';
}

function setNavTransparent() {
  nav.style.background = 'transparent';
  nav.style.backdropFilter = 'none';
  nav.style.WebkitBackdropFilter = 'none';
  nav.style.borderBottom = 'none';
  nav.style.boxShadow = 'none';
  nav.querySelectorAll('.nav-link').forEach(link => {
    link.style.color = 'rgba(255,255,255,0.8)';
  });
  const logoText = nav.querySelector('span.font-display');
  if (logoText) logoText.style.color = 'white';
  const portalLink = nav.querySelector('a[href="/login"]');
  if (portalLink) portalLink.style.color = 'rgba(255,255,255,0.8)';
}

// ─── Scroll handler ───────────────────────────────────────────────────────────
let ticking = false;

function updateNav() {
  const scrolled = window.scrollY > 40;

  if (!hasDarkHero || scrolled) {
    setNavScrolled();
  } else {
    setNavTransparent();
  }

  ticking = false;
}

window.addEventListener('scroll', () => {
  if (!ticking) {
    requestAnimationFrame(updateNav);
    ticking = true;
  }
}, { passive: true });

// Initial state
updateNav();

// ─── Active link indicator ────────────────────────────────────────────────────
const currentPath = window.location.pathname;
document.querySelectorAll('.nav-link').forEach(link => {
  const href = link.getAttribute('href');
  if (href === currentPath || (href !== '/' && currentPath.startsWith(href))) {
    link.classList.add('nav-active');
    link.style.fontWeight = '600';
  }
});

// ─── Mobile drawer ────────────────────────────────────────────────────────────

function openDrawer() {
  drawer.style.display = 'block';
  drawer.setAttribute('aria-hidden', 'false');
  mobileBtn.setAttribute('aria-expanded', 'true');
  document.body.style.overflow = 'hidden';

  const backdropEl = drawer.querySelector('#drawer-backdrop');
  if (backdropEl) {
    backdropEl.style.opacity = '0';
    backdropEl.style.transition = 'opacity 300ms ease';
    requestAnimationFrame(() => { backdropEl.style.opacity = '1'; });
  }

  if (!prefersReducedMotion) {
    drawerItems.forEach((item, i) => {
      item.style.transition = `opacity 300ms ease ${i * 60}ms, transform 300ms ease ${i * 60}ms`;
      item.style.transform = 'translateX(-20px)';
      item.style.opacity = '0';
      requestAnimationFrame(() => {
        item.style.opacity = '1';
        item.style.transform = 'translateX(0)';
      });
    });
  } else {
    drawerItems.forEach(item => {
      item.style.opacity = '1';
      item.style.transform = 'none';
    });
  }

  setTimeout(() => { drawer.querySelector('a')?.focus(); }, 100);
}

function closeDrawer() {
  drawer.setAttribute('aria-hidden', 'true');
  mobileBtn.setAttribute('aria-expanded', 'false');
  document.body.style.overflow = '';

  const backdropEl = drawer.querySelector('#drawer-backdrop');
  if (backdropEl) { backdropEl.style.opacity = '0'; }
  setTimeout(() => { drawer.style.display = 'none'; }, 300);
}

mobileBtn?.addEventListener('click', openDrawer);
closeBtn?.addEventListener('click', closeDrawer);
backdrop?.addEventListener('click', closeDrawer);

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && drawer?.style.display !== 'none') {
    closeDrawer();
    mobileBtn?.focus();
  }
});

// ─── Hero entrance animations ─────────────────────────────────────────────────
if (!prefersReducedMotion && hasDarkHero) {
  const heroElements = [
    { sel: '.hero-badge', delay: 200 },
    { sel: '.hero-title', delay: 400 },
    { sel: '.hero-motto', delay: 550 },
    { sel: '.hero-desc', delay: 650 },
    { sel: '.hero-ctas', delay: 800 },
    { sel: '.hero-scroll', delay: 1000 },
  ];

  heroElements.forEach(({ sel, delay }) => {
    const el = document.querySelector(sel);
    if (!el) return;
    el.style.transform = 'translateY(24px)';
    el.style.transition = `opacity 600ms ease ${delay}ms, transform 600ms ease ${delay}ms`;
    setTimeout(() => {
      el.style.opacity = '1';
      el.style.transform = 'translateY(0)';
    }, 50);
  });
}

// ─── Frosted glass on scroll ──────────────────────────────────────────────────

let lastScrollY = 0;
let ticking = false;

function updateNav() {
  const scrolled = window.scrollY > 40;

  if (scrolled) {
    nav.classList.add('nav-scrolled');
    nav.style.background = 'rgba(255,255,255,0.85)';
    nav.style.backdropFilter = 'blur(16px)';
    nav.style.WebkitBackdropFilter = 'blur(16px)';
    nav.style.borderBottom = '1px solid rgba(0,0,0,0.06)';
    nav.style.boxShadow = '0 4px 24px rgba(0,0,0,0.06)';
    // Switch text colors for scrolled state
    nav.querySelectorAll('.nav-link').forEach(link => {
      link.style.color = '';
    });
  } else {
    nav.classList.remove('nav-scrolled');
    nav.style.background = 'transparent';
    nav.style.backdropFilter = 'none';
    nav.style.WebkitBackdropFilter = 'none';
    nav.style.borderBottom = 'none';
    nav.style.boxShadow = 'none';
  }

  ticking = false;
}

window.addEventListener('scroll', () => {
  lastScrollY = window.scrollY;
  if (!ticking) {
    requestAnimationFrame(updateNav);
    ticking = true;
  }
}, { passive: true });

// Initial state
updateNav();

// ─── Active link indicator ────────────────────────────────────────────────────

const currentPath = window.location.pathname;
document.querySelectorAll('.nav-link').forEach(link => {
  const href = link.getAttribute('href');
  if (href === currentPath || (href !== '/' && currentPath.startsWith(href))) {
    link.classList.add('nav-active');
    link.style.fontWeight = '600';
  }
});

// ─── Mobile drawer ────────────────────────────────────────────────────────────

function openDrawer() {
  drawer.style.display = 'block';
  drawer.setAttribute('aria-hidden', 'false');
  mobileBtn.setAttribute('aria-expanded', 'true');
  document.body.style.overflow = 'hidden';

  // Animate backdrop
  const backdropEl = drawer.querySelector('#drawer-backdrop');
  if (backdropEl) {
    backdropEl.style.opacity = '0';
    backdropEl.style.transition = 'opacity 300ms ease';
    requestAnimationFrame(() => { backdropEl.style.opacity = '1'; });
  }

  // Stagger drawer items
  if (!prefersReducedMotion) {
    drawerItems.forEach((item, i) => {
      item.style.transition = `opacity 300ms ease ${i * 60}ms, transform 300ms ease ${i * 60}ms`;
      item.style.transform = 'translateX(-20px)';
      item.style.opacity = '0';
      requestAnimationFrame(() => {
        item.style.opacity = '1';
        item.style.transform = 'translateX(0)';
      });
    });
  } else {
    drawerItems.forEach(item => {
      item.style.opacity = '1';
      item.style.transform = 'none';
    });
  }

  // Focus first link
  setTimeout(() => {
    drawer.querySelector('a')?.focus();
  }, 100);
}

function closeDrawer() {
  drawer.setAttribute('aria-hidden', 'true');
  mobileBtn.setAttribute('aria-expanded', 'false');
  document.body.style.overflow = '';

  const backdropEl = drawer.querySelector('#drawer-backdrop');
  if (backdropEl) {
    backdropEl.style.opacity = '0';
  }

  setTimeout(() => {
    drawer.style.display = 'none';
  }, 300);
}

mobileBtn?.addEventListener('click', openDrawer);
closeBtn?.addEventListener('click', closeDrawer);
backdrop?.addEventListener('click', closeDrawer);

// Close on Escape key
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && drawer?.style.display !== 'none') {
    closeDrawer();
    mobileBtn?.focus();
  }
});

// ─── Hero entrance animations ─────────────────────────────────────────────────

if (!prefersReducedMotion) {
  const heroElements = [
    { sel: '.hero-badge', delay: 200 },
    { sel: '.hero-title', delay: 400 },
    { sel: '.hero-motto', delay: 550 },
    { sel: '.hero-desc', delay: 650 },
    { sel: '.hero-ctas', delay: 800 },
    { sel: '.hero-scroll', delay: 1000 },
  ];

  heroElements.forEach(({ sel, delay }) => {
    const el = document.querySelector(sel);
    if (!el) return;
    el.style.transform = 'translateY(24px)';
    el.style.transition = `opacity 600ms ease ${delay}ms, transform 600ms ease ${delay}ms`;
    setTimeout(() => {
      el.style.opacity = '1';
      el.style.transform = 'translateY(0)';
    }, 50);
  });
}
