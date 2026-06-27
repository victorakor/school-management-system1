/**
 * quiz.js — Full quiz interface with complete anti-cheat enforcement.
 *
 * Anti-cheat features implemented:
 * - Copy/paste/cut blocked
 * - Tab switch detection + auto-submit on limit
 * - Fullscreen enforcement
 * - Right-click disabled
 * - Keyboard shortcuts blocked (Ctrl+C, Ctrl+V, Ctrl+U, F12, etc.)
 * - Countdown timer with auto-submit
 * - Periodic answer save (every 30s)
 * - Violation logging to server
 * - Refresh handling (answers restored from server)
 */

import { api } from '../shared/api.js';
import { $, formatCountdown } from '../shared/utils.js';

let quizState = {
  attemptId: null,
  quizId: null,
  questions: [],
  answers: {},
  currentIndex: 0,
  totalSeconds: 0,
  remainingSeconds: 0,
  tabSwitchCount: 0,
  tabSwitchLimit: 3,
  fullscreenExitCount: 0,
  timerInterval: null,
  saveInterval: null,
  isSubmitted: false,
};

// ─── Init ─────────────────────────────────────────────────────────────────────

export async function initQuiz(quizId) {
  quizState.quizId = quizId;

  try {
    const data = await api.post(`/api/quizzes/${quizId}/start`);
    quizState.attemptId = data.attempt_id;
    quizState.questions = data.questions;
    quizState.tabSwitchLimit = data.tab_switch_limit ?? 3;
    quizState.totalSeconds = (data.duration_minutes ?? 60) * 60;
    quizState.remainingSeconds = quizState.totalSeconds;

    // Restore any saved answers
    if (data.saved_answers) {
      data.saved_answers.forEach(a => {
        quizState.answers[a.question_id] = a.answer;
      });
    }

    renderQuiz();
    enterFullscreen();
    attachAntiCheat();
    startTimer();
    startPeriodicSave();
  } catch (err) {
    showError(err.message || 'Failed to start quiz');
  }
}

// ─── Render ───────────────────────────────────────────────────────────────────

function renderQuiz() {
  const container = $('#quiz-container');
  if (!container) return;

  container.innerHTML = `
    <!-- Top bar -->
    <div class="fixed top-0 left-0 right-0 z-50 bg-primary text-white px-6 py-3 flex items-center justify-between shadow-lg" id="quiz-topbar">
      <span class="font-display font-semibold text-sm truncate max-w-xs" id="quiz-title">Quiz in Progress</span>
      <div class="flex items-center gap-6">
        <div id="quiz-timer" class="font-mono font-bold text-lg" aria-live="polite" aria-label="Time remaining">--:--</div>
        <span class="text-white/70 text-sm" id="quiz-progress">Q 1 / ${quizState.questions.length}</span>
      </div>
    </div>

    <!-- Question area -->
    <div class="min-h-screen bg-background pt-16 pb-24 px-4 flex items-start justify-center">
      <div class="w-full max-w-2xl mt-8">

        <!-- Question map -->
        <div class="flex flex-wrap gap-2 mb-8" id="question-map" role="navigation" aria-label="Question navigation">
          ${quizState.questions.map((_, i) => `
            <button
              class="question-dot w-9 h-9 rounded-full text-xs font-semibold border-2 transition-all duration-200 min-h-[44px] min-w-[44px]
                     ${quizState.answers[quizState.questions[i]?.question_id] ? 'bg-success border-success text-white' : 'bg-white border-neutral-200 text-text-secondary'}"
              data-index="${i}"
              onclick="window.quizGoTo(${i})"
              aria-label="Question ${i + 1}${quizState.answers[quizState.questions[i]?.question_id] ? ' (answered)' : ''}"
            >${i + 1}</button>`).join('')}
        </div>

        <!-- Question card -->
        <div class="bg-white rounded-2xl border border-neutral-100 shadow-card p-8 mb-6" id="question-card">
          <!-- Populated by renderQuestion() -->
        </div>

        <!-- Navigation -->
        <div class="flex items-center justify-between">
          <button id="quiz-prev" onclick="window.quizPrev()" class="btn btn-secondary" ${quizState.currentIndex === 0 ? 'disabled' : ''}>
            ← Previous
          </button>
          <button id="quiz-next" onclick="window.quizNext()" class="btn btn-primary" id="quiz-next-btn">
            ${quizState.currentIndex === quizState.questions.length - 1 ? 'Review & Submit' : 'Next →'}
          </button>
        </div>
      </div>
    </div>

    <!-- Submit button (shown on last question) -->
    <div id="quiz-submit-bar" class="fixed bottom-0 left-0 right-0 bg-white border-t border-neutral-100 p-4 hidden">
      <div class="max-w-2xl mx-auto">
        <button onclick="window.quizSubmit()" class="btn btn-primary w-full text-base py-4">
          Submit Quiz (${Object.keys(quizState.answers).length}/${quizState.questions.length} answered)
        </button>
      </div>
    </div>

    <!-- Violation warning overlay -->
    <div id="violation-overlay" class="fixed inset-0 z-[100] bg-danger/95 text-white flex items-center justify-center hidden" role="alertdialog" aria-modal="true" aria-labelledby="violation-title">
      <div class="text-center max-w-md px-8">
        <div class="text-6xl mb-4" aria-hidden="true">⚠️</div>
        <h2 id="violation-title" class="font-display font-bold text-2xl mb-3">Warning!</h2>
        <p id="violation-message" class="text-white/90 mb-6">You have left the quiz window. This has been recorded.</p>
        <button onclick="window.dismissViolation()" class="btn bg-white text-danger font-bold px-8 py-3">
          Return to Quiz
        </button>
      </div>
    </div>`;

  renderQuestion(quizState.currentIndex);

  // Expose global functions for inline handlers
  window.quizGoTo = (i) => { quizState.currentIndex = i; renderQuestion(i); };
  window.quizPrev = () => { if (quizState.currentIndex > 0) { quizState.currentIndex--; renderQuestion(quizState.currentIndex); } };
  window.quizNext = () => {
    if (quizState.currentIndex < quizState.questions.length - 1) {
      quizState.currentIndex++;
      renderQuestion(quizState.currentIndex);
    } else {
      showSubmitBar();
    }
  };
  window.quizSubmit = () => submitQuiz();
  window.dismissViolation = () => {
    $('#violation-overlay')?.classList.add('hidden');
    enterFullscreen();
  };
}

