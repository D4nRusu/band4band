import { useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { createGroup, getGroups, type Group, type GroupInput } from '@/lib/groups-api';

export function useGroups(user: AuthenticatedUser) {
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const createRequest = useRef<AbortController | null>(null);
  const cancelLoad = useRef<(() => void) | null>(null);

  useEffect(() => {
    return () => {
      createRequest.current?.abort();
      createRequest.current = null;
    };
  }, []);

  const reload = useCallback(() => {
    cancelLoad.current?.();

    const controller = new AbortController();
    setLoading(true);
    setError(null);

    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Loading groups timed out. Check your connection and retry.');
    }, 15000);

    getGroups(user, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setGroups(loaded);
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setError(cause instanceof Error ? cause.message : 'Could not load your groups.');
        }
      })
      .finally(() => {
        clearTimeout(timeout);
        if (!controller.signal.aborted) setLoading(false);
      });

    cancelLoad.current = () => {
      clearTimeout(timeout);
      controller.abort();
    };
  }, [user]);

  useFocusEffect(useCallback(() => {
    reload();

    return () => cancelLoad.current?.();
  }, [reload]));

  async function submitGroup(input: GroupInput): Promise<boolean> {
    if (createRequest.current) return false;

    const controller = new AbortController();
    createRequest.current = controller;
    setCreating(true);
    setCreateError(null);

    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      const group = await createGroup(user, input, controller.signal);
      if (controller.signal.aborted) return false;

      setGroups((previous) => [group, ...previous.filter((item) => item.id !== group.id)]);
      return true;
    } catch (cause) {
      if (createRequest.current !== controller) return false;

      let message = 'Could not create your group.';

      if (controller.signal.aborted) {
        message = 'Creation could not be confirmed. Close this window and refresh your groups before trying again.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setCreateError(message);
      return false;
    } finally {
      clearTimeout(timeout);

      if (createRequest.current === controller) {
        createRequest.current = null;
        setCreating(false);
      }
    }
  }

  function clearCreateError() {
    setCreateError(null);
  }

  return {
    groups, loading, error, creating, createError,
    submitGroup, reload, clearCreateError,
  };
}
