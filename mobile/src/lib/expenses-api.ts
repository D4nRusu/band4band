import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';
import { categoryIcons, type CategoryIcon } from '@/lib/categories-api';
import { GroupUnavailableError } from '@/lib/groups-api';

export class ExpenseRejectedError extends Error {}

type ExpenseFields = {
  name: string;
  categoryID: string;
  paidBy: string;
  amountMinor: number;
};

export type ExpenseShareInput = { uid: string; amountMinor: number };
export type ExpenseSplit =
  | { splitType?: 'equal'; participantUIDs: string[] }
  | { splitType: 'unequal'; contributions: ExpenseShareInput[] };
export type ExpenseInput = ExpenseFields & ExpenseSplit;

export type ExpenseContribution = {
  uid: string;
  name: string;
  amountMinor: number;
};

export type ExpensePosition = {
  kind: 'lent' | 'borrowed' | 'none' | 'not_involved' | 'unallocated';
  amountMinor: number;
};

export type Expense = ExpenseFields & {
  viewerPosition: ExpensePosition;
  splitType?: 'equal' | 'unequal';
  contributions?: ExpenseContribution[];
  id: string;
  groupID: string;
  categoryName: string;
  categoryIcon: CategoryIcon;
  payerName: string;
  currency: string;
  createdBy: string;
  createdAt: string;
};

export async function getExpenses(user: AuthenticatedUser, groupID: string, signal?: AbortSignal): Promise<Expense[]> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/expenses`, { signal });
  await checkResponse(response);

  const body: unknown = await response.json();
  if (!Array.isArray(body) || !body.every(isExpense)) throw new Error('The backend returned invalid expenses.');

  return body;
}

export async function createExpense(
  user: AuthenticatedUser, groupID: string, input: ExpenseInput, signal?: AbortSignal,
): Promise<Expense> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/expenses`, {
    method: 'POST', body: JSON.stringify(input), signal,
  });
  await checkResponse(response);

  const body: unknown = await response.json();
  if (!isExpense(body)) throw new Error('The backend returned an invalid expense. Refresh the list before retrying.');

  return body;
}

export async function deleteExpense(
  user: AuthenticatedUser, groupID: string, expenseID: string, signal?: AbortSignal,
): Promise<void> {
  const response = await authenticatedRequest(user,
    `/api/groups/${encodeURIComponent(groupID)}/expenses/${encodeURIComponent(expenseID)}`, {
      method: 'DELETE', signal,
    });
  await checkResponse(response);
}

async function checkResponse(response: Response) {
  if (response.status === 403 || response.status === 404) {
    throw new GroupUnavailableError('This group is no longer available.');
  }

  if (response.status === 401) throw new ExpenseRejectedError('Your sign-in session could not be verified. Sign out and sign in again before retrying.');

  if (response.status === 400) {
    const body: unknown = await response.json();
    if (body && typeof body === 'object' && 'code' in body && body.code === 'invalid_expense'
      && 'message' in body && typeof body.message === 'string') {
      throw new ExpenseRejectedError(`${body.message.charAt(0).toUpperCase()}${body.message.slice(1)}.`);
    }

    throw new ExpenseRejectedError('Check the name and amount, then select a category, payer, and at least one contributor.');
  }

  if (!response.ok) throw new Error('The server could not complete the expense request.');
}

function isExpense(value: unknown): value is Expense {
  if (!value || typeof value !== 'object') return false;

  const expense = value as Record<string, unknown>;
  const hasIdentity = ['id', 'groupID', 'name', 'categoryID', 'categoryName', 'paidBy', 'payerName', 'createdBy']
    .every((key) => typeof expense[key] === 'string' && expense[key].length > 0);
  const hasAmount = typeof expense.amountMinor === 'number'
    && Number.isSafeInteger(expense.amountMinor) && expense.amountMinor > 0;
  const hasCategoryIcon = typeof expense.categoryIcon === 'string' && Object.hasOwn(categoryIcons, expense.categoryIcon);
  const hasDate = typeof expense.createdAt === 'string' && Number.isFinite(Date.parse(expense.createdAt));

  return hasIdentity && hasAmount && hasCategoryIcon && hasDate && expense.currency === 'EUR'
    && hasValidContributions(expense) && hasValidPosition(expense);
}

function hasValidContributions(expense: Record<string, unknown>): boolean {
  if (expense.splitType === undefined && expense.contributions === undefined) return true;

  if ((expense.splitType !== 'equal' && expense.splitType !== 'unequal') || !Array.isArray(expense.contributions)
    || expense.contributions.length === 0 || !expense.contributions.every(isContribution)) return false;

  const contributions = expense.contributions;
  const uniqueUIDs = new Set(contributions.map((contribution) => contribution.uid));
  const total = contributions.reduce((sum, contribution) => sum + BigInt(contribution.amountMinor), 0n);

  return uniqueUIDs.size === contributions.length && total === BigInt(expense.amountMinor as number);
}

function isContribution(value: unknown): value is ExpenseContribution {
  if (!value || typeof value !== 'object') return false;

  const contribution = value as Record<string, unknown>;
  return typeof contribution.uid === 'string' && contribution.uid.length > 0
    && typeof contribution.name === 'string' && contribution.name.length > 0
    && typeof contribution.amountMinor === 'number'
    && Number.isSafeInteger(contribution.amountMinor) && contribution.amountMinor >= 0;
}

function hasValidPosition(expense: Record<string, unknown>): boolean {
  const position = expense.viewerPosition;
  if (!position || typeof position !== 'object') return false;

  const value = position as Record<string, unknown>;
  if (typeof value.amountMinor !== 'number' || !Number.isSafeInteger(value.amountMinor)
    || value.amountMinor < 0 || value.amountMinor > (expense.amountMinor as number)) return false;

  if (value.kind === 'lent' || value.kind === 'borrowed') return value.amountMinor > 0;

  return ['none', 'not_involved', 'unallocated'].includes(value.kind as string) && value.amountMinor === 0;
}
