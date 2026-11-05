import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';

export interface Group {
  id: string;
  name: string;
  description: string;
  createdBy: string;
  memberUIDs: string[];
  createdAt: string;
}

export interface GroupInput {
  name: string;
  description: string;
}

export async function getGroups(user: AuthenticatedUser, signal?: AbortSignal): Promise<Group[]> {
  const response = await authenticatedRequest(user, '/api/groups', { signal });
  await checkGroupResponse(response);

  const body: unknown = await response.json();
  if (!Array.isArray(body) || !body.every(isGroup)) {
    throw new Error('The backend returned an invalid groups response.');
  }

  return body;
}

export async function createGroup(
  user: AuthenticatedUser,
  input: GroupInput,
  signal?: AbortSignal,
): Promise<Group> {
  const response = await authenticatedRequest(user, '/api/groups', {
    method: 'POST',
    body: JSON.stringify(input),
    signal,
  });
  await checkGroupResponse(response);

  const body: unknown = await response.json();
  if (!isGroup(body)) {
    throw new Error('The backend returned an invalid group response.');
  }

  return body;
}

export async function deleteGroup(user: AuthenticatedUser, groupID: string, signal?: AbortSignal): Promise<void> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}`, {
    method: 'DELETE',
    signal,
  });

  // A stale list or a retried delete may refer to a group already removed.
  if (response.status === 404) return;

  await checkGroupResponse(response);
}

async function checkGroupResponse(response: Response): Promise<void> {
  if (response.status === 401) {
    throw new Error('Your session was rejected. Sign out and sign in again.');
  }

  if (response.status === 400) {
    throw new Error('Use a name of 1–100 characters and a description of at most 1,000 characters.');
  }

  if (response.status === 403) {
    throw new Error('Only the group owner can delete this group.');
  }

  if (!response.ok) {
    throw new Error(`Could not complete the group operation (HTTP ${response.status}).`);
  }
}

export async function leaveGroup(user: AuthenticatedUser, groupID: string, signal?: AbortSignal): Promise<void> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/leave`, {
    method: 'POST',
    signal,
  });

  if (response.status === 404) return;

  if (response.status === 403) {
    throw new Error('The owner cannot leave the group. Delete it instead.');
  }

  await checkGroupResponse(response);
}

function isGroup(value: unknown): value is Group {
  if (!value || typeof value !== 'object') return false;

  const group = value as Record<string, unknown>;
  const hasDetails = typeof group.id === 'string' && group.id.length > 0
    && typeof group.name === 'string' && group.name.length > 0
    && typeof group.description === 'string';
  const hasCreator = typeof group.createdBy === 'string' && group.createdBy.length > 0;
  const hasCreationDate = typeof group.createdAt === 'string'
    && Number.isFinite(Date.parse(group.createdAt));
  const hasMembers = Array.isArray(group.memberUIDs)
    && group.memberUIDs.every((uid) => typeof uid === 'string' && uid.length > 0);

  return hasDetails && hasCreator && hasCreationDate && hasMembers;
}
