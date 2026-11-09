import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError } from '@/lib/groups-api';

export const categoryIcons = {
  food: { image: '🍽️', label: 'Food & drinks' },
  groceries: { image: '🛒', label: 'Groceries' },
  transport: { image: '🚌', label: 'Transport' },
  accommodation: { image: '🏠', label: 'Accommodation' },
  utilities: { image: '💡', label: 'Utilities' },
  entertainment: { image: '🎬', label: 'Entertainment' },
  shopping: { image: '🛍️', label: 'Shopping' },
  health: { image: '💊', label: 'Health' },
  travel: { image: '✈️', label: 'Travel' },
  other: { image: '🏷️', label: 'Other' },
} as const;

export type CategoryIcon = keyof typeof categoryIcons;
export type CategoryInput = { name: string; icon: CategoryIcon };
export type Category = CategoryInput & { id: string; isDefault: boolean };

export async function getCategories(user: AuthenticatedUser, groupID: string, signal?: AbortSignal): Promise<Category[]> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/categories`, { signal });
  await checkResponse(response);

  const body: unknown = await response.json();
  if (!Array.isArray(body) || !body.every(isCategory)) throw new Error('The backend returned invalid categories.');

  return body;
}

export async function addCategory(
  user: AuthenticatedUser, groupID: string, input: CategoryInput, signal?: AbortSignal,
): Promise<Category> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/categories`, {
    method: 'POST', body: JSON.stringify(input), signal,
  });
  await checkResponse(response);

  const body: unknown = await response.json();
  if (!isCategory(body)) throw new Error('The backend returned an invalid category.');

  return body;
}

export async function deleteCategory(
  user: AuthenticatedUser, groupID: string, categoryID: string, signal?: AbortSignal,
): Promise<void> {
  const response = await authenticatedRequest(user,
    `/api/groups/${encodeURIComponent(groupID)}/categories/${encodeURIComponent(categoryID)}`, {
      method: 'DELETE', signal,
    });

  if (response.status === 400) throw new Error('This category cannot be deleted.');

  await checkResponse(response);
}

async function checkResponse(response: Response) {
  if (response.status === 409) throw new Error('That category name already exists. Choose a different name.');

  if (response.status === 400) throw new Error('Use a name of 1–40 characters and choose an icon.');

  if (response.status === 403 || response.status === 404) throw new GroupUnavailableError('This group is no longer available.');

  if (response.status === 401) throw new Error('Your session was rejected. Sign out and sign in again.');

  if (!response.ok) throw new Error('Could not complete the category request. Please retry.');
}

function isCategory(value: unknown): value is Category {
  if (!value || typeof value !== 'object') return false;

  const category = value as Record<string, unknown>;
  return typeof category.id === 'string' && category.id.length > 0
    && typeof category.name === 'string' && category.name.length > 0
    && typeof category.icon === 'string' && Object.hasOwn(categoryIcons, category.icon)
    && typeof category.isDefault === 'boolean';
}
