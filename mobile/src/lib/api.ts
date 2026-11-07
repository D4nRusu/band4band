export interface UserProfile {
  uid: string;
  email: string;
  name: string;
  username: string;
  photoURL: string;
  createdAt: string;
  profileComplete: boolean;
}

export type AuthenticatedUser = { getIdToken(): Promise<string> };

export function getCurrentUser(user: AuthenticatedUser, signal?: AbortSignal): Promise<UserProfile> {
  return requestProfile(user, '/api/users/me', { signal });
}

export function completeProfile(
  user: AuthenticatedUser,
  username: string,
  signal?: AbortSignal,
): Promise<UserProfile> {
  return requestProfile(user, '/api/users/me/username', {
    method: 'PUT',
    body: JSON.stringify({ username }),
    signal,
  });
}

async function requestProfile(
  user: AuthenticatedUser,
  path: string,
  options: { method?: string; body?: string; signal?: AbortSignal },
): Promise<UserProfile> {
  const response = await authenticatedRequest(user, path, options);

  await checkProfileResponse(response);

  const body: unknown = await response.json();
  if (!isUserProfile(body)) {
    throw new Error('The backend returned an invalid user response.');
  }

  return body;
}

export async function authenticatedRequest(
  user: AuthenticatedUser,
  path: string,
  options: { method?: string; body?: string; signal?: AbortSignal } = {},
): Promise<Response> {
  const baseUrl = getApiBaseUrl();
  const firebaseIdToken = await getFirebaseIdToken(user);

  return fetch(`${baseUrl}${path}`, {
    ...options,
    headers: {
      Authorization: `Bearer ${firebaseIdToken}`,
      'Content-Type': 'application/json',
    },
    redirect: 'error',
  });
}

function getApiBaseUrl(): string {
  const baseUrl = process.env.EXPO_PUBLIC_API_BASE_URL?.trim();
  if (!baseUrl) throw new Error('Set EXPO_PUBLIC_API_BASE_URL and restart Metro.');

  const url = new URL(baseUrl);
  if (url.protocol !== 'https:' && !(url.protocol === 'http:' && __DEV__)) {
    throw new Error('The API URL must use HTTPS (HTTP is allowed only in development).');
  }

  if (url.username || url.password || url.search || url.hash) {
    throw new Error('The API URL must not contain credentials, a query, or a fragment.');
  }

  return baseUrl.replace(/\/+$/, '');
}

async function getFirebaseIdToken(user: AuthenticatedUser): Promise<string> {
  // Firebase refreshes its ID token when needed; the Google token is never sent to Go.
  try {
    return await user.getIdToken();
  } catch {
    throw new Error('Could not obtain a Firebase ID token. Check your connection or sign in again.');
  }
}

async function checkProfileResponse(response: Response): Promise<void> {
  if (response.status === 409) {
    const conflict: unknown = await response.json();
    if (conflict && typeof conflict === 'object' && 'code' in conflict && conflict.code === 'username_taken') {
      throw new Error('That username is already taken. Please choose another.');
    }

    throw new Error('Your profile already has a username. Reload your profile.');
  }

  if (response.status === 400) {
    throw new Error('Use 3–30 letters, numbers, or underscores for your username.');
  }

  if (!response.ok) {
    throw new Error(response.status === 401
      ? 'The backend rejected the Firebase session. Sign out and sign in again.'
      : `The backend returned HTTP ${response.status}.`);
  }
}

function isUserProfile(value: unknown): value is UserProfile {
  if (!value || typeof value !== 'object') return false;

  const profile = value as Record<string, unknown>;
  const hasIdentity = typeof profile.uid === 'string' && profile.uid.length > 0;
  const hasDisplayFields = typeof profile.email === 'string'
    && typeof profile.name === 'string'
    && typeof profile.photoURL === 'string';
  const hasCreationDate = typeof profile.createdAt === 'string'
    && Number.isFinite(Date.parse(profile.createdAt));

  if (!hasIdentity || !hasDisplayFields || !hasCreationDate) return false;

  if (typeof profile.username !== 'string' || typeof profile.profileComplete !== 'boolean') {
    return false;
  }

  if (profile.profileComplete) {
    return /^[a-z0-9_]{3,30}$/.test(profile.username);
  }

  return profile.username === '';
}
