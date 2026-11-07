import { router } from 'expo-router';
import { useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { deleteGroup, leaveGroup, type Group } from '@/lib/groups-api';

export function useExitGroup(user: AuthenticatedUser, group: Group, isOwner: boolean) {
  const [working, setWorking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);

  useEffect(() => {
    return () => {
      request.current?.abort();
      request.current = null;
    };
  }, []);

  async function exitGroup() {
    if (request.current) return;

    const controller = new AbortController();
    request.current = controller;
    setWorking(true);
    setError(null);
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      if (isOwner) {
        await deleteGroup(user, group.id, controller.signal);
      } else {
        await leaveGroup(user, group.id, controller.signal);
      }

      if (!controller.signal.aborted) router.dismissTo('/groups');
    } catch (cause) {
      if (request.current !== controller) return;

      let message = isOwner ? 'Could not delete your group.' : 'Could not leave this group.';

      if (controller.signal.aborted) {
        message = 'The change could not be confirmed. Close settings and refresh the group before trying again.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setError(message);
    } finally {
      clearTimeout(timeout);

      if (request.current === controller) {
        request.current = null;
        setWorking(false);
      }
    }
  }

  return { working, error, exitGroup };
}
