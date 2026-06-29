/**
 * feed-admin.js — Activity feed management for admin roles.
 *
 * Role capabilities per RBAC spec:
 *   OWNER / PRINCIPAL / VP / HEAD / ASST_HEAD / BLOG_MANAGER — can create, edit, archive posts
 *   All other authenticated roles — read-only view of published feed
 */
import { api } from '../shared/api.js';
import { formatDate, showToast } from '../shared/utils.js';

// Roles that can create / edit / delete posts
const CAN_POST = [
  'OWNER', 'PRINCIPAL', 'VICE_PRINCIPAL', 'HEAD_TEACHER', 'ASST_HEAD_TEACHER',
  'BLOG_MANAGER',
];

// Module-level user reference for inner functions.
let _feedUser = null;

export async function initFeedAdmin(container, user) {
  _feedUser = user;
  const canPost = user && CAN_POST.includes(user.role);

  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Activity Feed</h1>
      ${canPost ? '<button id="new-post-btn" class="btn btn-primary">New Post</button>' : ''}
    </div>
    <div id="feed-admin-list" class="grid grid-cols-1 lg:grid-cols-2 gap-4"></div>
    <div id="post-modal" class="hidden"></div>
  `;

  if (canPost) {
    document.getElementById('new-post-btn').addEventListener('click', openPostModal);
  }
  await loadPosts();
}

async function loadPosts() {
  const list = document.getElementById('feed-admin-list');
  list.innerHTML = Array(4).fill(0).map(() =>
    `<div class="card p-4"><div class="skeleton h-40 rounded mb-3"></div><div class="skeleton h-4 w-3/4 rounded"></div></div>`
  ).join('');

  try {
    const posts = await api.get('/api/feed?include_archived=true');
    if (!posts.length) {
      list.innerHTML = `<div class="col-span-2 card p-12 text-center text-secondary">No posts yet. Create your first post!</div>`;
      return;
    }
    list.innerHTML = posts.map(post => `
      <div class="card overflow-hidden ${post.is_archived ? 'opacity-50' : ''}">
        ${post.media_urls?.length ? `
          <div class="aspect-video bg-neutral-100 overflow-hidden">
            ${post.media_urls[0].type === 'VIDEO'
              ? `<video src="${post.media_urls[0].url}" class="w-full h-full object-cover" muted></video>`
              : `<img src="${post.media_urls[0].url}" class="w-full h-full object-cover" alt="">`}
          </div>` : ''}
        <div class="p-4">
          <div class="flex items-start justify-between gap-2 mb-2">
            <p class="text-sm text-primary line-clamp-2">${post.caption || 'No caption'}</p>
            <span class="badge badge-${post.is_published ? 'success' : 'warning'} flex-shrink-0">
              ${post.is_published ? 'Published' : 'Draft'}
            </span>
          </div>
          <div class="flex items-center justify-between mt-3">
            <div class="flex items-center gap-2">
              <span class="badge badge-neutral text-xs">${post.division_tag}</span>
              <span class="text-xs text-secondary">${formatDate(post.created_at)}</span>
            </div>
            <div class="flex gap-1">
              ${_feedUser && CAN_POST.includes(_feedUser.role) ? `
                <button class="btn btn-outline btn-sm" onclick="window._editPost('${post.id}')">Edit</button>
                <button class="btn btn-${post.is_archived ? 'success' : 'danger'} btn-sm"
                        onclick="window._toggleArchivePost('${post.id}', ${post.is_archived})">
                  ${post.is_archived ? 'Restore' : 'Archive'}
                </button>
              ` : ''}
            </div>
          </div>
        </div>
      </div>`).join('');

    window._editPost = (id) => openPostModal(id);
    window._toggleArchivePost = async (id, isArchived) => {
      try {
        if (isArchived) {
          await api.put(`/api/feed/${id}`, { is_archived: false });
        } else {
          await api.delete(`/api/feed/${id}`);
        }
        showToast(isArchived ? 'Post restored' : 'Post archived', 'success');
        loadPosts();
      } catch {
        showToast('Action failed', 'error');
      }
    };
  } catch {
    list.innerHTML = `<div class="col-span-2 card p-6 text-danger text-sm">Failed to load posts.</div>`;
  }
}

function openPostModal(editID = null) {
  const modal = document.getElementById('post-modal');
  modal.className = 'fixed inset-0 z-50 flex items-center justify-center p-4';
  modal.innerHTML = `
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" onclick="window._closePostModal()"></div>
    <div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-lg z-10 p-6 max-h-[90vh] overflow-y-auto">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold text-primary">${editID ? 'Edit Post' : 'New Post'}</h2>
        <button onclick="window._closePostModal()" class="text-secondary hover:text-primary">✕</button>
      </div>
      <form id="post-form" class="space-y-4">
        <div>
          <label class="label block mb-1">Caption</label>
          <textarea id="post-caption" class="input w-full" rows="3" placeholder="Write a caption..."></textarea>
        </div>
        <div>
          <label class="label block mb-1">Division Tag</label>
          <select id="post-division" class="input w-full">
            <option value="SCHOOL">School-wide</option>
            <option value="NURSERY">Nursery</option>
            <option value="PRIMARY">Primary</option>
            <option value="SECONDARY">Secondary</option>
          </select>
        </div>
        <div>
          <label class="label block mb-1">Media (Photos/Videos)</label>
          <div id="post-dropzone" class="border-2 border-dashed border-neutral-200 rounded-xl p-6 text-center cursor-pointer hover:border-accent transition-colors">
            <p class="text-secondary text-sm">Click to upload photos or videos</p>
            <input type="file" id="post-file-input" class="hidden" accept="image/*,video/mp4" multiple>
          </div>
          <div id="post-media-preview" class="grid grid-cols-3 gap-2 mt-2"></div>
        </div>
        <div class="flex items-center gap-2">
          <input type="checkbox" id="post-published" class="w-4 h-4">
          <label for="post-published" class="text-sm">Publish immediately</label>
        </div>
        <div id="post-error" class="text-danger text-sm hidden"></div>
        <div class="flex gap-3 pt-2">
          <button type="submit" class="btn btn-primary flex-1">
            ${editID ? 'Update Post' : 'Create Post'}
          </button>
          <button type="button" onclick="window._closePostModal()" class="btn btn-outline flex-1">Cancel</button>
        </div>
      </form>
    </div>`;

  window._closePostModal = () => {
    modal.className = 'hidden';
    modal.innerHTML = '';
  };

  const dropzone = document.getElementById('post-dropzone');
  const fileInput = document.getElementById('post-file-input');
  const mediaURLs = [];

  dropzone.addEventListener('click', () => fileInput.click());
  fileInput.addEventListener('change', async () => {
    const files = Array.from(fileInput.files);
    for (const file of files) {
      try {
        const sig = await api.post('/api/upload/sign', {
          folder: 'feed',
          asset_type: file.type.startsWith('video') ? 'video' : 'image',
        });
        const formData = new FormData();
        formData.append('file', file);
        formData.append('api_key', sig.api_key);
        formData.append('timestamp', sig.timestamp);
        formData.append('signature', sig.signature);
        formData.append('folder', sig.folder);
        const resourceType = file.type.startsWith('video') ? 'video' : 'image';
        const res = await fetch(`https://api.cloudinary.com/v1_1/${sig.cloud_name}/${resourceType}/upload`, {
          method: 'POST', body: formData,
        });
        const data = await res.json();
        mediaURLs.push({ url: data.secure_url, type: file.type.startsWith('video') ? 'VIDEO' : 'IMAGE' });
        const preview = document.getElementById('post-media-preview');
        const thumb = document.createElement('div');
        thumb.className = 'aspect-square rounded-lg overflow-hidden bg-neutral-100';
        thumb.innerHTML = resourceType === 'video'
          ? `<video src="${data.secure_url}" class="w-full h-full object-cover" muted></video>`
          : `<img src="${data.secure_url}" class="w-full h-full object-cover" alt="">`;
        preview.appendChild(thumb);
      } catch {
        showToast('Failed to upload file', 'error');
      }
    }
  });

  document.getElementById('post-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const caption = document.getElementById('post-caption').value;
    const division = document.getElementById('post-division').value;
    const isPublished = document.getElementById('post-published').checked;
    const errEl = document.getElementById('post-error');

    try {
      const payload = { caption, division_tag: division, is_published: isPublished, media_urls: mediaURLs };
      if (editID) {
        await api.put(`/api/feed/${editID}`, payload);
        showToast('Post updated', 'success');
      } else {
        await api.post('/api/feed', payload);
        showToast('Post created', 'success');
      }
      window._closePostModal();
      loadPosts();
    } catch (err) {
      errEl.textContent = err.message || 'Failed to save post';
      errEl.classList.remove('hidden');
    }
  });
}
