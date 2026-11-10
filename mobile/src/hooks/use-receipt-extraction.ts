import { useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { extractReceipt, type ReceiptDraft, type ReceiptPhoto } from '@/lib/receipts-api';

export function useReceiptExtraction(user: AuthenticatedUser, groupID: string) {
  const [photo, setPhoto] = useState<ReceiptPhoto | null>(null);
  const [draft, setDraft] = useState<ReceiptDraft | null>(null);
  const [scanning, setScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);

  useEffect(() => () => {
    request.current?.abort();
    request.current = null;
  }, []);

  function cancel() {
    request.current?.abort();
    request.current = null;
    setScanning(false);
  }

  function choosePhoto(value: ReceiptPhoto) {
    cancel();
    setPhoto(value);
    setDraft(null);
    setError(null);
  }

  async function scan() {
    if (!photo || request.current) return;

    const controller = new AbortController();
    request.current = controller;
    setScanning(true);
    setDraft(null);
    setError(null);
    const timeout = setTimeout(() => controller.abort(), 150000);

    try {
      const result = await extractReceipt(user, groupID, photo, controller.signal);
      if (request.current === controller && !controller.signal.aborted) setDraft(result);
    } catch (cause) {
      if (request.current !== controller) return;

      setError(controller.signal.aborted
        ? 'Scanning took too long. Check your connection, then try again with a clearer or smaller photo.'
        : cause instanceof Error ? cause.message : 'Could not scan this receipt. Check your connection and try again.');
    } finally {
      clearTimeout(timeout);
      if (request.current === controller) {
        request.current = null;
        setScanning(false);
      }
    }
  }

  return { photo, draft, scanning, error, setError, choosePhoto, scan, cancel };
}
