import { useEffect, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { getGroupMembers, type GroupMember } from '@/lib/groups-api';

export function useGroupMembers(user: AuthenticatedUser, groupID: string) {
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Loading members timed out. Please retry.');
    }, 15000);

    getGroupMembers(user, groupID, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setMembers(loaded);
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setError(cause instanceof Error ? cause.message : 'Could not load group members.');
        }
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

  function retry() {
    setLoading(true);
    setError(null);
    setAttempt((previous) => previous + 1);
  }

  return { members, loading, error, retry };
}
