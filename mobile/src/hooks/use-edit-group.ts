import { router } from 'expo-router';
import { useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError, updateGroup, type GroupInput } from '@/lib/groups-api';

export function useEditGroup(user: AuthenticatedUser, groupID: string) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);

  useEffect(() => () => {
    request.current?.abort();
    request.current = null;
  }, []);

  async function save(input: GroupInput) {
    if (request.current) return null;

    const controller = new AbortController();
    request.current = controller;
    setSaving(true);
    setError(null);
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      const group = await updateGroup(user, groupID, input, controller.signal);
      return controller.signal.aborted ? null : group;
    } catch (cause) {
      if (request.current !== controller) return null;

      if (cause instanceof GroupUnavailableError) {
        router.dismissTo('/groups');
        return null;
      }

      let message = 'Could not update this group.';

      if (controller.signal.aborted) {
        message = 'Saving could not be confirmed. Close this menu and refresh the group before retrying.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setError(message);
      return null;
    } finally {
      clearTimeout(timeout);

      if (request.current === controller) {
        request.current = null;
        setSaving(false);
      }
    }
  }

  return { saving, error, save };
}
