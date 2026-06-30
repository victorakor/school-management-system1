/**
 * finance.js — Financial management portal module
 * REPLACE: web/static/js/portal/finance.js
 *
 * Changes from original:
 *   - Added Finance Summary stat cards (total collected, outstanding, debtor count)
 *   - Added Income Report tab with date range filter and printable table
 *   - Added Debt Report tab with outstanding balances per student
 *   - Tabs: Overview | Income Report | Debt Report
 *   - Original fee structure + record payment preserved on Overview tab
 */
import { api } from '../shared/api.js';
import { formatDate, formatCurrency, showToast } from '../shared/utils.js';

let _activeTab = 'overview';

export async function initFinance(container, user) {
  const canManage = user && (user.role === 'BURSAR' || user.role === 'OWNER');
  const isOwner   = user && user.role === 'OWNER';

  container.innerHTML = `
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-bold text-primary">Finance</h1>
      ${canManage ? '<button id="record-payment-btn" class="btn btn-primary">Record Payment</button>' : ''}
    </div>

    <!-- Summary Cards (Owner / Bursar) -->
    ${canManage ? `
    <div id="finance-summary-cards" class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
      <div class="skeleton h-24 rounded-2xl"></div>
      <div class="skeleton h-24 rounded-2xl"></div>
      <div class="skeleton h-24 rounded-2xl"></div>
    </div>` : ''}

    <!-- Tabs -->
    <div class="flex gap-1 mb-6 border-b border-neutral-100">
      <button class="finance-tab tab-btn px-4 py-2 text-sm font-medium border-b-2 transition-colors" data-tab="overview">Overview</button>
      ${canManage ? `
      <button class="finance-tab tab-btn px-4 py-2 text-sm font-medium border-b-2 transition-colors" data-tab="income">Income Report</button>
      <button class="finance-tab tab-btn px-4 py-2 text-sm font-medium border-b-2 transition-colors" data-tab="debts">Debt Report</button>
      ` : ''}
    </div>

    <div id="finance-tab-content"></div>
    <div id="payment-modal" class="hidden"></div>
  `;

  if (canManage) {
    await loadSummaryCards();
    document.getElementById('record-payment-btn')?.addEventListener('click', openPaymentModal);
  }

  // Tab routing
  container.querySelectorAll('.finance-tab').forEach(btn => {
    btn.addEventListener('click', () => switchTab(btn.dataset.tab, canManage));
  });

  switchTab('overview', canManage);
}

// ─── Tab Switcher ─────────────────────────────────────────────────────────────

function switchTab(tab, canManage) {
  _activeTab = tab;
  document.querySelectorAll('.finance-tab').forEach(b => {
    const active = b.dataset.tab === tab;
    b.classList.toggle('border-primary', active);
    b.classList.toggle('text-primary', active);
    b.classList.toggle('border-transparent', !active);
    b.classList.toggle('text-text-secondary', !active);
  });
  const content = document.getElementById('finance-tab-content');
  if (!content) return;

  if (tab === 'overview')   renderOverviewTab(content, canManage);
  if (tab === 'income')     renderIncomeTab(content);
  if (tab === 'debts')      renderDebtTab(content);
}

// ─── Summary Cards ────────────────────────────────────────────────────────────

async function loadSummaryCards() {
  const el = document.getElementById('finance-summary-cards');
  if (!el) return;
  try {
    const data = await api.get('/api/finance/summary');
    el.innerHTML = `
      ${summaryCard('Total Collected', formatCurrency(data.total_collected ?? 0), '#22c55e', `${data.payment_count ?? 0} payment(s)`)}
      ${summaryCard('Outstanding', formatCurrency(data.total_outstanding ?? 0), '#f59e0b', `${data.debtor_count ?? 0} debtor(s)`)}
      ${summaryCard('Debtors', data.debtor_count ?? 0, '#ef4444', 'Students with balance due')}
    `;
  } catch {
    el.innerHTML = `<p class="col-span-3 text-sm text-text-secondary">Could not load finance summary.</p>`;
  }
}

