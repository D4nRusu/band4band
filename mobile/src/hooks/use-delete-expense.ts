import { router } from 'expo-router';
import { useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { deleteExpense, type Expense } from '@/lib/expenses-api';
import { GroupUnavailableError } from '@/lib/groups-api';

export function useDeleteExpense(user: AuthenticatedUser, expense: Expense) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef<AbortController | null>(null);

  useEffect(() => () => {
    request.current?.abort();
    request.current = null;
  }, []);

  async function remove(): Promise<boolean> {
    if (request.current) return false;

    const controller = new AbortController();
    request.current = controller;
    setDeleting(true);
    setError(null);
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      await deleteExpense(user, expense.groupID, expense.id, controller.signal);
      return !controller.signal.aborted;
    } catch (cause) {
      if (request.current !== controller) return false;

      if (cause instanceof GroupUnavailableError) {
        router.dismissTo('/groups');
        return false;
      }

      const message = cause instanceof Error ? cause.message : 'Could not delete this expense.';
      setError(controller.signal.aborted
        ? 'Deletion could not be confirmed. Retry or close this menu and refresh the expense list.'
        : message);
      return false;
    } finally {
      clearTimeout(timeout);

      if (request.current === controller) {
        request.current = null;
        setDeleting(false);
      }
    }
  }

  return { deleting, error, remove };
}
