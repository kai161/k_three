import test from 'node:test';
import assert from 'node:assert/strict';
import { initialScreen, progressText, validateReflection } from './state.js';

test('reflection rejects blank and overlong text', () => {
  assert.equal(validateReflection('  '), '写下一点此刻的感受吧');
  assert.equal(validateReflection('事'.repeat(201)), '最多写 200 字，已经不少了');
  assert.equal(validateReflection('今天有点累'), '');
});

test('initial screen shows review, lock, or writing based on progress', () => {
  assert.equal(initialScreen({ finished: true }), 'review');
  assert.equal(initialScreen({ finished: false, nextDay: 3, activated: false }), 'locked');
  assert.equal(initialScreen({ finished: false, nextDay: 2, activated: false }), 'write');
  assert.equal(initialScreen({ finished: false, nextDay: 3, activated: true }), 'write');
});

test('progress text explains free days without pressure', () => {
  assert.equal(progressText({ completedDays: 0, nextDay: 1, totalDays: 7 }), '第 1 天 · 7 个夜晚');
  assert.equal(progressText({ completedDays: 2, nextDay: 3, totalDays: 7 }), '已留下 2 个夜晚');
});

test('public beta never sends users to activation', () => {
  assert.equal(initialScreen({ publicBeta: true, finished: false, nextDay: 3, activated: false }), 'write');
  assert.equal(initialScreen({ publicBeta: true, finished: false, nextDay: 7, activated: false }), 'write');
});
