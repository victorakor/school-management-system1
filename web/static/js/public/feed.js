/**
 * feed.js — Masonry grid, infinite scroll, and lightbox for the activity feed.
 */

import { api } from '../shared/api.js';
import { $, $$, onVisible, prefersReducedMotion } from '../shared/utils.js';

const grid = $('#feed-grid');
const filterBtns = $$('.feed-filter-btn');
const lightbox = $('#feed-lightbox');
const lightboxMedia = $('#lightbox-media');
const lightboxCaption = $('#lightbox-caption');
const lightboxClose = $('#lightbox-close');
const lightboxPrev = $('#lightbox-prev');
const lightboxNext = $('#lightbox-next');
const sentinel = $('#feed-sentinel');

let currentPage = 1;
let currentFilter = '';
let isLoading = false;
let hasMore = true;
let currentLightboxItems = [];
let currentLightboxIndex = 0;

// ─── Filter buttons ───────────────────────────────────────────────────────────

filterBtns.forEach(btn => {
  btn.addEventListener('click', () => {
    filterBtns.forEach(b => b.classList.remove('active', 'bg-accent', 'text-white'));
    filterBtns.forEach(b => b.classList.add('bg-white', 'text-text-secondary'));
    btn.classList.add('active', 'bg-accent', 'text-white');
    btn.classList.remove('bg-white', 'text-text-secondary');

    currentFilter = btn.dataset.division || '';
    currentPage = 1;
    hasMore = true;
    if (grid) grid.innerHTML = '';
    loadPosts();
  });
});

// ─── Load posts ───────────────────────────────────────────────────────────────

async function loadPosts() {
  if (isLoading || !hasMore) return;
  isLoading = true;

  const params = { page: currentPage };
  if (currentFilter) params.division = currentFilter;

  try {
    const data = await api.get('/api/feed', params);
    const posts = Array.isArray(data) ? data : (data?.data ?? []);
    hasMore = data?.has_more ?? false;

    posts.forEach(post => {
      const card = createPostCard(post);
      grid?.appendChild(card);
    });

    // Animate new cards in
    const newCards = grid?.querySelectorAll('.feed-card:not(.animated)') ?? [];
    if (!prefersReducedMotion) {
      newCards.forEach((card, i) => {
        card.style.opacity = '0';
        card.style.transform = 'translateY(16px)';
        card.style.transition = `opacity 400ms ease ${i * 60}ms, transform 400ms ease ${i * 60}ms`;
        requestAnimationFrame(() => {
          card.style.opacity = '1';
          card.style.transform = 'translateY(0)';
        });
        card.classList.add('animated');
      });
    } else {
      newCards.forEach(card => card.classList.add('animated'));
    }

    currentPage++;
  } catch (err) {
    console.error('Failed to load feed:', err);
  } finally {
    isLoading = false;
  }
}

