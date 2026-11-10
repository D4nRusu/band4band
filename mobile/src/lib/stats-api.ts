import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';
import { categoryIcons, type CategoryIcon } from '@/lib/categories-api';
import { GroupUnavailableError } from '@/lib/groups-api';

export type CategoryTotal = {
  categoryID: string;
  categoryName: string;
  categoryIcon: CategoryIcon;
  amountMinor: number;
};

export type MemberShareTotal = {
  uid: string;
  name: string;
  amountMinor: number;
};

export type GroupStats = {
  totalAmountMinor: number;
  byCategory: CategoryTotal[];
  byMember: MemberShareTotal[];
  personalTotalAmountMinor: number;
  personalByCategory: CategoryTotal[];
};

export async function getGroupStats(user: AuthenticatedUser, groupID: string, signal?: AbortSignal): Promise<GroupStats> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/stats`, { signal });

  if (response.status === 403 || response.status === 404) {
    throw new GroupUnavailableError('This group is no longer available.');
  }

  if (response.status === 401) {
    throw new Error('Your sign-in session could not be verified. Sign out and sign in again before retrying.');
  }

  if (!response.ok) throw new Error('The server could not load group statistics.');

  const body: unknown = await response.json();
  if (!isGroupStats(body)) throw new Error('The backend returned invalid group statistics.');

  return body;
}

function isGroupStats(value: unknown): value is GroupStats {
  if (!value || typeof value !== 'object') return false;

  const stats = value as Record<string, unknown>;
  if (!isAmount(stats.totalAmountMinor) || !isAmount(stats.personalTotalAmountMinor)) return false;
  if (!Array.isArray(stats.byCategory) || !stats.byCategory.every(isCategoryTotal)) return false;
  if (!Array.isArray(stats.personalByCategory) || !stats.personalByCategory.every(isCategoryTotal)) return false;

  return Array.isArray(stats.byMember) && stats.byMember.every(isMemberShareTotal);
}

function isCategoryTotal(value: unknown): value is CategoryTotal {
  if (!value || typeof value !== 'object') return false;

  const total = value as Record<string, unknown>;
  return typeof total.categoryID === 'string'
    && typeof total.categoryName === 'string'
    && typeof total.categoryIcon === 'string' && Object.hasOwn(categoryIcons, total.categoryIcon)
    && isAmount(total.amountMinor);
}

function isMemberShareTotal(value: unknown): value is MemberShareTotal {
  if (!value || typeof value !== 'object') return false;

  const total = value as Record<string, unknown>;
  return typeof total.uid === 'string' && typeof total.name === 'string' && isAmount(total.amountMinor);
}

function isAmount(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;
}
