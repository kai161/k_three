export function validateReflection(text) {
  const value = text.trim();
  if (!value) return '写下一点此刻的感受吧';
  if (Array.from(value).length > 200) return '最多写 200 字，已经不少了';
  return '';
}

export function initialScreen(status) {
  if (status.finished) return 'review';
  if (!status.publicBeta && status.nextDay > 2 && !status.activated) return 'locked';
  return 'write';
}

export function progressText(status) {
  if (status.completedDays === 0) return `第 ${status.nextDay} 天 · ${status.totalDays} 个夜晚`;
  return `已留下 ${status.completedDays} 个夜晚`;
}