function createPostCard(post) {
  const card = document.createElement('article');
  card.className = 'feed-card break-inside-avoid mb-4 bg-white rounded-2xl overflow-hidden border border-neutral-100 shadow-card cursor-pointer hover:shadow-lg transition-all duration-200 group';
  card.setAttribute('role', 'button');
  card.setAttribute('tabindex', '0');
  card.setAttribute('aria-label', `View post: ${post.caption || 'School activity'}`);

  const firstMedia = post.media_urls?.[0];
  const isVideo = firstMedia?.type === 'VIDEO';

  card.innerHTML = `
    ${firstMedia ? `
    <div class="relative overflow-hidden">
      ${isVideo
        ? `<video src="${firstMedia.url}" class="w-full object-cover group-hover:scale-105 transition-transform duration-500" muted playsinline preload="metadata" aria-label="Activity video"></video>`
        : `<img src="${firstMedia.url}" alt="School activity" class="w-full object-cover group-hover:scale-105 transition-transform duration-500" loading="lazy" />`
      }
      ${post.media_urls?.length > 1 ? `<span class="absolute top-2 right-2 bg-black/60 text-white text-xs font-medium px-2 py-1 rounded-full">+${post.media_urls.length - 1}</span>` : ''}
      ${isVideo ? `<div class="absolute inset-0 flex items-center justify-center"><div class="w-12 h-12 bg-white/90 rounded-full flex items-center justify-center shadow-lg"><svg class="w-5 h-5 text-primary ml-0.5" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path d="M8 5v14l11-7z"/></svg></div></div>` : ''}
    </div>` : ''}
    <div class="p-4">
      <span class="inline-block bg-accent/10 text-accent text-xs font-semibold px-2.5 py-1 rounded-full mb-2">${post.division_tag}</span>
      ${post.caption ? `<p class="text-text-secondary text-sm leading-relaxed">${post.caption}</p>` : ''}
    </div>`;

  // Open lightbox on click/enter
  const openLightbox = () => {
    if (post.media_urls?.length > 0) {
      currentLightboxItems = post.media_urls;
      currentLightboxIndex = 0;
      showLightboxItem(0, post.caption);
    }
  };

  card.addEventListener('click', openLightbox);
  card.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      openLightbox();
    }
  });

  return card;
}

// ─── Lightbox ─────────────────────────────────────────────────────────────────

function showLightboxItem(index, caption) {
  if (!lightbox || !lightboxMedia) return;

  const item = currentLightboxItems[index];
  if (!item) return;

  currentLightboxIndex = index;

  if (item.type === 'VIDEO') {
    lightboxMedia.innerHTML = `<video src="${item.url}" class="max-w-full max-h-[80vh] rounded-xl" controls autoplay muted playsinline aria-label="Activity video"></video>`;
  } else {
    lightboxMedia.innerHTML = `<img src="${item.url}" alt="School activity" class="max-w-full max-h-[80vh] rounded-xl object-contain" />`;
  }

  if (lightboxCaption) {
    lightboxCaption.textContent = caption || '';
  }

  // Show/hide nav arrows
  if (lightboxPrev) lightboxPrev.style.display = index > 0 ? 'flex' : 'none';
  if (lightboxNext) lightboxNext.style.display = index < currentLightboxItems.length - 1 ? 'flex' : 'none';

  // Show lightbox
  lightbox.classList.remove('hidden');
  lightbox.style.opacity = '0';
  lightbox.style.transition = 'opacity 200ms ease';
  requestAnimationFrame(() => { lightbox.style.opacity = '1'; });
  document.body.style.overflow = 'hidden';

  // Focus close button
  lightboxClose?.focus();
}

function closeLightbox() {
  if (!lightbox) return;
  lightbox.style.opacity = '0';
  setTimeout(() => {
    lightbox.classList.add('hidden');
    lightboxMedia.innerHTML = '';
    document.body.style.overflow = '';
  }, 200);
}

lightboxClose?.addEventListener('click', closeLightbox);
lightbox?.addEventListener('click', (e) => {
  if (e.target === lightbox) closeLightbox();
});

lightboxPrev?.addEventListener('click', () => {
  if (currentLightboxIndex > 0) {
    showLightboxItem(currentLightboxIndex - 1, lightboxCaption?.textContent);
  }
});

lightboxNext?.addEventListener('click', () => {
  if (currentLightboxIndex < currentLightboxItems.length - 1) {
    showLightboxItem(currentLightboxIndex + 1, lightboxCaption?.textContent);
  }
});

document.addEventListener('keydown', (e) => {
  if (lightbox?.classList.contains('hidden')) return;
  if (e.key === 'Escape') closeLightbox();
  if (e.key === 'ArrowLeft') lightboxPrev?.click();
  if (e.key === 'ArrowRight') lightboxNext?.click();
});

// ─── Infinite scroll ──────────────────────────────────────────────────────────

if (sentinel) {
  const observer = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && hasMore && !isLoading) {
      loadPosts();
    }
  }, { threshold: 0.1 });
  observer.observe(sentinel);
}

// Initial load
loadPosts();
