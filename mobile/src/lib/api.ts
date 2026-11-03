export async function getCurrentUser(
  user: { getIdToken(): Promise<string> },
  signal?: AbortSignal,
): Promise<string> {
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
  const response = await fetch(`${baseUrl.replace(/\/+$/, '')}/api/users/me`, {
    headers: { Authorization: `Bearer ${firebaseIdToken}` },
    signal,
    redirect: 'error',
  });
  if (!response.ok) {
    throw new Error(response.status === 401
      ? 'The backend rejected the Firebase session. Sign out and sign in again.'
      : `The backend returned HTTP ${response.status}.`);
  }
  const body: unknown = await response.json();
  if (!body || typeof body !== 'object' || !('uid' in body)
    || typeof body.uid !== 'string' || !body.uid) {
    throw new Error('The backend returned an invalid user response.');
  }
  return body.uid;
}
