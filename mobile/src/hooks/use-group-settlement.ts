import { router, useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError } from '@/lib/groups-api';
import {
  getSettlements,
  getUserReceivableSummary,
  getUserOwedSummary,
  SettlementRejectedError,
  settleUp,
  type Settlement,
  type SettleUpInput,
  type UserReceivableSummary,
  type UserOwedSummary,
} from '@/lib/settlements-api';

export function useGroupSettlement(user: AuthenticatedUser & { uid: string }, groupID: string) {
  const [owedSummary, setOwedSummary] = useState<UserOwedSummary | null>(null);
  const [receivableSummary, setReceivableSummary] = useState<UserReceivableSummary | null>(null);
  const [settlements, setSettlements] = useState<Settlement[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [settling, setSettling] = useState(false);
  const [settleError, setSettleError] = useState<string | null>(null);

  const cancelLoad = useRef<(() => void) | null>(null);
  const settleRequest = useRef<AbortController | null>(null);

  useEffect(() => () => {
    cancelLoad.current?.();
    settleRequest.current?.abort();
    settleRequest.current = null;
  }, []);

  const reload = useCallback(() => {
    cancelLoad.current?.();
    const controller = new AbortController();
    setLoading(true);
    setError(null);

    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Debt summary took too long to load. Check your connection, then retry.');
    }, 15000);

    Promise.all([
      getUserOwedSummary(user, groupID, user.uid, controller.signal),
      getUserReceivableSummary(user, groupID, user.uid, controller.signal),
      getSettlements(user, groupID, controller.signal),
    ])
      .then(([loadedOwed, loadedReceivable, loadedSettlements]) => {
        if (controller.signal.aborted) return;

        setOwedSummary(loadedOwed);
        setReceivableSummary(loadedReceivable);
        setSettlements(loadedSettlements);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;

        if (cause instanceof GroupUnavailableError) {
          router.dismissTo('/groups');
          return;
        }

        setError(cause instanceof Error ? cause.message : 'Could not load debt summary.');
      })
      .finally(() => {
        clearTimeout(timeout);
        if (!controller.signal.aborted) setLoading(false);
      });

    cancelLoad.current = () => {
      clearTimeout(timeout);
      controller.abort();
    };
  }, [groupID, user]);

  useFocusEffect(useCallback(() => {
    reload();
    return () => cancelLoad.current?.();
  }, [reload]));

  async function settle(input: SettleUpInput): Promise<boolean> {
    if (settleRequest.current) return false;

    const controller = new AbortController();
    settleRequest.current = controller;
    setSettling(true);
    setSettleError(null);
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      await settleUp(user, groupID, input, controller.signal);
      if (controller.signal.aborted) return false;

      const [refreshedOwed, refreshedReceivable, refreshedSettlements] = await Promise.all([
        getUserOwedSummary(user, groupID, user.uid, controller.signal),
        getUserReceivableSummary(user, groupID, user.uid, controller.signal),
        getSettlements(user, groupID, controller.signal),
      ]);
      if (!controller.signal.aborted) {
        setOwedSummary(refreshedOwed);
        setReceivableSummary(refreshedReceivable);
        setSettlements(refreshedSettlements);
      }

      return !controller.signal.aborted;
    } catch (cause) {
      if (settleRequest.current !== controller) return false;

      if (cause instanceof GroupUnavailableError) {
        router.dismissTo('/groups');
        return false;
      }

      if (cause instanceof SettlementRejectedError) {
        setSettleError(cause.message);
        return false;
      }

      const message = cause instanceof Error ? cause.message : 'The connection was interrupted while settling up.';
      setSettleError(controller.signal.aborted
        ? 'Settling up took too long, so we could not confirm whether it finished. Refresh this group before retrying to avoid duplicates.'
        : `${message} We could not confirm whether the settlement finished. Refresh this group before retrying.`);
      return false;
    } finally {
      clearTimeout(timeout);
      if (settleRequest.current === controller) {
        settleRequest.current = null;
        setSettling(false);
      }
    }
  }

  function clearSettleError() {
    setSettleError(null);
  }

  return {
    owedSummary,
    receivableSummary,
    settlements,
    loading,
    error,
    settling,
    settleError,
    reload,
    settle,
    clearSettleError,
  };
}
