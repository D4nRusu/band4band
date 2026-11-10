import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';
import { GroupUnavailableError } from '@/lib/groups-api';

export class SettlementRejectedError extends Error {}

export type SettleUpInput = {
  fromUID: string;
  toUID: string;
  amountMinor: number;
};

export type Settlement = {
  id: string;
  groupID: string;
  fromUID: string;
  fromName: string;
  toUID: string;
  toName: string;
  amountMinor: number;
  currency: string;
  createdBy: string;
  createdAt: string;
};

export type UserOwedDebt = {
  toUID: string;
  amountMinor: number;
};

export type UserOwedSummary = {
  userUID: string;
  currency: string;
  totalAmountMinor: number;
  debts: UserOwedDebt[];
};

export type UserReceivableDebt = {
  fromUID: string;
  amountMinor: number;
};

export type UserReceivableSummary = {
  userUID: string;
  currency: string;
  totalAmountMinor: number;
  debts: UserReceivableDebt[];
};

export async function getUserOwedSummary(
  user: AuthenticatedUser,
  groupID: string,
  userID: string,
  signal?: AbortSignal,
): Promise<UserOwedSummary> {
  const response = await authenticatedRequest(
    user,
    `/api/groups/${encodeURIComponent(groupID)}/users/${encodeURIComponent(userID)}/owed`,
    { signal },
  );
  await checkSettlementResponse(response);

  const body: unknown = await response.json();
  if (!isUserOwedSummary(body)) throw new Error('The backend returned an invalid debt summary.');

  return body;
}

export async function getSettlements(
  user: AuthenticatedUser,
  groupID: string,
  signal?: AbortSignal,
): Promise<Settlement[]> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/settlements`, { signal });
  await checkSettlementResponse(response);

  const body: unknown = await response.json();
  if (!Array.isArray(body) || !body.every(isSettlement)) throw new Error('The backend returned an invalid settlement history.');

  return body;
}

export async function settleUp(
  user: AuthenticatedUser,
  groupID: string,
  input: SettleUpInput,
  signal?: AbortSignal,
): Promise<void> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/settlements`, {
    method: 'POST',
    body: JSON.stringify(input),
    signal,
  });
  await checkSettlementResponse(response);
}

export async function getUserReceivableSummary(
  user: AuthenticatedUser,
  groupID: string,
  userID: string,
  signal?: AbortSignal,
): Promise<UserReceivableSummary> {
  const response = await authenticatedRequest(
    user,
    `/api/groups/${encodeURIComponent(groupID)}/users/${encodeURIComponent(userID)}/receivable`,
    { signal },
  );
  await checkSettlementResponse(response);

  const body: unknown = await response.json();
  if (!isUserReceivableSummary(body)) throw new Error('The backend returned an invalid receivable summary.');

  return body;
}

async function checkSettlementResponse(response: Response) {
  if (response.status === 403 || response.status === 404) {
    throw new GroupUnavailableError('This group is no longer available.');
  }

  if (response.status === 401) {
    throw new SettlementRejectedError('Your sign-in session could not be verified. Sign out and sign in again before retrying.');
  }

  if (response.status === 400) {
    const body: unknown = await response.json();
    if (body && typeof body === 'object' && 'message' in body && typeof body.message === 'string') {
      const message = body.message.trim();
      throw new SettlementRejectedError(message.length > 0 ? capitalizeMessage(message) : 'The settlement request is invalid.');
    }

    throw new SettlementRejectedError('The settlement request is invalid.');
  }

  if (!response.ok) throw new Error('The server could not complete the settlement request.');
}

function capitalizeMessage(message: string): string {
  if (message.length === 0) return message;

  const normalized = `${message.charAt(0).toUpperCase()}${message.slice(1)}`;
  return /[.!?]$/.test(normalized) ? normalized : `${normalized}.`;
}

function isUserOwedSummary(value: unknown): value is UserOwedSummary {
  if (!value || typeof value !== 'object') return false;

  const summary = value as Record<string, unknown>;
  if (typeof summary.userUID !== 'string' || summary.userUID.length === 0) return false;
  if (typeof summary.currency !== 'string' || summary.currency.length === 0) return false;
  if (!isSafeMinor(summary.totalAmountMinor) || summary.totalAmountMinor < 0) return false;
  if (!Array.isArray(summary.debts) || !summary.debts.every(isUserDebt)) return false;

  return true;
}

function isUserDebt(value: unknown): value is UserOwedDebt {
  if (!value || typeof value !== 'object') return false;

  const debt = value as Record<string, unknown>;
  return typeof debt.toUID === 'string' && debt.toUID.length > 0
    && isSafeMinor(debt.amountMinor) && debt.amountMinor > 0;
}

function isSafeMinor(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value);
}

function isUserReceivableSummary(value: unknown): value is UserReceivableSummary {
  if (!value || typeof value !== 'object') return false;

  const summary = value as Record<string, unknown>;
  if (typeof summary.userUID !== 'string' || summary.userUID.length === 0) return false;
  if (typeof summary.currency !== 'string' || summary.currency.length === 0) return false;
  if (!isSafeMinor(summary.totalAmountMinor) || summary.totalAmountMinor < 0) return false;
  if (!Array.isArray(summary.debts) || !summary.debts.every(isUserReceivableDebt)) return false;

  return true;
}

function isUserReceivableDebt(value: unknown): value is UserReceivableDebt {
  if (!value || typeof value !== 'object') return false;

  const debt = value as Record<string, unknown>;
  return typeof debt.fromUID === 'string' && debt.fromUID.length > 0
    && isSafeMinor(debt.amountMinor) && debt.amountMinor > 0;
}

function isSettlement(value: unknown): value is Settlement {
  if (!value || typeof value !== 'object') return false;

  const settlement = value as Record<string, unknown>;
  const textFields = ['id', 'groupID', 'fromUID', 'fromName', 'toUID', 'toName', 'currency', 'createdBy', 'createdAt'];
  if (!textFields.every((field) => typeof settlement[field] === 'string')) return false;
  if (!isSafeMinor(settlement.amountMinor) || settlement.amountMinor <= 0) return false;

  return !Number.isNaN(Date.parse(settlement.createdAt as string));
}
