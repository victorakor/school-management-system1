/**
 * counters.js — Count-up animation on scroll using Intersection Observer.
 * Respects prefers-reduced-motion.
 */

import { prefersReducedMotion, onVisible } from '../shared/utils.js';

const counters = document.querySelectorAll('.counter[data-target]');

if (counters.length > 0) {
  onVisible(counters, (el) => {
    const target = parseInt(el.dataset.target, 10);
    if (isNaN(target)) return;

    if (prefersReducedMotion) {
      el.textContent = target.toLocaleString('en-NG');
      return;
    }

    animateCounter(el, target);
  });
}

function animateCounter(el, target) {
  const duration = 1800; // ms
  const start = performance.now();
  const startValue = 0;

  function update(now) {
    const elapsed = now - start;
    const progress = Math.min(elapsed / duration, 1);
    // Ease out cubic
    const eased = 1 - Math.pow(1 - progress, 3);
    const current = Math.round(startValue + (target - startValue) * eased);
    el.textContent = current.toLocaleString('en-NG');

    if (progress < 1) {
      requestAnimationFrame(update);
    } else {
      el.textContent = target.toLocaleString('en-NG');
    }
  }

  requestAnimationFrame(update);
}
