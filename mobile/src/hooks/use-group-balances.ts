import { useEffect, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { getMemberBalances, type MemberBalance } from '@/lib/settlements-api';

export function useGroupBalances(user: AuthenticatedUser, groupID: string) {
  const [balances, setBalances] = useState<MemberBalance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Loading balances timed out. Please retry.');
    }, 15000);

    getMemberBalances(user, groupID, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setBalances(loaded);
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setError(cause instanceof Error ? cause.message : 'Could not load balances.');
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

  return { balances, loading, error, retry };
}
