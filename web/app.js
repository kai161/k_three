import { api } from './api.js';
import { initialScreen, progressText, validateReflection } from './state.js';

const state = { status: null, reply: null, ritual: null };
const screens = [...document.querySelectorAll('.screen')];
const $ = (selector) => document.querySelector(selector);
let draftKey = 'unwind-draft';

function show(name) {
  screens.forEach((screen) => { screen.hidden = screen.id !== `screen-${name}`; });
  $('#global-error').hidden = true;
  window.scrollTo({ top: 0, behavior: 'instant' });
  const heading = $(`#screen-${name} h1`);
  if (heading) {
    heading.setAttribute('tabindex', '-1');
    heading.focus({ preventScroll: true });
  }
}

function showError(error) {
  const banner = $('#global-error');
  banner.textContent = error.message || '暂时没有成功，请稍后重试';
  banner.hidden = false;
  banner.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function renderStatus() {
  $('#progress-label').textContent = progressText(state.status);
  const dots = $('#day-dots');
  dots.replaceChildren();
  for (let day = 1; day <= state.status.totalDays; day += 1) {
    const dot = document.createElement('span');
    dot.className = 'day-dot';
    if (day <= state.status.completedDays) dot.classList.add('done');
    if (day === state.status.nextDay && !state.status.finished) dot.classList.add('current');
    dots.append(dot);
  }
}

function showWrite() {
  renderStatus();
  const textarea = $('#reflection');
  textarea.value = localStorage.getItem(draftKey) || '';
  updateCount();
  show('write');
}

function updateCount() {
  $('#word-count').textContent = `${Array.from($('#reflection').value).length} / 200`;
}

async function generateReply(event) {
  event.preventDefault();
  const text = $('#reflection').value;
  const validation = validateReflection(text);
  if (validation) {
    showError(new Error(validation));
    return;
  }
  const button = $('#reply-button');
  button.disabled = true;
  button.textContent = '正在认真读…';
  try {
    state.reply = await api.reply(text.trim());
    if (state.reply.source === 'safety') {
      $('#safety-message').textContent = state.reply.message;
      $('#safety-action').textContent = state.reply.action;
      show('safety');
    } else {
      $('#reply-source').textContent = state.reply.source === 'ai' ? 'AI 回应' : '人工晚安卡片';
      $('#reply-message').textContent = state.reply.message;
      $('#reply-action').textContent = state.reply.action;
      show('reply');
    }
  } catch (error) {
    if (error.status === 402) show('locked');
    else showError(error);
  } finally {
    button.disabled = false;
    button.replaceChildren(document.createTextNode('为今天收个尾 '), Object.assign(document.createElement('span'), { textContent: '→' }));
  }
}

function beginRitual(event) {
  state.ritual = event.currentTarget.dataset.ritual;
  const paper = $('#paper');
  paper.className = `paper ${state.ritual === '揉掉' ? 'crumple' : 'keep'}`;
  $('#ritual-title').textContent = state.ritual === '揉掉' ? '让今晚轻一点' : '先放在这里，明天再看';
  show('ritual');
  window.setTimeout(() => show('mood'), 800);
}

async function completeEntry(event) {
  const button = event.currentTarget;
  if (button.disabled) return;
  document.querySelectorAll('[data-mood]').forEach((item) => { item.disabled = true; });
  try {
    state.status = await api.complete(state.status.nextDay, button.dataset.mood, state.ritual);
    localStorage.removeItem(draftKey);
    if (state.status.finished) await showReview();
    else {
      $('#completion-copy').textContent = `这是你留下的第 ${state.status.completedDays} 个夜晚。每一次愿意停下来，都算数。`;
      $('#feedback-day2-form').hidden = !(state.status.publicBeta && state.status.completedDays === 2);
      show('complete');
    }
  } catch (error) {
    showError(error);
  } finally {
    document.querySelectorAll('[data-mood]').forEach((item) => { item.disabled = false; });
  }
}

async function activate(event) {
  event.preventDefault();
  const button = $('#activation-button');
  button.disabled = true;
  try {
    state.status = await api.activate($('#activation-code').value.trim());
    showWrite();
  } catch (error) {
    showError(error);
  } finally {
    button.disabled = false;
  }
}

async function showReview() {
  const [review, status] = await Promise.all([api.review(), Promise.resolve(state.status)]);
  $('#review-mood').textContent = review.mostFrequentMood || '没有固定答案';
  $('#review-message').textContent = review.message;
  const entries = $('#review-entries');
  entries.replaceChildren();
  $('#feedback-day7-form').hidden = !status.publicBeta;
  status.entries.forEach((entry) => {
    const row = document.createElement('div');
    row.className = 'review-entry';
    const day = document.createElement('span');
    day.textContent = `第 ${entry.day} 天 · ${entry.ritual}`;
    const mood = document.createElement('span');
    mood.textContent = entry.mood;
    row.append(day, mood);
    entries.append(row);
  });
  show('review');
}

async function submitFeedback(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector('[type="submit"]');
  const textarea = form.querySelector('textarea');
  button.disabled = true;
  try {
    await api.feedback(Number(form.dataset.day), textarea.value.trim());
    button.textContent = '已收到，谢谢你';
    textarea.disabled = true;
  } catch (error) {
    button.disabled = false;
    showError(error);
  }
}

function continuePlan() {
  if (initialScreen(state.status) === 'write') showWrite();
  else show(initialScreen(state.status));
}

async function init() {
  $('#reflection-form').addEventListener('submit', generateReply);
  $('#reflection').addEventListener('input', () => {
    updateCount();
    localStorage.setItem(draftKey, $('#reflection').value);
  });
  document.querySelectorAll('[data-ritual]').forEach((button) => button.addEventListener('click', beginRitual));
  document.querySelectorAll('[data-mood]').forEach((button) => button.addEventListener('click', completeEntry));
  document.querySelectorAll('.feedback-form').forEach((form) => form.addEventListener('submit', submitFeedback));
  document.querySelectorAll('[data-skip-feedback]').forEach((button) => button.addEventListener('click', () => {
    button.closest('.feedback-form').hidden = true;
  }));
  $('#activation-form').addEventListener('submit', activate);
  $('#safety-back').addEventListener('click', showWrite);
  $('#continue-button').addEventListener('click', continuePlan);
  try {
    state.status = await api.session();
    if (state.status.finished) await showReview();
    else show(initialScreen(state.status));
    if (initialScreen(state.status) === 'write') showWrite();
  } catch (error) {
    showError(error);
    show('write');
  }
}

init();
