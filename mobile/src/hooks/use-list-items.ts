import { router, useFocusEffect } from 'expo-router';
import { useCallback, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError } from '@/lib/groups-api';
import {
  addListItem,
  clearDoneItems,
  deleteListItem,
  getListItems,
  setListItemDone,
  type ListItem,
  type ListItemInput,
} from '@/lib/lists-api';

export function useListItems(user: AuthenticatedUser, groupID: string, listID: string) {
  const [items, setItems] = useState<ListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const cancelLoad = useRef<(() => void) | null>(null);

  const reload = useCallback(() => {
    cancelLoad.current?.();
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('The list took too long to load. Check your connection, then retry.');
    }, 15000);

    getListItems(user, groupID, listID, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setItems(loaded);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;

        if (cause instanceof GroupUnavailableError) {
          router.dismissTo('/groups');
          return;
        }

        setError(cause instanceof Error ? cause.message : 'Could not load the list.');
      })
      .finally(() => {
        clearTimeout(timeout);
        if (!controller.signal.aborted) setLoading(false);
      });

    cancelLoad.current = () => {
      clearTimeout(timeout);
      controller.abort();
    };
  }, [user, groupID, listID]);

  useFocusEffect(useCallback(() => {
    reload();
    return () => cancelLoad.current?.();
  }, [reload]));

  async function run(action: () => Promise<unknown>): Promise<boolean> {
    if (saving) return false;

    setSaving(true);
    setSaveError(null);
    try {
      await action();
      reload();
      return true;
    } catch (cause) {
      if (cause instanceof GroupUnavailableError) {
        router.dismissTo('/groups');
        return false;
      }

      setSaveError(cause instanceof Error ? cause.message : 'The change could not be saved.');
      reload();
      return false;
    } finally {
      setSaving(false);
    }
  }

  function add(input: ListItemInput) {
    return run(() => addListItem(user, groupID, listID, input));
  }

  function toggle(item: ListItem) {
    return run(() => setListItemDone(user, groupID, listID, item.id, !item.done));
  }

  function remove(itemID: string) {
    return run(() => deleteListItem(user, groupID, listID, itemID));
  }

  function clearDone() {
    return run(() => clearDoneItems(user, groupID, listID));
  }

  function clearSaveError() {
    setSaveError(null);
  }

  return { items, loading, error, saving, saveError, reload, add, toggle, remove, clearDone, clearSaveError };
}
