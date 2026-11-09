import { router, useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';

import type { AuthenticatedUser } from '@/lib/api';
import { addCategory, deleteCategory, getCategories, type Category, type CategoryInput } from '@/lib/categories-api';
import { GroupUnavailableError } from '@/lib/groups-api';

export function useGroupCategories(user: AuthenticatedUser, groupID: string) {
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [removingID, setRemovingID] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const cancelLoad = useRef<(() => void) | null>(null);
  const saveRequest = useRef<AbortController | null>(null);

  useEffect(() => () => {
    saveRequest.current?.abort();
    saveRequest.current = null;
  }, []);

  const reload = useCallback(() => {
    if (saveRequest.current) return;

    cancelLoad.current?.();
    const controller = new AbortController();
    setLoading(true);
    setError(null);

    const timeout = setTimeout(() => {
      controller.abort();
      setLoading(false);
      setError('Loading categories timed out. Please retry.');
    }, 15000);

    getCategories(user, groupID, controller.signal)
      .then((loaded) => {
        if (!controller.signal.aborted) setCategories(loaded);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;

        if (cause instanceof GroupUnavailableError) {
          router.dismissTo('/groups');
          return;
        }

        setError(cause instanceof Error ? cause.message : 'Could not load categories.');
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

  async function save(input: CategoryInput): Promise<boolean> {
    if (saveRequest.current) return false;

    cancelLoad.current?.();
    setLoading(false);
    const controller = new AbortController();
    saveRequest.current = controller;
    setSaving(true);
    setSaveError(null);
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      const category = await addCategory(user, groupID, input, controller.signal);
      if (controller.signal.aborted) return false;

      setCategories((previous) => [...previous.filter((item) => item.id !== category.id), category]);
      return true;
    } catch (cause) {
      if (saveRequest.current !== controller) return false;

      if (cause instanceof GroupUnavailableError) {
        router.dismissTo('/groups');
        return false;
      }

      let message = 'Could not add this category.';

      if (controller.signal.aborted) {
        message = 'Saving could not be confirmed. Close this window and refresh categories before retrying.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setSaveError(message);
      return false;
    } finally {
      clearTimeout(timeout);

      if (saveRequest.current === controller) {
        saveRequest.current = null;
        setSaving(false);
      }
    }
  }

  async function remove(categoryID: string) {
    if (saveRequest.current) return;

    cancelLoad.current?.();
    setLoading(false);
    const controller = new AbortController();
    saveRequest.current = controller;
    setRemovingID(categoryID);
    setError(null);
    const timeout = setTimeout(() => controller.abort(), 15000);

    try {
      await deleteCategory(user, groupID, categoryID, controller.signal);
      if (controller.signal.aborted) return;

      setCategories((previous) => previous.filter((category) => category.id !== categoryID));
    } catch (cause) {
      if (saveRequest.current !== controller) return;

      if (cause instanceof GroupUnavailableError) {
        router.dismissTo('/groups');
        return;
      }

      let message = 'Could not delete this category.';

      if (controller.signal.aborted) {
        message = 'Deletion could not be confirmed. Refresh categories before retrying.';
      } else if (cause instanceof Error) {
        message = cause.message;
      }

      setError(message);
    } finally {
      clearTimeout(timeout);

      if (saveRequest.current === controller) {
        saveRequest.current = null;
        setRemovingID(null);
      }
    }
  }

  function clearSaveError() {
    setSaveError(null);
  }

  return { categories, loading, error, removingID, remove, saving, saveError, reload, save, clearSaveError };
}
