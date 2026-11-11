import type { ReceiptPhoto } from '@/lib/receipts-api';

export async function createReceiptUpload(photo: ReceiptPhoto, signal: AbortSignal): Promise<FormData> {
  const response = await fetch(photo.uri, { signal });
  if (!response.ok) throw new Error('Could not read this photo. Take or choose the receipt photo again.');

  const blob = await response.blob();
  if (blob.size === 0) throw new Error('This photo is empty. Take or choose another receipt photo.');

  if (blob.size > 10 * 1024 * 1024) {
    throw new Error('This photo exceeds 10 MiB. Take or choose a smaller receipt photo.');
  }

  // Expo SDK 57 requires actual Blob/File data, not React Native's legacy URI parts.
  const body = new FormData();
  body.append('image', blob.slice(0, blob.size, photo.mimeType), photo.name);
  return body;
}
