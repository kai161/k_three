async function request(path, options = {}) {
  let response;
  try {
    response = await fetch(path, {
      ...options,
      headers: { 'Content-Type': 'application/json', ...options.headers },
    });
  } catch {
    throw new Error('网络似乎走神了，请检查连接后再试');
  }
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(data.error || '暂时没有成功，请稍后重试');
    error.status = response.status;
    throw error;
  }
  return data;
}

export const api = {
  session: () => request('/api/session', { method: 'POST' }),
  reply: (text) => request('/api/replies', { method: 'POST', body: JSON.stringify({ text }) }),
  complete: (day, mood, ritual) => request('/api/entries', {
    method: 'POST',
    body: JSON.stringify({ day, mood, ritual }),
  }),
  activate: (code) => request('/api/activate', { method: 'POST', body: JSON.stringify({ code }) }),
  feedback: (day, text) => request('/api/feedback', { method: 'POST', body: JSON.stringify({ day, text }) }),
  review: () => request('/api/review'),
};
