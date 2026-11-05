package database

import (
	"context"
	"slices"
	"sort"

	"cloud.google.com/go/firestore"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (s *InvitationStore) ListForUser(ctx context.Context, uid string) ([]models.GroupInvitation, error) {
	snapshots, err := s.db.Collection("groupInvitations").Where("recipientUID", "==", uid).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	invitations := make([]models.GroupInvitation, 0, len(snapshots))
	refs := make([]*firestore.DocumentRef, 0, len(snapshots))

	for _, snapshot := range snapshots {
		var invitation models.GroupInvitation
		if err := snapshot.DataTo(&invitation); err != nil {
			return nil, err
		}

		if !validDocumentID(invitation.GroupID) {
			continue
		}

		invitation.ID = snapshot.Ref.ID
		invitations = append(invitations, invitation)
		refs = append(refs, s.db.Collection("groups").Doc(invitation.GroupID))
	}

	if len(refs) == 0 {
		return []models.GroupInvitation{}, nil
	}

	groups, err := s.db.GetAll(ctx, refs)
	if err != nil {
		return nil, err
	}

	visible := make([]models.GroupInvitation, 0, len(invitations))

	for i, snapshot := range groups {
		if !snapshot.Exists() {
			continue
		}

		var group models.Group
		if err := snapshot.DataTo(&group); err != nil {
			return nil, err
		}

		invitation := invitations[i]
		if group.PendingInvitations[uid] != invitation.ID || slices.Contains(group.MemberUIDs, uid) {
			continue
		}

		invitation.GroupName = group.Name
		visible = append(visible, invitation)
	}

	sort.Slice(visible, func(i, j int) bool {
		return visible[i].CreatedAt.After(visible[j].CreatedAt)
	})

	return visible, nil
}