function summaryCard(label, value, color, sub) {
  return `
    <div class="card p-5 flex flex-col gap-2">
      <span class="text-text-secondary text-xs font-medium uppercase tracking-wide">${label}</span>
      <span class="text-3xl font-bold text-primary">${value}</span>
      <span class="text-xs text-text-secondary">${sub}</span>
    </div>`;
}

// ─── Overview Tab (original fee structure + recent payments) ──────────────────

function renderOverviewTab(content, canManage) {
  content.innerHTML = `
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
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-primary">Fee Structure</h3>
          ${canManage ? '<button class="btn btn-outline btn-sm" onclick="window._openFeeStructureForm()">Edit Structure</button>' : ''}
        </div>
        <div id="fee-structure-content" class="text-secondary text-sm">Select division, session, and term to view fee structure.</div>
      </div>
      <div id="recent-payments-panel" class="card p-5">
        <h3 class="font-semibold text-primary mb-4">Recent Payments</h3>
        <div id="recent-payments-content" class="text-secondary text-sm">No payments recorded yet.</div>
      </div>
    </div>`;

  loadSessions();
  document.getElementById('fin-session').addEventListener('change', onSessionChange);
  document.getElementById('fin-term').addEventListener('change', loadFeeStructure);
  document.getElementById('fin-division').addEventListener('change', loadFeeStructure);
}

async function loadSessions() {
  try {
    const sessions = await api.get('/api/sessions');
    const sel = document.getElementById('fin-session');
    if (!sel) return;
    sessions.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.id; opt.textContent = s.name;
      if (s.is_active) opt.selected = true;
      sel.appendChild(opt);
    });
    if (sel.value) await onSessionChange();
  } catch {}
}

async function onSessionChange() {
  const sessionID = document.getElementById('fin-session')?.value;
  if (!sessionID) return;
  try {
    const terms = await api.get(`/api/sessions/${sessionID}/terms`);
    const sel = document.getElementById('fin-term');
    if (!sel) return;
    sel.innerHTML = '<option value="">Select Term</option>';
    terms.forEach(t => {
      const opt = document.createElement('option');
      opt.value = t.id; opt.textContent = t.name;
      sel.appendChild(opt);
    });
  } catch {}
}