function renderQuestion(index) {
  const card = $('#question-card');
  const progress = $('#quiz-progress');
  const prevBtn = $('#quiz-prev');
  const nextBtn = $('#quiz-next');
  if (!card) return;

  const q = quizState.questions[index];
  if (!q) return;

  if (progress) progress.textContent = `Q ${index + 1} / ${quizState.questions.length}`;
  if (prevBtn) prevBtn.disabled = index === 0;
  if (nextBtn) nextBtn.textContent = index === quizState.questions.length - 1 ? 'Review & Submit' : 'Next →';

  const savedAnswer = quizState.answers[q.question_id];

  card.innerHTML = `
    <div class="mb-2">
      <span class="badge badge-accent">Question ${index + 1}</span>
    </div>
    <p class="font-display font-semibold text-lg text-text-primary mb-6 leading-relaxed no-select">${q.text}</p>
    <div class="space-y-3" id="options-container" role="radiogroup" aria-label="Answer options">
      ${renderOptions(q, savedAnswer)}
    </div>`;

  // Attach option click handlers
  card.querySelectorAll('.quiz-option').forEach(btn => {
    btn.addEventListener('click', () => {
      const answer = btn.dataset.value;
      quizState.answers[q.question_id] = answer;
      // Update UI
      card.querySelectorAll('.quiz-option').forEach(b => b.classList.remove('selected'));
      btn.classList.add('selected');
      // Update question map dot
      updateQuestionDot(index, true);
      // Update submit bar count
      updateSubmitBar();
    });
  });

  // Update question map
  updateQuestionDot(index, !!savedAnswer);
}

function renderOptions(q, savedAnswer) {
  if (q.type === 'TRUE_FALSE') {
    return ['True', 'False'].map(opt => `
      <button class="quiz-option ${savedAnswer === opt ? 'selected' : ''}" data-value="${opt}" role="radio" aria-checked="${savedAnswer === opt}">
        ${opt}
      </button>`).join('');
  }

  if (q.type === 'FILL_BLANK') {
    return `
      <input
        type="text"
        class="input"
        placeholder="Type your answer here..."
        value="${savedAnswer || ''}"
        id="fill-blank-input"
        aria-label="Your answer"
      />`;
  }

  // MCQ
  const options = Array.isArray(q.options) ? q.options : [];
  return options.map(opt => `
    <button class="quiz-option ${savedAnswer === opt ? 'selected' : ''}" data-value="${opt}" role="radio" aria-checked="${savedAnswer === opt}">
      ${opt}
    </button>`).join('');
}

