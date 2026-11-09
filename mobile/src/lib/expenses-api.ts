import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';
import { categoryIcons, type CategoryIcon } from '@/lib/categories-api';
import { GroupUnavailableError } from '@/lib/groups-api';

export type ExpenseInput = {
  name: string;
  categoryID: string;
  paidBy: string;
  amountMinor: number;
};

export type Expense = ExpenseInput & {
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

async function checkResponse(response: Response) {
  if (response.status === 403 || response.status === 404) {
    throw new GroupUnavailableError('This group is no longer available.');
  }

  if (response.status === 401) throw new Error('Your session was rejected. Sign out and sign in again.');

  if (response.status === 400) {
    const body: unknown = await response.json();
    if (body && typeof body === 'object' && 'code' in body && body.code === 'invalid_expense'
      && 'message' in body && typeof body.message === 'string') {
      throw new Error(body.message);
    }

    throw new Error('Check the expense name, amount, category, and payer.');
  }

  if (!response.ok) throw new Error('The expense request could not be confirmed. Refresh before retrying.');
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

  return hasIdentity && hasAmount && hasCategoryIcon && hasDate && expense.currency === 'EUR';
}
