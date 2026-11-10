import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError } from '@/lib/groups-api';

export type ReceiptPhoto = { uri: string; mimeType: 'image/jpeg' | 'image/png'; name: string };
export type ReceiptItem = { id: string; name: string; amountMinor: number };
export type ReceiptDraft = {
  currency: string;
  totalMinor: number;
  itemsTotalMinor: number;
  totalsMatch: boolean;
  items: ReceiptItem[];
};

export async function extractReceipt(
  user: AuthenticatedUser, groupID: string, photo: ReceiptPhoto, signal: AbortSignal,
): Promise<ReceiptDraft> {
  const body = new FormData();
  // React Native accepts URI file parts; the DOM typings describe browser Blobs only.
  body.append('image', { uri: photo.uri, type: photo.mimeType, name: photo.name } as unknown as Blob);
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/receipts/extract`, {
    method: 'POST', body, signal,
  });

  if (response.status === 401) throw new Error('Your session could not be verified. Sign out and sign in again.');

  if (response.status === 403 || response.status === 404) {
    throw new GroupUnavailableError('This group is no longer available to you.');
  }

  if (!response.ok) {
    const messages: Record<number, string> = {
      400: 'Use a clear JPEG or PNG receipt photo under 25 megapixels.',
      413: 'This photo is too large. Choose a photo smaller than 10 MiB.',
      422: 'The receipt could not be read or its currency differs from the group. Try a clearer photo in the group currency.',
      429: 'Another receipt is being scanned. Wait a moment, then tap Scan receipt again.',
      502: 'The receipt reader returned incomplete or invalid items. Try again or use a clearer photo.',
      503: 'The receipt reader is temporarily unavailable. Please try again shortly.',
      504: 'Reading the receipt took too long. Try again with a clearer, smaller photo.',
    };
    throw new Error(messages[response.status] ?? 'Could not scan the receipt. Check your connection and try again.');
  }

  const draft: unknown = await response.json();
  if (!isReceiptDraft(draft)) throw new Error('The backend returned invalid receipt data. Try scanning again.');

  return draft;
}

function isReceiptDraft(value: unknown): value is ReceiptDraft {
  if (!value || typeof value !== 'object') return false;

  const draft = value as Record<string, unknown>;
  if (typeof draft.currency !== 'string' || !/^[A-Z]{3}$/.test(draft.currency)
    || typeof draft.totalMinor !== 'number' || !Number.isSafeInteger(draft.totalMinor) || draft.totalMinor <= 0
    || typeof draft.itemsTotalMinor !== 'number' || !Number.isSafeInteger(draft.itemsTotalMinor)
    || typeof draft.totalsMatch !== 'boolean'
    || !Array.isArray(draft.items) || draft.items.length < 1 || draft.items.length > 200) return false;

  if (!draft.items.every(isReceiptItem)) return false;

  const sum = draft.items.reduce((total, item) => total + BigInt(item.amountMinor), 0n);
  const uniqueIDs = new Set(draft.items.map((item) => item.id));
  return uniqueIDs.size === draft.items.length && sum === BigInt(draft.itemsTotalMinor)
    && draft.totalsMatch === (sum === BigInt(draft.totalMinor));
}

function isReceiptItem(value: unknown): value is ReceiptItem {
  if (!value || typeof value !== 'object') return false;

  const item = value as Record<string, unknown>;
  return typeof item.id === 'string' && item.id.length > 0
    && typeof item.name === 'string' && item.name.trim().length > 0 && Array.from(item.name).length <= 200
    && typeof item.amountMinor === 'number' && Number.isSafeInteger(item.amountMinor);
}
