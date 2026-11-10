import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError } from '@/lib/groups-api';

export class ListRejectedError extends Error {}

export type GroupList = {
  id: string;
  groupID: string;
  name: string;
  createdBy: string;
  createdAt: string;
};

export type ListItem = {
  id: string;
  listID: string;
  text: string;
  createdBy: string;
  createdAt: string;
  assigneeUID: string;
  assigneeName: string;
  rotationUIDs: string[];
  done: boolean;
  doneBy: string;
  doneByName: string;
  doneAt?: string;
  clearedAt?: string;
  followUpID: string;
};

export type ListItemInput = {
  text: string;
  assigneeUID: string;
  rotationUIDs: string[];
};

function listsPath(groupID: string) {
  return `/api/groups/${encodeURIComponent(groupID)}/lists`;
}

function itemsPath(groupID: string, listID: string) {
  return `${listsPath(groupID)}/${encodeURIComponent(listID)}/items`;
}

export async function getLists(user: AuthenticatedUser, groupID: string, signal?: AbortSignal): Promise<GroupList[]> {
  const response = await authenticatedRequest(user, listsPath(groupID), { signal });
  await checkListResponse(response);

  const body: unknown = await response.json();
  if (!Array.isArray(body) || !body.every(isGroupList)) throw new Error('The backend returned invalid lists.');

  return body;
}

export async function createList(user: AuthenticatedUser, groupID: string, name: string, signal?: AbortSignal): Promise<GroupList> {
  const response = await authenticatedRequest(user, listsPath(groupID), {
    method: 'POST',
    body: JSON.stringify({ name }),
    signal,
  });
  await checkListResponse(response);

  const body: unknown = await response.json();
  if (!isGroupList(body)) throw new Error('The backend returned an invalid list.');

  return body;
}

export async function deleteList(user: AuthenticatedUser, groupID: string, listID: string, signal?: AbortSignal): Promise<void> {
  const response = await authenticatedRequest(user, `${listsPath(groupID)}/${encodeURIComponent(listID)}`, { method: 'DELETE', signal });
  await checkListResponse(response);
}

export async function getListItems(user: AuthenticatedUser, groupID: string, listID: string, signal?: AbortSignal): Promise<ListItem[]> {
  const response = await authenticatedRequest(user, itemsPath(groupID, listID), { signal });
  await checkListResponse(response);

  const body: unknown = await response.json();
  if (!Array.isArray(body) || !body.every(isListItem)) throw new Error('The backend returned invalid list items.');

  return body;
}

export async function addListItem(
  user: AuthenticatedUser,
  groupID: string,
  listID: string,
  input: ListItemInput,
  signal?: AbortSignal,
): Promise<void> {
  const response = await authenticatedRequest(user, itemsPath(groupID, listID), {
    method: 'POST',
    body: JSON.stringify(input),
    signal,
  });
  await checkListResponse(response);
}

export async function setListItemDone(
  user: AuthenticatedUser,
  groupID: string,
  listID: string,
  itemID: string,
  done: boolean,
  signal?: AbortSignal,
): Promise<void> {
  const response = await authenticatedRequest(user, `${itemsPath(groupID, listID)}/${encodeURIComponent(itemID)}`, {
    method: 'PATCH',
    body: JSON.stringify({ done }),
    signal,
  });
  await checkListResponse(response);
}

export async function deleteListItem(
  user: AuthenticatedUser,
  groupID: string,
  listID: string,
  itemID: string,
  signal?: AbortSignal,
): Promise<void> {
  const response = await authenticatedRequest(user, `${itemsPath(groupID, listID)}/${encodeURIComponent(itemID)}`, {
    method: 'DELETE',
    signal,
  });
  await checkListResponse(response);
}

export async function clearDoneItems(user: AuthenticatedUser, groupID: string, listID: string, signal?: AbortSignal): Promise<void> {
  const response = await authenticatedRequest(user, `${listsPath(groupID)}/${encodeURIComponent(listID)}/clear-done`, {
    method: 'POST',
    signal,
  });
  await checkListResponse(response);
}

async function checkListResponse(response: Response) {
  if (response.ok) return;

  if (response.status === 401) {
    throw new ListRejectedError('Your sign-in session could not be verified. Sign out and sign in again before retrying.');
  }

  const body: unknown = await response.json().catch(() => null);
  const code = body && typeof body === 'object' && 'code' in body ? body.code : null;
  const message = body && typeof body === 'object' && 'message' in body && typeof body.message === 'string' ? body.message : '';

  if (code === 'invalid_list' || code === 'list_not_found' || code === 'list_forbidden') {
    throw new ListRejectedError(capitalize(message || 'The list request was rejected.'));
  }

  if (response.status === 403 || response.status === 404) {
    throw new GroupUnavailableError('This group is no longer available.');
  }

  if (response.status === 400) throw new ListRejectedError(capitalize(message || 'The list request is invalid.'));

  throw new Error('The server could not complete the list request.');
}

function capitalize(message: string): string {
  const normalized = `${message.charAt(0).toUpperCase()}${message.slice(1)}`;
  return /[.!?]$/.test(normalized) ? normalized : `${normalized}.`;
}

function isGroupList(value: unknown): value is GroupList {
  if (!value || typeof value !== 'object') return false;

  const list = value as Record<string, unknown>;
  return ['id', 'groupID', 'name', 'createdBy', 'createdAt'].every((field) => typeof list[field] === 'string');
}

function isListItem(value: unknown): value is ListItem {
  if (!value || typeof value !== 'object') return false;

  const item = value as Record<string, unknown>;
  const textFields = ['id', 'listID', 'text', 'createdBy', 'createdAt', 'assigneeUID', 'assigneeName', 'doneBy', 'doneByName', 'followUpID'];
  if (!textFields.every((field) => typeof item[field] === 'string')) return false;
  if (typeof item.done !== 'boolean') return false;
  if (!Array.isArray(item.rotationUIDs) || !item.rotationUIDs.every((uid) => typeof uid === 'string')) return false;

  return isOptionalDate(item.doneAt) && isOptionalDate(item.clearedAt);
}

function isOptionalDate(value: unknown): boolean {
  return value === undefined || (typeof value === 'string' && !Number.isNaN(Date.parse(value)));
}
