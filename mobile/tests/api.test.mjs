import assert from 'node:assert/strict';
import { afterEach, test } from 'node:test';

import { getCurrentUser } from '../src/lib/api.ts';

const originalFetch = globalThis.fetch;
const originalUrl = process.env.EXPO_PUBLIC_API_BASE_URL;
globalThis.__DEV__ = true;

afterEach(() => {
  globalThis.fetch = originalFetch;
  globalThis.__DEV__ = true;
  if (originalUrl === undefined) delete process.env.EXPO_PUBLIC_API_BASE_URL;
  else process.env.EXPO_PUBLIC_API_BASE_URL = originalUrl;
});

test('sends the Firebase user token to the protected Go endpoint', async () => {
  process.env.EXPO_PUBLIC_API_BASE_URL = 'http://127.0.0.1:8080/';
  const controller = new AbortController();
  const user = { getIdToken: async () => 'mock-firebase-token' };
  globalThis.fetch = async (url, options) => {
    assert.equal(url, 'http://127.0.0.1:8080/api/users/me');
    assert.equal(options.headers.Authorization, 'Bearer mock-firebase-token');
    assert.equal(options.signal, controller.signal);
    assert.equal(options.redirect, 'error');
    return Response.json({ uid: 'verified-uid' });
  };
  assert.equal(await getCurrentUser(user, controller.signal), 'verified-uid');
});

test('rejects missing configuration before requesting a token', async () => {
  delete process.env.EXPO_PUBLIC_API_BASE_URL;
  await assert.rejects(getCurrentUser({ getIdToken() { assert.fail('must not request a token'); } }), /Set EXPO_PUBLIC_API_BASE_URL/);
});

test('rejects plaintext HTTP outside development', async () => {
  globalThis.__DEV__ = false;
  process.env.EXPO_PUBLIC_API_BASE_URL = 'http://127.0.0.1:8080';
  await assert.rejects(getCurrentUser({ getIdToken: async () => '' }), /HTTPS/);
});

test('reports unauthorized responses without reflecting backend content', async () => {
  process.env.EXPO_PUBLIC_API_BASE_URL = 'https://api.example.test';
  globalThis.fetch = async () => new Response('private error details', { status: 401 });
  await assert.rejects(getCurrentUser({ getIdToken: async () => 'mock-firebase-token' }), /backend rejected the Firebase session/);
});

test('rejects invalid backend user responses', async () => {
  process.env.EXPO_PUBLIC_API_BASE_URL = 'https://api.example.test';
  for (const body of [null, {}, { uid: 123 }, { uid: '' }]) {
    globalThis.fetch = async () => Response.json(body);
    await assert.rejects(getCurrentUser({ getIdToken: async () => 'mock-firebase-token' }), /invalid user response/);
  }
});
