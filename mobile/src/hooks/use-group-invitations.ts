import { useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';
import { AppState } from 'react-native';

import type { AuthenticatedUser } from '@/lib/api';
import { getInvitations, respondToInvitation, type GroupInvitation } from '@/lib/invitations-api';

export function useGroupInvitations(user: AuthenticatedUser, onMembershipChanged: () => void) {
  const [invitations, setInvitations] = useState<GroupInvitation[]>([]);
  const [loading, setLoading] = useState(true);
  const [respondingID, setRespondingID] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const cancelLoad = useRef<(() => void) | null>(null);
  const responseRequest = useRef<AbortController | null>(null);

  const reload = useCallback((showLoading = true) => {
    if (responseRequest.current) return;

    cancelLoad.current?.();
    const controller = new AbortController();

    if (showLoading) setLoading(true);

    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Loading invitations timed out. Pull to refresh and retry.');
    }, 15000);

    getInvitations(user, controller.signal)
      .then((loaded) => {
        if (controller.signal.aborted) return;

        setInvitations(loaded);
        setError(null);
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setError(cause instanceof Error ? cause.message : 'Could not load invitations.');
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
    function refreshWhileActive() {
      if (AppState.currentState === 'active') reload(false);
    }

    refreshWhileActive();
    const interval = setInterval(refreshWhileActive, 15000);
    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active') refreshWhileActive();
      else cancelLoad.current?.();
    });

    return () => {
      clearInterval(interval);
      subscription.remove();
      cancelLoad.current?.();
    };
  }, [reload]));

  useEffect(() => {
    return () => {
      responseRequest.current?.abort();
      responseRequest.current = null;
    };
  }, []);

  async function respond(invitationID: string, accept: boolean): Promise<void> {
    if (responseRequest.current) return;

    const controller = new AbortController();
    responseRequest.current = controller;
    cancelLoad.current?.();
    setLoading(false);
    setRespondingID(invitationID);
    setError(null);

    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      await respondToInvitation(user, invitationID, accept, controller.signal);
      if (controller.signal.aborted) return;

      setInvitations((previous) => previous.filter((invitation) => invitation.id !== invitationID));
      if (accept) onMembershipChanged();
    } catch (cause) {
      if (responseRequest.current !== controller) return;

      let message = 'Could not respond to this invitation.';

      if (controller.signal.aborted) {
        message = 'The response could not be confirmed. Pull to refresh before trying again.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setError(message);
      onMembershipChanged();
    } finally {
      clearTimeout(timeout);

      if (responseRequest.current === controller) {
        responseRequest.current = null;
        setRespondingID(null);
      }
    }
  }

  return { invitations, loading, error, respondingID, reload, respond };
}
