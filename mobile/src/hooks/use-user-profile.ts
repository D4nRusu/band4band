import { useEffect, useRef, useState } from 'react';

import { completeProfile, getCurrentUser, type AuthenticatedUser, type UserProfile } from '@/lib/api';

export function useUserProfile(user: AuthenticatedUser) {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [checking, setChecking] = useState(true);
  const [saving, setSaving] = useState(false);
  const [username, setUsername] = useState('');
  const [usernameError, setUsernameError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const saveRequest = useRef<AbortController | null>(null);

  useEffect(() => {
    return () => {
      saveRequest.current?.abort();
      saveRequest.current = null;
    };
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    const timeout = setTimeout(() => {
      controller.abort();
      setChecking(false);
      setError('The backend request timed out. Check the API URL and connection, then retry.');
    }, 15000);

    getCurrentUser(user, controller.signal)
      .then((currentProfile) => {
        if (!controller.signal.aborted) setProfile(currentProfile);
      })
      .catch((cause: unknown) => {
        if (!controller.signal.aborted) {
          setError(cause instanceof Error ? cause.message : 'Could not reach the backend.');
        }
      })
      .finally(() => {
        clearTimeout(timeout);
        if (!controller.signal.aborted) setChecking(false);
      });

    return () => {
      clearTimeout(timeout);
      controller.abort();
    };
  }, [user, attempt]);

  async function saveUsername() {
    if (!/^[a-zA-Z0-9_]{3,30}$/.test(username)) {
      setUsernameError('Use 3–30 letters, numbers, or underscores.');
      return;
    }

    setSaving(true);
    setUsernameError(null);

    const controller = new AbortController();
    saveRequest.current = controller;
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      const saved = await completeProfile(user, username, controller.signal);
      if (!controller.signal.aborted) setProfile(saved);
    } catch (cause) {
      if (saveRequest.current !== controller) return;

      let message = 'Could not save your username. Please retry.';
      if (controller.signal.aborted) {
        message = 'Saving timed out. Retry or reload your profile to check whether it was saved.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setUsernameError(message);
    } finally {
      clearTimeout(timeout);

      if (saveRequest.current === controller) {
        saveRequest.current = null;
        setSaving(false);
      }
    }
  }

  function changeUsername(value: string) {
    setUsername(value);
    setUsernameError(null);
  }

  function reload() {
    setChecking(true);
    setError(null);
    setProfile(null);
    setAttempt((previous) => previous + 1);
  }

  return {
    profile,
    checking,
    saving,
    error,
    username,
    usernameError,
    changeUsername,
    saveUsername,
    reload,
  };
}
