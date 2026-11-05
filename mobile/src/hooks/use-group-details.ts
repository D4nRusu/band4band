import { useFocusEffect } from 'expo-router';
import { useCallback, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { getGroup, type Group } from '@/lib/groups-api';

export function useGroupDetails(user: AuthenticatedUser, groupID: string) {
  const [group, setGroup] = useState<Group | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const cancelLoad = useRef<(() => void) | null>(null);

  const reload = useCallback(() => {
    cancelLoad.current?.();
    const controller = new AbortController();
    setLoading(true);
    setError(null);

    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setGroup(null);
      setError('Loading the group timed out. Please retry.');
    }, 15000);

    getGroup(user, groupID, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setGroup(loaded);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;

        setGroup(null);
        setError(cause instanceof Error ? cause.message : 'Could not load this group.');
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

  return { group, loading, error, reload };
}