function updateQuestionDot(index, answered) {
  const dot = document.querySelector(`.question-dot[data-index="${index}"]`);
  if (!dot) return;
  if (answered) {
    dot.classList.add('bg-success', 'border-success', 'text-white');
    dot.classList.remove('bg-white', 'border-neutral-200', 'text-text-secondary');
  } else {
    dot.classList.remove('bg-success', 'border-success', 'text-white');
    dot.classList.add('bg-white', 'border-neutral-200', 'text-text-secondary');
  }
}

function showSubmitBar() {
  const bar = $('#quiz-submit-bar');
  bar?.classList.remove('hidden');
}

function updateSubmitBar() {
  const bar = $('#quiz-submit-bar');
  if (!bar) return;
  const count = Object.keys(quizState.answers).length;
  const btn = bar.querySelector('button');
  if (btn) btn.textContent = `Submit Quiz (${count}/${quizState.questions.length} answered)`;
}

// ─── Timer ────────────────────────────────────────────────────────────────────

function startTimer() {
  const timerEl = $('#quiz-timer');

  quizState.timerInterval = setInterval(() => {
    quizState.remainingSeconds--;

    if (timerEl) {
      timerEl.textContent = formatCountdown(quizState.remainingSeconds);

      // Color changes
      if (quizState.remainingSeconds <= 60) {
        timerEl.classList.add('text-danger', 'animate-pulse');
        timerEl.classList.remove('text-amber-300');
      } else if (quizState.remainingSeconds <= 300) {
        timerEl.classList.add('text-amber-300');
      }
    }

    if (quizState.remainingSeconds <= 0) {
      clearInterval(quizState.timerInterval);
      submitQuiz(true);
    }
  }, 1000);
}

// ─── Periodic Save ────────────────────────────────────────────────────────────

function startPeriodicSave() {
  quizState.saveInterval = setInterval(async () => {
    if (quizState.isSubmitted || !quizState.attemptId) return;
    try {
      const answers = buildAnswersPayload();
      await api.post(`/api/quizzes/attempts/${quizState.attemptId}/save`, { answers });
    } catch {}
  }, 30_000);
}

// ─── Submit ───────────────────────────────────────────────────────────────────

async function submitQuiz(autoSubmit = false) {
  if (quizState.isSubmitted) return;
  quizState.isSubmitted = true;

  clearInterval(quizState.timerInterval);
  clearInterval(quizState.saveInterval);

  // Capture fill-blank answer if on that question
  const fillBlank = $('#fill-blank-input');
  if (fillBlank) {
    const q = quizState.questions[quizState.currentIndex];
    if (q) quizState.answers[q.question_id] = fillBlank.value;
  }

  const answers = buildAnswersPayload();

  try {
    const result = await api.post(`/api/quizzes/attempts/${quizState.attemptId}/submit`, { answers });
    showResult(result, autoSubmit);
  } catch (err) {
    showError('Failed to submit quiz. Please contact your teacher.');
  }

  exitFullscreen();
}

function buildAnswersPayload() {
  return Object.entries(quizState.answers).map(([question_id, answer]) => ({
    question_id,
    answer,
  }));
}

function showResult(result, autoSubmit) {
  const container = $('#quiz-container');
  if (!container) return;

  container.innerHTML = `
    <div class="min-h-screen bg-background flex items-center justify-center p-6">
      <div class="bg-white rounded-2xl border border-neutral-100 shadow-card p-10 max-w-md w-full text-center">
        <div class="w-16 h-16 bg-success/10 rounded-full flex items-center justify-center mx-auto mb-4" aria-hidden="true">
          <svg class="w-8 h-8 text-success" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>
        </div>
        <h2 class="font-display font-bold text-2xl text-primary mb-2">${autoSubmit ? 'Time Up!' : 'Quiz Submitted!'}</h2>
        <p class="text-text-secondary mb-6">${autoSubmit ? 'Your answers were automatically submitted when time expired.' : 'Your answers have been recorded successfully.'}</p>
        <div class="bg-background rounded-xl p-6 mb-6">
          <p class="text-4xl font-display font-bold text-primary">${result.score}<span class="text-xl text-text-secondary">/${result.total}</span></p>
          <p class="text-text-secondary text-sm mt-1">Score</p>
        </div>
        <a href="/portal/results" class="btn btn-primary w-full">View Results</a>
      </div>
    </div>`;
}

