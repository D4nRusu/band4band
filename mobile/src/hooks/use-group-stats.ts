import { router } from 'expo-router';
import { useEffect, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError } from '@/lib/groups-api';
import { getGroupStats, type GroupStats } from '@/lib/stats-api';

export function useGroupStats(user: AuthenticatedUser, groupID: string) {
  const [stats, setStats] = useState<GroupStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Loading statistics timed out. Please retry.');
    }, 15000);

    getGroupStats(user, groupID, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setStats(loaded);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;

        if (cause instanceof GroupUnavailableError) {
          router.dismissTo('/groups');
          return;
        }

        setError(cause instanceof Error ? cause.message : 'Could not load statistics.');
      })
      .finally(() => {
        clearTimeout(timeout);
        if (!controller.signal.aborted) setLoading(false);
      });

    return () => {
      clearTimeout(timeout);
      controller.abort();
    };
  }, [user, groupID, attempt]);

  function reload() {
    setLoading(true);
    setError(null);
    setAttempt((previous) => previous + 1);
  }

  return { stats, loading, error, reload };
}
