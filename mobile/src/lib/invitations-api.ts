import { authenticatedRequest, type AuthenticatedUser } from '@/lib/api';

export interface GroupInvitation {
  id: string;
  groupID: string;
  groupName: string;
  inviterUID: string;
  inviterUsername: string;
  inviterName: string;
  createdAt: string;
}

export async function getInvitations(user: AuthenticatedUser, signal?: AbortSignal): Promise<GroupInvitation[]> {
  const response = await authenticatedRequest(user, '/api/invitations', { signal });
  await checkInvitationResponse(response);

  const body: unknown = await response.json();
  if (!Array.isArray(body) || !body.every(isInvitation)) {
    throw new Error('The backend returned invalid invitations.');
  }

  return body;
}

export async function sendGroupInvitation(
  user: AuthenticatedUser,
  groupID: string,
  username: string,
  signal?: AbortSignal,
): Promise<void> {
  const response = await authenticatedRequest(user, `/api/groups/${encodeURIComponent(groupID)}/invitations`, {
    method: 'POST',
    body: JSON.stringify({ username }),
    signal,
  });

  await checkInvitationResponse(response);
}

export async function respondToInvitation(
  user: AuthenticatedUser,
  invitationID: string,
  accept: boolean,
  signal?: AbortSignal,
): Promise<void> {
  const action = accept ? 'accept' : 'decline';
  const response = await authenticatedRequest(user, `/api/invitations/${encodeURIComponent(invitationID)}/${action}`, {
    method: 'POST',
    signal,
  });

  await checkInvitationResponse(response);
}

async function checkInvitationResponse(response: Response): Promise<void> {
  if (response.ok) return;

  if (response.status === 401) {
    throw new Error('Your session was rejected. Sign out and sign in again.');
  }

  const body: unknown = await response.json();
  const code = body && typeof body === 'object' && 'code' in body ? body.code : null;
  const messages: Record<string, string> = {
    user_not_found: 'No user has that username. Check the spelling and try again.',
    already_member: 'That user is already in this group.',
    invitation_not_found: 'This invitation is no longer available. Refresh your invitations.',
    invitation_forbidden: 'Only the person invited can respond to this invitation.',
    group_membership_forbidden: 'You must be a current group member to invite someone.',
    group_not_found: 'This group no longer exists. Refresh your groups.',
    invalid_invitation: 'Enter a username of 3–30 letters, numbers, or underscores.',
  };

  if (typeof code === 'string' && Object.hasOwn(messages, code)) {
    throw new Error(messages[code]);
  }

  throw new Error(`Could not complete the invitation request (HTTP ${response.status}).`);
}

function isInvitation(value: unknown): value is GroupInvitation {
  if (!value || typeof value !== 'object') return false;

  const invitation = value as Record<string, unknown>;
  const hasIDs = typeof invitation.id === 'string' && invitation.id.length > 0
    && typeof invitation.groupID === 'string' && invitation.groupID.length > 0
    && typeof invitation.inviterUID === 'string' && invitation.inviterUID.length > 0;
  const hasDisplayFields = typeof invitation.groupName === 'string'
    && typeof invitation.inviterUsername === 'string'
    && typeof invitation.inviterName === 'string';
  const hasCreationDate = typeof invitation.createdAt === 'string'
    && Number.isFinite(Date.parse(invitation.createdAt));

  return hasIDs && hasDisplayFields && hasCreationDate;
}