async function loadFeeStructure() {
  const divisionID = document.getElementById('fin-division')?.value;
  const sessionID  = document.getElementById('fin-session')?.value;
  const termID     = document.getElementById('fin-term')?.value;
  const content    = document.getElementById('fee-structure-content');
  if (!content || !divisionID || !sessionID || !termID) return;
  content.innerHTML = `<div class="skeleton h-32 rounded"></div>`;
  try {
    const structure = await api.get(`/api/finance/structure?division_id=${divisionID}&session_id=${sessionID}&term_id=${termID}`);
    if (!structure?.categories?.length) {
      content.innerHTML = `<p class="text-secondary text-sm">No fee structure set for this period.</p>`;
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

// ─── Income Report Tab ────────────────────────────────────────────────────────

function renderIncomeTab(content) {
  const today = new Date().toISOString().slice(0, 10);
  const monthStart = today.slice(0, 7) + '-01';

  content.innerHTML = `
    <div class="card p-5 mb-4">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3 items-end">
        <div>
          <label class="label text-xs">From</label>
          <input id="income-from" type="date" class="input text-sm py-1.5" value="${monthStart}" />
        </div>
        <div>
          <label class="label text-xs">To</label>
          <input id="income-to" type="date" class="input text-sm py-1.5" value="${today}" />
        </div>
        <div>
          <label class="label text-xs">Division</label>
          <select id="income-division" class="input text-sm py-1.5">
            <option value="">All Divisions</option>
            <option value="NURSERY">Nursery</option>
            <option value="PRIMARY">Primary</option>
            <option value="SECONDARY">Secondary</option>
          </select>
        </div>
        <button id="load-income-btn" class="btn btn-primary text-sm">Load Report</button>
      </div>
    </div>
    <div id="income-results"></div>`;

  document.getElementById('load-income-btn').addEventListener('click', loadIncomeReport);
  loadIncomeReport();
}

async function loadIncomeReport() {
  const results = document.getElementById('income-results');
  if (!results) return;
  results.innerHTML = `<div class="card p-4 space-y-2">${Array(6).fill('<div class="skeleton h-10 rounded"></div>').join('')}</div>`;

  const params = new URLSearchParams({
    from:     document.getElementById('income-from')?.value || '',
    to:       document.getElementById('income-to')?.value || '',
    division: document.getElementById('income-division')?.value || '',
  });

  try {
    const data = await api.get(`/api/finance/reports/income?${params}`);
    const payments = data.payments || [];

    if (!payments.length) {
      results.innerHTML = `<div class="card p-12 text-center text-text-secondary">No payments found for this period.</div>`;
      return;
    }

    const total = payments.reduce((s, p) => s + (p.amount_paid || 0), 0);
    results.innerHTML = `
      <div class="flex items-center justify-between mb-3">
        <p class="text-sm text-text-secondary">${payments.length} payment(s) found</p>
        <p class="font-semibold text-primary">Total: ${formatCurrency(total)}</p>
      </div>
      <div class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="data-table w-full text-sm">
            <thead><tr>
              <th>Date</th><th>Student</th><th>Category</th>
              <th class="text-right">Owed</th><th class="text-right">Paid</th>
              <th class="text-right">Balance</th><th>Receipt</th>
            </tr></thead>
            <tbody>
              ${payments.map(p => `
                <tr>
                  <td class="whitespace-nowrap text-xs text-text-secondary">${formatDate(new Date(p.payment_date))}</td>
                  <td class="font-medium text-primary">${p.student_name}</td>
                  <td class="text-text-secondary text-xs">${p.category_name}</td>
                  <td class="text-right text-xs">${formatCurrency(p.amount_owed)}</td>
                  <td class="text-right text-xs font-medium text-success">${formatCurrency(p.amount_paid)}</td>
                  <td class="text-right text-xs ${p.balance_after > 0 ? 'text-danger' : 'text-success'}">${formatCurrency(p.balance_after)}</td>
                  <td class="font-mono text-xs text-text-secondary">${p.receipt_ref || '—'}</td>
                </tr>`).join('')}
            </tbody>
          </table>
        </div>
      </div>`;
  } catch (err) {
    results.innerHTML = `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
  }
}

// ─── Debt Report Tab ──────────────────────────────────────────────────────────

function renderDebtTab(content) {
  content.innerHTML = `
    <div class="card p-4 mb-4">
      <div class="grid grid-cols-2 md:grid-cols-3 gap-3 items-end">
        <div>
          <label class="label text-xs">Division</label>
          <select id="debt-division" class="input text-sm py-1.5">
            <option value="">All Divisions</option>
            <option value="NURSERY">Nursery</option>
            <option value="PRIMARY">Primary</option>
            <option value="SECONDARY">Secondary</option>
          </select>
        </div>
        <button id="load-debt-btn" class="btn btn-primary text-sm">Load Debtors</button>
      </div>
    </div>
    <div id="debt-results"></div>`;

  document.getElementById('load-debt-btn').addEventListener('click', loadDebtReport);
  loadDebtReport();
}

async function loadDebtReport() {
  const results = document.getElementById('debt-results');
  if (!results) return;
  results.innerHTML = `<div class="card p-4 space-y-2">${Array(5).fill('<div class="skeleton h-10 rounded"></div>').join('')}</div>`;

  const division = document.getElementById('debt-division')?.value || '';
  const params = new URLSearchParams({ ...(division && { division }) });

  try {
    const data = await api.get(`/api/finance/reports/debts?${params}`);
    const debts = data.debts || [];

    if (!debts.length) {
      results.innerHTML = `<div class="card p-12 text-center text-success font-medium">No outstanding debts. 🎉</div>`;
      return;
    }

    results.innerHTML = `
      <div class="flex items-center justify-between mb-3">
        <p class="text-sm text-text-secondary">${debts.length} student(s) with outstanding balance</p>
        <p class="font-semibold text-danger">Total outstanding: ${formatCurrency(data.total_outstanding || 0)}</p>
      </div>
      <div class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="data-table w-full text-sm">
            <thead><tr>
              <th>Student</th><th>Admission ID</th><th>Category</th>
              <th class="text-right">Owed</th><th class="text-right">Paid</th>
              <th class="text-right">Balance Due</th>
            </tr></thead>
            <tbody>
              ${debts.map(d => `
                <tr>
                  <td class="font-medium text-primary">${d.student_name}</td>
                  <td class="font-mono text-xs text-text-secondary">${d.admission_id || '—'}</td>
                  <td class="text-text-secondary text-xs">${d.category_name}</td>
                  <td class="text-right text-xs">${formatCurrency(d.amount_owed)}</td>
                  <td class="text-right text-xs">${formatCurrency(d.amount_paid)}</td>
                  <td class="text-right text-xs font-semibold text-danger">${formatCurrency(d.balance_after)}</td>
                </tr>`).join('')}
            </tbody>
          </table>
        </div>
      </div>`;
  } catch (err) {
    results.innerHTML = `<div class="card p-6 text-danger text-sm">${err.message}</div>`;
  }
}

// ─── Record Payment Modal (unchanged from original) ───────────────────────────

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
          <input type="text" id="pay-student-search" class="input w-full" placeholder="Search student…">
          <input type="hidden" id="pay-student-id">
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div><label class="label block mb-1">Fee Category</label><input type="text" id="pay-category" class="input w-full" placeholder="e.g. Tuition"></div>
          <div><label class="label block mb-1">Amount Owed (₦)</label><input type="number" id="pay-owed" class="input w-full" min="0" step="0.01"></div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div><label class="label block mb-1">Amount Paid (₦)</label><input type="number" id="pay-amount" class="input w-full" min="0" step="0.01"></div>
          <div><label class="label block mb-1">Payment Date</label><input type="date" id="pay-date" class="input w-full" value="${new Date().toISOString().split('T')[0]}"></div>
        </div>
        <div><label class="label block mb-1">Notes (optional)</label><textarea id="pay-notes" class="input w-full" rows="2"></textarea></div>
        <div id="pay-error" class="text-danger text-sm hidden"></div>
        <div class="flex gap-3 pt-2">
          <button type="submit" class="btn btn-primary flex-1">Record Payment</button>
          <button type="button" onclick="window._closePaymentModal()" class="btn btn-outline flex-1">Cancel</button>
        </div>
      </form>
    </div>`;

  window._closePaymentModal = () => { modal.className = 'hidden'; modal.innerHTML = ''; };

  document.getElementById('payment-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const studentID = document.getElementById('pay-student-id').value;
    const category  = document.getElementById('pay-category').value;
    const owed      = parseFloat(document.getElementById('pay-owed').value);
    const paid      = parseFloat(document.getElementById('pay-amount').value);
    const date      = document.getElementById('pay-date').value;
    const notes     = document.getElementById('pay-notes').value;
    const errEl     = document.getElementById('pay-error');
    if (!studentID || !category || !owed || !paid) {
      errEl.textContent = 'Please fill in all required fields.';
      errEl.classList.remove('hidden');
      return;
    }
    const sessionID = document.getElementById('fin-session')?.value;
    const termID    = document.getElementById('fin-term')?.value;
    try {
      const result = await api.post('/api/finance/payments', {
        student_id: studentID, category_name: category,
        amount_owed: owed, amount_paid: paid,
        payment_date: date, notes, session_id: sessionID, term_id: termID,
      });
      showToast(`Payment recorded. Receipt: ${result.receipt_ref}`, 'success');
      window._closePaymentModal();
      await loadSummaryCards();
    } catch (err) {
      errEl.textContent = err.message || 'Failed to record payment.';
      errEl.classList.remove('hidden');
    }
  });
}
