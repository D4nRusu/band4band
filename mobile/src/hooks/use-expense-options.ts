import { useEffect, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { getCategories, type Category } from '@/lib/categories-api';
import { getGroupMembers, type GroupMember } from '@/lib/groups-api';

export function useExpenseOptions(user: AuthenticatedUser, groupID: string) {
  const [categories, setCategories] = useState<Category[]>([]);
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Loading categories and members timed out. Please retry.');
    }, 15000);

    Promise.all([getCategories(user, groupID, controller.signal), getGroupMembers(user, groupID, controller.signal)])
      .then(([loadedCategories, loadedMembers]) => {
        if (controller.signal.aborted) return;

        setCategories(loadedCategories);
        setMembers(loadedMembers);
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) setError(cause instanceof Error ? cause.message : 'Could not load expense options.');
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
    setError(null);
    setLoading(true);
    setAttempt((previous) => previous + 1);
  }

  return { categories, members, loading, error, retry };
}
