/**
 * finance.js — Financial management portal module
 */
import { api } from '../shared/api.js';
import { formatDate, formatCurrency, showToast } from '../shared/utils.js';

export async function initFinance(container, user) {
  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Finance</h1>
      <button id="record-payment-btn" class="btn btn-primary">Record Payment</button>
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
      <select id="fin-division" class="input"><option value="">Select Division</option>
        <option value="NURSERY">Nursery</option>
        <option value="PRIMARY">Primary</option>
        <option value="SECONDARY">Secondary</option>
      </select>
      <select id="fin-session" class="input"><option value="">Select Session</option></select>
      <select id="fin-term" class="input"><option value="">Select Term</option></select>
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div id="fee-structure-panel" class="card p-5">
        <h3 class="font-semibold text-primary mb-4">Fee Structure</h3>
        <div id="fee-structure-content" class="text-secondary text-sm">Select division, session, and term to view fee structure.</div>
      </div>
      <div id="recent-payments-panel" class="card p-5">
        <h3 class="font-semibold text-primary mb-4">Recent Payments</h3>
        <div id="recent-payments-content" class="text-secondary text-sm">No payments recorded yet.</div>
      </div>
    </div>
    <div id="payment-modal" class="hidden"></div>
  `;

  await loadSessions();
  document.getElementById('fin-session').addEventListener('change', onSessionChange);
  document.getElementById('fin-term').addEventListener('change', loadFeeStructure);
  document.getElementById('fin-division').addEventListener('change', loadFeeStructure);
  document.getElementById('record-payment-btn').addEventListener('click', openPaymentModal);
}

async function loadSessions() {
  try {
    const sessions = await api.get('/api/sessions');
    const sel = document.getElementById('fin-session');
    sessions.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.id;
      opt.textContent = s.name;
      if (s.is_active) opt.selected = true;
      sel.appendChild(opt);
    });
    if (sel.value) await onSessionChange();
  } catch {}
}

async function onSessionChange() {
  const sessionID = document.getElementById('fin-session').value;
  if (!sessionID) return;
  try {
    const terms = await api.get(`/api/sessions/${sessionID}/terms`);
    const sel = document.getElementById('fin-term');
    sel.innerHTML = '<option value="">Select Term</option>';
    terms.forEach(t => {
      const opt = document.createElement('option');
      opt.value = t.id;
      opt.textContent = t.name;
      sel.appendChild(opt);
    });
  } catch {}
}

async function loadFeeStructure() {
  const divisionID = document.getElementById('fin-division').value;
  const sessionID = document.getElementById('fin-session').value;
  const termID = document.getElementById('fin-term').value;
  const content = document.getElementById('fee-structure-content');

  if (!divisionID || !sessionID || !termID) return;

  content.innerHTML = `<div class="skeleton h-32 rounded"></div>`;

  try {
    const structure = await api.get(`/api/finance/structure?division_id=${divisionID}&session_id=${sessionID}&term_id=${termID}`);
    if (!structure || !structure.categories?.length) {
      content.innerHTML = `<p class="text-secondary text-sm">No fee structure set for this period.</p>
        <button class="btn btn-outline btn-sm mt-3" onclick="window._openFeeStructureForm()">Set Fee Structure</button>`;
      return;
    }
    content.innerHTML = `
      <div class="space-y-2">
        ${structure.categories.map(cat => `
          <div class="flex items-center justify-between py-2 border-b border-neutral-100 last:border-0">
            <span class="text-sm font-medium">${cat.name}</span>
            <span class="text-sm text-secondary">Variable by class</span>
          </div>`).join('')}
      </div>`;
  } catch {
    content.innerHTML = `<p class="text-secondary text-sm">No fee structure found.</p>`;
  }
}

function openPaymentModal() {
  const modal = document.getElementById('payment-modal');
  modal.className = 'fixed inset-0 z-50 flex items-center justify-center p-4';
  modal.innerHTML = `
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" onclick="window._closePaymentModal()"></div>
    <div class="relative bg-white rounded-2xl shadow-2xl w-full max-w-lg z-10 p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold text-primary">Record Payment</h2>
        <button onclick="window._closePaymentModal()" class="text-secondary hover:text-primary">✕</button>
      </div>
      <form id="payment-form" class="space-y-4">
        <div>
          <label class="label block mb-1">Student ID or Name</label>
          <input type="text" id="pay-student-search" class="input w-full" placeholder="Search student...">
          <input type="hidden" id="pay-student-id">
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="label block mb-1">Fee Category</label>
            <input type="text" id="pay-category" class="input w-full" placeholder="e.g. Tuition">
          </div>
          <div>
            <label class="label block mb-1">Amount Owed (₦)</label>
            <input type="number" id="pay-owed" class="input w-full" min="0" step="0.01">
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="label block mb-1">Amount Paid (₦)</label>
            <input type="number" id="pay-amount" class="input w-full" min="0" step="0.01">
          </div>
          <div>
            <label class="label block mb-1">Payment Date</label>
            <input type="date" id="pay-date" class="input w-full" value="${new Date().toISOString().split('T')[0]}">
          </div>
        </div>
        <div>
          <label class="label block mb-1">Notes (optional)</label>
          <textarea id="pay-notes" class="input w-full" rows="2"></textarea>
        </div>
        <div id="pay-error" class="text-danger text-sm hidden"></div>
        <div class="flex gap-3 pt-2">
          <button type="submit" class="btn btn-primary flex-1">Record Payment</button>
          <button type="button" onclick="window._closePaymentModal()" class="btn btn-outline flex-1">Cancel</button>
        </div>
      </form>
    </div>`;

  window._closePaymentModal = () => {
    modal.className = 'hidden';
    modal.innerHTML = '';
  };

  document.getElementById('payment-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const studentID = document.getElementById('pay-student-id').value;
    const category = document.getElementById('pay-category').value;
    const owed = parseFloat(document.getElementById('pay-owed').value);
    const paid = parseFloat(document.getElementById('pay-amount').value);
    const date = document.getElementById('pay-date').value;
    const notes = document.getElementById('pay-notes').value;
    const errEl = document.getElementById('pay-error');

    if (!studentID || !category || !owed || !paid) {
      errEl.textContent = 'Please fill in all required fields.';
      errEl.classList.remove('hidden');
      return;
    }

    const sessionID = document.getElementById('fin-session').value;
    const termID = document.getElementById('fin-term').value;

    try {
      const result = await api.post('/api/finance/payments', {
        student_id: studentID,
        category_name: category,
        amount_owed: owed,
        amount_paid: paid,
        payment_date: date,
        notes,
        session_id: sessionID,
        term_id: termID,
      });
      showToast(`Payment recorded. Receipt: ${result.receipt_ref}`, 'success');
      window._closePaymentModal();
    } catch (err) {
      errEl.textContent = err.message || 'Failed to record payment.';
      errEl.classList.remove('hidden');
    }
  });
}
