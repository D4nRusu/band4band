export interface UserProfile {
  uid: string;
  email: string;
  name: string;
  username: string;
  photoURL: string;
  createdAt: string;
  profileComplete: boolean;
}

type AuthenticatedUser = { getIdToken(): Promise<string> };

export function getCurrentUser(user: AuthenticatedUser, signal?: AbortSignal): Promise<UserProfile> {
  return requestProfile(user, '/api/users/me', { signal });
}

export function completeProfile(
  user: AuthenticatedUser,
  username: string,
  signal?: AbortSignal,
): Promise<UserProfile> {
  return requestProfile(user, '/api/users/me/username', {
    method: 'PUT', body: JSON.stringify({ username }), signal,
  });
}

async function requestProfile(
  user: AuthenticatedUser,
  path: string,
  options: { method?: string; body?: string; signal?: AbortSignal },
): Promise<UserProfile> {
  const baseUrl = process.env.EXPO_PUBLIC_API_BASE_URL?.trim();
  if (!baseUrl) throw new Error('Set EXPO_PUBLIC_API_BASE_URL and restart Metro.');
  const url = new URL(baseUrl);
  if (url.protocol !== 'https:' && !(url.protocol === 'http:' && __DEV__)) {
    throw new Error('The API URL must use HTTPS (HTTP is allowed only in development).');
  }
  if (url.username || url.password || url.search || url.hash) {
    throw new Error('The API URL must not contain credentials, a query, or a fragment.');
  }
  // Firebase ID token, not the Google token used for the credential exchange.
  let firebaseIdToken: string;
  try {
    firebaseIdToken = await user.getIdToken();
  } catch {
    throw new Error('Could not obtain a Firebase ID token. Check your connection or sign in again.');
  }
  const response = await fetch(`${baseUrl.replace(/\/+$/, '')}${path}`, {
    ...options,
    headers: { Authorization: `Bearer ${firebaseIdToken}`, 'Content-Type': 'application/json' },
    redirect: 'error',
  });
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
  const body: unknown = await response.json();
  if (!body || typeof body !== 'object' || !('uid' in body)
    || typeof body.uid !== 'string' || !body.uid
    || !('email' in body) || typeof body.email !== 'string'
    || !('name' in body) || typeof body.name !== 'string'
    || !('username' in body) || typeof body.username !== 'string'
    || !('photoURL' in body) || typeof body.photoURL !== 'string'
    || !('createdAt' in body) || typeof body.createdAt !== 'string' || !Number.isFinite(Date.parse(body.createdAt))
    || !('profileComplete' in body) || typeof body.profileComplete !== 'boolean'
    || (body.profileComplete ? !/^[a-z0-9_]{3,30}$/.test(body.username) : body.username !== '')) {
    throw new Error('The backend returned an invalid user response.');
  }
  return body as UserProfile;
}
