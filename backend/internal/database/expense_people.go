package database

import (
	"slices"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (s *GroupStore) expensePeople(tx *firestore.Transaction, group models.Group, input models.ExpenseInput) (string, []models.ExpenseContribution, error) {
	contributions, err := models.EqualContributions(input.AmountMinor, input.ParticipantUIDs)
	if err != nil {
		return "", nil, err
	}

	for _, contribution := range contributions {
		if !validDocumentID(contribution.UID) || !slices.Contains(group.MemberUIDs, contribution.UID) {
			return "", nil, models.ErrInvalidExpenseParticipants
		}
	}

	uids := []string{input.PaidBy}
	for _, contribution := range contributions {
		if contribution.UID != input.PaidBy {
			uids = append(uids, contribution.UID)
		}
	}

	names, err := s.userNames(tx, uids)
	if err != nil {
		return "", nil, err
	}

	for i := range contributions {
		contributions[i].Name = names[contributions[i].UID]
	}

	return names[input.PaidBy], contributions, nil
}

// userNames snapshots display names so history stays readable after profile changes.
func (s *GroupStore) userNames(tx *firestore.Transaction, uids []string) (map[string]string, error) {
	refs := make([]*firestore.DocumentRef, 0, len(uids))
	for _, uid := range uids {
		refs = append(refs, s.db.Collection("users").Doc(uid))
	}

	snapshots, err := tx.GetAll(refs)
	if err != nil {
		return nil, err
	}

	names := make(map[string]string, len(uids))
	for i, snapshot := range snapshots {
		uid := uids[i]
		names[uid] = uid
		if !snapshot.Exists() {
			continue
		}

		var profile models.User
		if err := snapshot.DataTo(&profile); err != nil {
			return nil, err
		}

		if profile.Name != "" {
			names[uid] = profile.Name
		} else if profile.Username != "" {
			names[uid] = profile.Username
		}
	}

	return names, nil
}
