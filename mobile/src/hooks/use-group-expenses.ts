import { router, useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { createExpense, ExpenseRejectedError, getExpenses, type Expense, type ExpenseInput } from '@/lib/expenses-api';
import { GroupUnavailableError } from '@/lib/groups-api';

export function useGroupExpenses(user: AuthenticatedUser, groupID: string) {
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const cancelLoad = useRef<(() => void) | null>(null);
  const request = useRef<AbortController | null>(null);

  useEffect(() => () => {
    request.current?.abort();
    request.current = null;
  }, []);

  const reload = useCallback(() => {
    if (request.current) return;

    cancelLoad.current?.();
    const controller = new AbortController();
    setLoading(true);
    setError(null);
    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Expenses took too long to load. Check your connection, then tap Retry expenses.');
    }, 15000);

    getExpenses(user, groupID, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setExpenses(loaded);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;

        if (cause instanceof GroupUnavailableError) {
          router.dismissTo('/groups');
          return;
        }

        setError(cause instanceof Error ? cause.message : 'Could not load expenses. Check your connection, then tap Retry expenses.');
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

  async function save(input: ExpenseInput): Promise<boolean> {
    if (request.current) return false;

    cancelLoad.current?.();
    setLoading(false);
    const controller = new AbortController();
    request.current = controller;
    setSaving(true);
    setSaveError(null);
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      const expense = await createExpense(user, groupID, input, controller.signal);
      if (controller.signal.aborted) return false;

      setExpenses((previous) => [expense, ...previous.filter((item) => item.id !== expense.id)]);
      return true;
    } catch (cause) {
      if (request.current !== controller) return false;

      if (cause instanceof GroupUnavailableError) {
        router.dismissTo('/groups');
        return false;
      }

      if (cause instanceof ExpenseRejectedError) {
        setSaveError(cause.message);
        return false;
      }

      const message = cause instanceof Error ? cause.message : 'The connection was interrupted while saving.';
      setSaveError(controller.signal.aborted
        ? 'Saving took too long, so we could not confirm whether it finished. Use the back arrow and refresh expenses before trying again to avoid a duplicate.'
        : `${message} We could not confirm whether the expense was saved. Use the back arrow and refresh expenses before trying again.`);
      return false;
    } finally {
      clearTimeout(timeout);

      if (request.current === controller) {
        request.current = null;
        setSaving(false);
      }
    }
  }

  function removeFromList(expenseID: string) {
    cancelLoad.current?.();
    setLoading(false);
    setExpenses((previous) => previous.filter((expense) => expense.id !== expenseID));
  }

  function clearSaveError() {
    setSaveError(null);
  }

  return { expenses, loading, error, saving, saveError, reload, save, clearSaveError, removeFromList };
}
