import { router, useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { createGroup, deleteGroup, getGroups, leaveGroup, type Group, type GroupInput } from '@/lib/groups-api';

export function useGroups(user: AuthenticatedUser) {
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [removingID, setRemovingID] = useState<string | null>(null);
  const createRequest = useRef<AbortController | null>(null);
  const removeRequest = useRef<AbortController | null>(null);
  const cancelLoad = useRef<(() => void) | null>(null);

  useEffect(() => {
    return () => {
      createRequest.current?.abort();
      createRequest.current = null;
      removeRequest.current?.abort();
      removeRequest.current = null;
    };
  }, []);

  const reload = useCallback(() => {
    if (removeRequest.current) return;

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
    if (createRequest.current || removeRequest.current) return false;

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

  async function removeGroupFromList(groupID: string, action: 'delete' | 'leave'): Promise<void> {
    if (removeRequest.current || createRequest.current) return;

    const controller = new AbortController();
    removeRequest.current = controller;
    cancelLoad.current?.();
    setLoading(false);
    setRemovingID(groupID);
    setError(null);

    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      if (action === 'delete') {
        await deleteGroup(user, groupID, controller.signal);
      } else {
        await leaveGroup(user, groupID, controller.signal);
      }

      if (controller.signal.aborted) return;

      setGroups((previous) => previous.filter((group) => group.id !== groupID));

      router.dismissTo('/groups');
    } catch (cause) {
      if (removeRequest.current !== controller) return;

      let message = action === 'delete' ? 'Could not delete your group.' : 'Could not leave this group.';

      if (controller.signal.aborted) {
        message = 'The change could not be confirmed. Refresh your groups before trying again.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setError(message);
    } finally {
      clearTimeout(timeout);

      if (removeRequest.current === controller) {
        removeRequest.current = null;
        setRemovingID(null);
      }
    }
  }

  function removeGroup(groupID: string) {
    return removeGroupFromList(groupID, 'delete');
  }

  function leave(groupID: string) {
    return removeGroupFromList(groupID, 'leave');
  }

  return {
    groups, loading, error, creating, createError, removingID,
    submitGroup, removeGroup, leave, reload, clearCreateError,
  };
}
