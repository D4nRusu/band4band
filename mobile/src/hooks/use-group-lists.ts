import { router, useFocusEffect } from 'expo-router';
import { useCallback, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError } from '@/lib/groups-api';
import { createList, deleteList, getLists, ListRejectedError, type GroupList } from '@/lib/lists-api';

export function useGroupLists(user: AuthenticatedUser, groupID: string) {
  const [lists, setLists] = useState<GroupList[]>([]);
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
      setError('Lists took too long to load. Check your connection, then retry.');
    }, 15000);

    getLists(user, groupID, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setLists(loaded);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;

        if (cause instanceof GroupUnavailableError) {
          router.dismissTo('/groups');
          return;
        }

        setError(cause instanceof Error ? cause.message : 'Could not load lists.');
      })
      .finally(() => {
        clearTimeout(timeout);
        if (!controller.signal.aborted) setLoading(false);
      });

    cancelLoad.current = () => {
      clearTimeout(timeout);
      controller.abort();
    };
  }, [user, groupID]);

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

      setSaveError(cause instanceof ListRejectedError || cause instanceof Error ? cause.message : 'The list could not be saved.');
      return false;
    } finally {
      setSaving(false);
    }
  }

  function create(name: string) {
    return run(() => createList(user, groupID, name));
  }

  function remove(listID: string) {
    return run(() => deleteList(user, groupID, listID));
  }

  return { lists, loading, error, saving, saveError, reload, create, remove };
}
