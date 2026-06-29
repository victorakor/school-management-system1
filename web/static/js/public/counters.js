/**
 * counters.js — Count-up animation on scroll using Intersection Observer.
 * Respects prefers-reduced-motion.
 * Supports optional data-suffix attribute (e.g. data-suffix="+") appended after the number.
 */

import { prefersReducedMotion, onVisible } from '../shared/utils.js';

const counters = document.querySelectorAll('.counter[data-target]');

if (counters.length > 0) {
  onVisible(counters, (el) => {
    const target = parseInt(el.dataset.target, 10);
    if (isNaN(target)) return;

    const suffix = el.dataset.suffix || '';

    if (prefersReducedMotion) {
      el.textContent = target.toLocaleString('en-NG') + suffix;
      return;
    }

    animateCounter(el, target, suffix);
  });
}

function animateCounter(el, target, suffix = '') {
  const duration = 1800; // ms
  const start = performance.now();

  function update(now) {
    const elapsed = now - start;
    const progress = Math.min(elapsed / duration, 1);
    // Ease out cubic
    const eased = 1 - Math.pow(1 - progress, 3);
    const current = Math.round(target * eased);
    el.textContent = current.toLocaleString('en-NG') + suffix;

    if (progress < 1) {
      requestAnimationFrame(update);
    } else {
      el.textContent = target.toLocaleString('en-NG') + suffix;
    }
  }

  requestAnimationFrame(update);
}