function showError(message) {
  const container = $('#quiz-container');
  if (container) {
    container.innerHTML = `
      <div class="min-h-screen flex items-center justify-center p-6">
        <div class="text-center">
          <p class="text-danger font-semibold mb-2">Error</p>
          <p class="text-text-secondary">${message}</p>
        </div>
      </div>`;
  }
}

// ─── Anti-Cheat ───────────────────────────────────────────────────────────────

function attachAntiCheat() {
  // Block copy/paste/cut
  ['copy', 'paste', 'cut'].forEach(event => {
    window.addEventListener(event, (e) => {
      e.preventDefault();
      logViolation(event.toUpperCase());
    });
  });

  // Block right-click
  window.addEventListener('contextmenu', (e) => {
    e.preventDefault();
    logViolation('RIGHT_CLICK');
  });

  // Block keyboard shortcuts
  window.addEventListener('keydown', (e) => {
    const blocked = (
      (e.ctrlKey && ['c', 'v', 'u', 'a', 's'].includes(e.key.toLowerCase())) ||
      (e.ctrlKey && e.shiftKey && ['i', 'j', 'c'].includes(e.key.toLowerCase())) ||
      e.key === 'F12'
    );
    if (blocked) {
      e.preventDefault();
      logViolation('KEYBOARD_SHORTCUT');
    }
  });

  // Tab switch / blur detection
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      quizState.tabSwitchCount++;
      logViolation('TAB_SWITCH');
      checkTabSwitchLimit();
    }
  });

  window.addEventListener('blur', () => {
    logViolation('BLUR');
  });

  // Fullscreen exit detection
  document.addEventListener('fullscreenchange', () => {
    if (!document.fullscreenElement && !quizState.isSubmitted) {
      quizState.fullscreenExitCount++;
      if (quizState.fullscreenExitCount === 1) {
        showViolationOverlay('You have exited fullscreen mode. Please return to fullscreen to continue.');
      } else {
        // Second exit — auto-submit
        logViolation('FULLSCREEN_EXIT');
        submitQuiz(true);
      }
    }
  });

  // Capture beforeunload — save answers
  window.addEventListener('beforeunload', (e) => {
    if (!quizState.isSubmitted) {
      // Synchronous save attempt
      const answers = buildAnswersPayload();
      navigator.sendBeacon(
        `/api/quizzes/attempts/${quizState.attemptId}/save`,
        JSON.stringify({ answers })
      );
    }
  });
}

function checkTabSwitchLimit() {
  if (quizState.tabSwitchCount >= quizState.tabSwitchLimit) {
    submitQuiz(true);
  } else {
    showViolationOverlay(
      `You have switched tabs ${quizState.tabSwitchCount} time(s). After ${quizState.tabSwitchLimit} switches, your quiz will be automatically submitted.`
    );
  }
}

function showViolationOverlay(message) {
  const overlay = $('#violation-overlay');
  const msg = $('#violation-message');
  if (overlay) overlay.classList.remove('hidden');
  if (msg) msg.textContent = message;
}

async function logViolation(type) {
  if (!quizState.attemptId) return;
  try {
    const result = await api.post(`/api/quizzes/attempts/${quizState.attemptId}/violation`, { type });
    if (result?.auto_submit) {
      submitQuiz(true);
    }
  } catch {}
}

// ─── Fullscreen ───────────────────────────────────────────────────────────────

function enterFullscreen() {
  const el = document.documentElement;
  if (el.requestFullscreen) {
    el.requestFullscreen().catch(() => {});
  } else if (el.webkitRequestFullscreen) {
    el.webkitRequestFullscreen();
  }
}

function exitFullscreen() {
  if (document.exitFullscreen) {
    document.exitFullscreen().catch(() => {});
  }
}

export default { initQuiz };
