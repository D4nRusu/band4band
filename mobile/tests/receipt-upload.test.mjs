import assert from 'node:assert/strict';
import { afterEach, test } from 'node:test';

import { createReceiptUpload } from '../src/lib/receipt-upload.ts';

const originalFetch = globalThis.fetch;
const photo = { uri: 'file:///cache/receipt.png', mimeType: 'image/png', name: 'receipt.png' };

afterEach(() => {
  globalThis.fetch = originalFetch;
});

test('reads the local photo and uploads its exact bytes as a named Blob', async () => {
  const bytes = new Uint8Array([137, 80, 78, 71, 0, 255]);
  const controller = new AbortController();
  globalThis.fetch = async (url, options) => {
    assert.equal(url, photo.uri);
    assert.equal(options.signal, controller.signal);
    assert.equal(options.headers, undefined);
    return new Response(bytes, { headers: { 'Content-Type': 'application/octet-stream' } });
  };

  const form = await createReceiptUpload(photo, controller.signal);
  assert.deepEqual([...form.keys()], ['image']);
  const image = form.get('image');
  assert.ok(image instanceof Blob);
  assert.equal(image.name, 'receipt.png');
  assert.equal(image.type, 'image/png');
  assert.deepEqual(new Uint8Array(await image.arrayBuffer()), bytes);

  const request = new Request('https://example.test/receipts', { method: 'POST', body: form });
  assert.match(request.headers.get('Content-Type'), /^multipart\/form-data; boundary=/);
});

test('rejects missing, empty, and oversized photo files', async () => {
  for (const [response, message] of [
    [new Response('missing', { status: 404 }), /Could not read this photo/],
    [new Response(new Blob([])), /photo is empty/],
    [new Response(new Blob([new Uint8Array((10 << 20) + 1)])), /exceeds 10 MiB/],
  ]) {
    globalThis.fetch = async () => response;
    await assert.rejects(createReceiptUpload(photo, new AbortController().signal), message);
  }
});

test('propagates cancellation while reading the local photo', async () => {
  const controller = new AbortController();
  controller.abort();
  globalThis.fetch = async (_url, { signal }) => {
    assert.equal(signal, controller.signal);
    signal.throwIfAborted();
  };

  await assert.rejects(createReceiptUpload(photo, controller.signal), { name: 'AbortError' });
});
