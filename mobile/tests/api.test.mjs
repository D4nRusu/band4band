import assert from 'node:assert/strict';
import { afterEach, test } from 'node:test';

import { getCurrentUser, completeProfile, authenticatedRequest } from '../src/lib/api.ts';

const profile = { uid: 'verified-uid', email: 'user@example.test', name: 'Example User', username: '', photoURL: '', createdAt: '2026-11-04T12:00:00Z', profileComplete: false };

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
    return Response.json(profile);
  };
  assert.deepEqual(await getCurrentUser(user, controller.signal), profile);
});

test('uploads multipart data with Firebase authentication and leaves its boundary to native fetch', async () => {
  process.env.EXPO_PUBLIC_API_BASE_URL = 'http://127.0.0.1:8080';
  const body = new FormData();
  body.append('image', new Blob(['test image'], { type: 'image/png' }), 'receipt.png');
  const controller = new AbortController();
  globalThis.fetch = async (url, options) => {
    assert.equal(url, 'http://127.0.0.1:8080/api/groups/group-id/receipts/extract');
    assert.equal(options.headers.Authorization, 'Bearer mock-firebase-token');
    assert.equal(Object.hasOwn(options.headers, 'Content-Type'), false);
    assert.equal(options.body, body);
    assert.equal(options.signal, controller.signal);
    return Response.json({});
  };

  await authenticatedRequest({ getIdToken: async () => 'mock-firebase-token' },
    '/api/groups/group-id/receipts/extract', { method: 'POST', body, signal: controller.signal });
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
  for (const body of [null, {}, { uid: 123 }, { uid: '' },
    { ...profile, createdAt: 'invalid' }, { ...profile, profileComplete: true },
    { ...profile, username: 'alice', profileComplete: false },
    { ...profile, email: null }, { ...profile, photoURL: 123 }]) {
    globalThis.fetch = async () => Response.json(body);
    await assert.rejects(getCurrentUser({ getIdToken: async () => 'mock-firebase-token' }), /invalid user response/);
  }
});

test('submits only the username with a Firebase token and returns the completed profile', async () => {
  process.env.EXPO_PUBLIC_API_BASE_URL = 'http://127.0.0.1:8080';
  const completed = { ...profile, username: 'alice_123', profileComplete: true };
  globalThis.fetch = async (url, options) => {
    assert.equal(url, 'http://127.0.0.1:8080/api/users/me/username');
    assert.equal(options.method, 'PUT');
    assert.equal(options.headers.Authorization, 'Bearer mock-firebase-token');
    assert.equal(options.headers['Content-Type'], 'application/json');
    assert.deepEqual(JSON.parse(options.body), { username: 'Alice_123' });
    return Response.json(completed);
  };
  assert.deepEqual(await completeProfile({ getIdToken: async () => 'mock-firebase-token' }, 'Alice_123'), completed);
});

test('gives actionable feedback when another user has claimed a username', async () => {
  process.env.EXPO_PUBLIC_API_BASE_URL = 'https://api.example.test';
  globalThis.fetch = async () => Response.json({ code: 'username_taken', message: 'untrusted details' }, { status: 409 });
  await assert.rejects(completeProfile({ getIdToken: async () => 'mock-firebase-token' }, 'alice'), /already taken.*choose another/);
});

test('preserves the distinction between username conflicts and completed profiles', async () => {
  process.env.EXPO_PUBLIC_API_BASE_URL = 'https://api.example.test';
  globalThis.fetch = async () => Response.json({ code: 'profile_complete' }, { status: 409 });
  await assert.rejects(completeProfile({ getIdToken: async () => 'mock-firebase-token' }, 'other'), /already has a username/);
});

test('reports backend username validation failures', async () => {
  process.env.EXPO_PUBLIC_API_BASE_URL = 'https://api.example.test';
  globalThis.fetch = async () => Response.json({ code: 'invalid_username' }, { status: 400 });
  await assert.rejects(completeProfile({ getIdToken: async () => 'mock-firebase-token' }, 'bad name'), /3–30/);
});
