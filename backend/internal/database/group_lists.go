package database

import (
	"context"
	"errors"
	"slices"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GroupStore) Lists(ctx context.Context, uid, groupID string) ([]models.GroupList, error) {
	if !validDocumentID(groupID) {
		return nil, models.ErrInvalidGroupID
	}

	var lists []models.GroupList
	err := s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := s.readMemberGroup(tx, uid, groupID); err != nil {
			return err
		}

		snapshots, err := tx.Documents(s.listsRef(groupID).Query).GetAll()
		if err != nil {
			return err
		}

		lists = make([]models.GroupList, 0, len(snapshots))
		for _, snapshot := range snapshots {
			list, err := decodeList(snapshot, groupID)
			if err != nil {
				return err
			}

			lists = append(lists, list)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(lists, func(i, j int) bool {
		if lists[i].CreatedAt.Equal(lists[j].CreatedAt) {
			return lists[i].ID < lists[j].ID
		}

		return lists[i].CreatedAt.Before(lists[j].CreatedAt)
	})

	return lists, nil
}

func (s *GroupStore) CreateList(ctx context.Context, uid, groupID string, input models.ListInput) (models.GroupList, error) {
	if !validDocumentID(groupID) {
		return models.GroupList{}, models.ErrInvalidGroupID
	}

	input, err := input.Normalize()
	if err != nil {
		return models.GroupList{}, err
	}

	ref := s.listsRef(groupID).NewDoc()
	err = s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := s.readMemberGroup(tx, uid, groupID); err != nil {
			return err
		}

		return tx.Create(ref, map[string]any{
			"name":      input.Name,
			"createdBy": uid,
			"createdAt": firestore.ServerTimestamp,
		})
	})
	if err != nil {
		return models.GroupList{}, err
	}

	snapshot, err := ref.Get(ctx)
	if err != nil {
		return models.GroupList{}, err
	}

	return decodeList(snapshot, groupID)
}

func (s *GroupStore) DeleteList(ctx context.Context, uid, groupID, listID string) error {
	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	if !validDocumentID(listID) {
		return models.ErrInvalidListID
	}

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := s.readMemberGroup(tx, uid, groupID); err != nil {
			return err
		}

		list, err := s.readList(tx, groupID, listID)
		if err != nil {
			return err
		}

		if list.CreatedBy != uid {
			return models.ErrListForbidden
		}

		items, err := tx.Documents(s.itemsRef(groupID, listID).Query).GetAll()
		if err != nil {
			return err
		}

		for _, item := range items {
			if err := tx.Delete(item.Ref); err != nil {
				return err
			}
		}

		return tx.Delete(s.listsRef(groupID).Doc(listID))
	})
}

func (s *GroupStore) ListItems(ctx context.Context, uid, groupID, listID string) ([]models.ListItem, error) {
	if !validDocumentID(groupID) {
		return nil, models.ErrInvalidGroupID
	}

	if !validDocumentID(listID) {
		return nil, models.ErrInvalidListID
	}

	var items []models.ListItem
	err := s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := s.readMemberGroup(tx, uid, groupID); err != nil {
			return err
		}

		if _, err := s.readList(tx, groupID, listID); err != nil {
			return err
		}

		snapshots, err := tx.Documents(s.itemsRef(groupID, listID).Query).GetAll()
		if err != nil {
			return err
		}

		items = make([]models.ListItem, 0, len(snapshots))
		for _, snapshot := range snapshots {
			item, err := decodeListItem(snapshot, listID)
			if err != nil {
				return err
			}

			items = append(items, item)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}

		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	return items, nil
}

func (s *GroupStore) AddListItem(ctx context.Context, uid, groupID, listID string, input models.ListItemInput) (models.ListItem, error) {
	if !validDocumentID(groupID) {
		return models.ListItem{}, models.ErrInvalidGroupID
	}

	if !validDocumentID(listID) {
		return models.ListItem{}, models.ErrInvalidListID
	}

	input, err := input.Normalize()
	if err != nil {
		return models.ListItem{}, err
	}

	ref := s.itemsRef(groupID, listID).NewDoc()
	err = s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		if _, err := s.readList(tx, groupID, listID); err != nil {
			return err
		}

		if input.AssigneeUID != "" && (!validDocumentID(input.AssigneeUID) || !slices.Contains(group.MemberUIDs, input.AssigneeUID)) {
			return models.ErrInvalidListAssignee
		}

		for _, rotationUID := range input.RotationUIDs {
			if !validDocumentID(rotationUID) || !slices.Contains(group.MemberUIDs, rotationUID) {
				return models.ErrInvalidListRotation
			}
		}

		assigneeName := ""
		if input.AssigneeUID != "" {
			names, err := s.userNames(tx, []string{input.AssigneeUID})
			if err != nil {
				return err
			}

			assigneeName = names[input.AssigneeUID]
		}

		return tx.Create(ref, newListItemData(input.Text, uid, input.AssigneeUID, assigneeName, input.RotationUIDs))
	})
	if err != nil {
		return models.ListItem{}, err
	}

	snapshot, err := ref.Get(ctx)
	if err != nil {
		return models.ListItem{}, err
	}

	return decodeListItem(snapshot, listID)
}

// SetListItemDone lets any member tick or untick an item. Ticking a rotating chore creates
// the next person's turn; unticking removes that turn again while it is still open.
func (s *GroupStore) SetListItemDone(ctx context.Context, uid, groupID, listID, itemID string, done bool) error {
	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	if !validDocumentID(listID) {
		return models.ErrInvalidListID
	}

	if !validDocumentID(itemID) {
		return models.ErrInvalidListItemID
	}

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		if _, err := s.readList(tx, groupID, listID); err != nil {
			return err
		}

		itemRef := s.itemsRef(groupID, listID).Doc(itemID)
		item, err := s.readListItem(tx, itemRef, listID)
		if err != nil {
			return err
		}

		if item.ClearedAt != nil {
			return models.ErrListItemCleared
		}

		if item.Done == done {
			return nil
		}

		if done {
			return s.completeListItem(tx, group, itemRef, item, uid)
		}

		return s.reopenListItem(tx, groupID, listID, itemRef, item)
	})
}

func (s *GroupStore) completeListItem(tx *firestore.Transaction, group models.Group, itemRef *firestore.DocumentRef, item models.ListItem, uid string) error {
	nextUID := ""
	if len(item.RotationUIDs) > 0 {
		nextUID = models.NextInRotation(item.RotationUIDs, item.AssigneeUID, group.MemberUIDs)
	}

	nameUIDs := []string{uid}
	if nextUID != "" && nextUID != uid {
		nameUIDs = append(nameUIDs, nextUID)
	}

	names, err := s.userNames(tx, nameUIDs)
	if err != nil {
		return err
	}

	followUpID := ""
	if nextUID != "" {
		followUpRef := itemRef.Parent.NewDoc()
		followUpID = followUpRef.ID
		if err := tx.Create(followUpRef, newListItemData(item.Text, item.CreatedBy, nextUID, names[nextUID], item.RotationUIDs)); err != nil {
			return err
		}
	}

	return tx.Update(itemRef, []firestore.Update{
		{Path: "done", Value: true},
		{Path: "doneBy", Value: uid},
		{Path: "doneByName", Value: names[uid]},
		{Path: "doneAt", Value: firestore.ServerTimestamp},
		{Path: "followUpID", Value: followUpID},
	})
}

func (s *GroupStore) reopenListItem(tx *firestore.Transaction, groupID, listID string, itemRef *firestore.DocumentRef, item models.ListItem) error {
	var followUpRef *firestore.DocumentRef
	if item.FollowUpID != "" {
		ref := s.itemsRef(groupID, listID).Doc(item.FollowUpID)
		followUp, err := s.readListItem(tx, ref, listID)
		if err != nil && !errors.Is(err, models.ErrListItemNotFound) {
			return err
		}

		if err == nil {
			if followUp.Done {
				return models.ErrListItemFollowUpDone
			}

			followUpRef = ref
		}
	}

	if followUpRef != nil {
		if err := tx.Delete(followUpRef); err != nil {
			return err
		}
	}

	return tx.Update(itemRef, []firestore.Update{
		{Path: "done", Value: false},
		{Path: "doneBy", Value: ""},
		{Path: "doneByName", Value: ""},
		{Path: "doneAt", Value: firestore.Delete},
		{Path: "followUpID", Value: ""},
	})
}

func (s *GroupStore) DeleteListItem(ctx context.Context, uid, groupID, listID, itemID string) error {
	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	if !validDocumentID(listID) {
		return models.ErrInvalidListID
	}

	if !validDocumentID(itemID) {
		return models.ErrInvalidListItemID
	}

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := s.readMemberGroup(tx, uid, groupID); err != nil {
			return err
		}

		if _, err := s.readList(tx, groupID, listID); err != nil {
			return err
		}

		itemRef := s.itemsRef(groupID, listID).Doc(itemID)
		item, err := s.readListItem(tx, itemRef, listID)
		if err != nil {
			return err
		}

		if item.CreatedBy != uid && item.AssigneeUID != uid {
			return models.ErrListItemForbidden
		}

		return tx.Delete(itemRef)
	})
}

// ClearDoneListItems moves ticked items into history; they are kept, not deleted.
func (s *GroupStore) ClearDoneListItems(ctx context.Context, uid, groupID, listID string) error {
	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	if !validDocumentID(listID) {
		return models.ErrInvalidListID
	}

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := s.readMemberGroup(tx, uid, groupID); err != nil {
			return err
		}

		if _, err := s.readList(tx, groupID, listID); err != nil {
			return err
		}

		snapshots, err := tx.Documents(s.itemsRef(groupID, listID).Where("done", "==", true)).GetAll()
		if err != nil {
			return err
		}

		for _, snapshot := range snapshots {
			item, err := decodeListItem(snapshot, listID)
			if err != nil {
				return err
			}

			if item.ClearedAt != nil {
				continue
			}

			if err := tx.Update(snapshot.Ref, []firestore.Update{{Path: "clearedAt", Value: firestore.ServerTimestamp}}); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *GroupStore) listsRef(groupID string) *firestore.CollectionRef {
	return s.db.Collection("groups").Doc(groupID).Collection("lists")
}

func (s *GroupStore) itemsRef(groupID, listID string) *firestore.CollectionRef {
	return s.listsRef(groupID).Doc(listID).Collection("items")
}

func (s *GroupStore) readList(tx *firestore.Transaction, groupID, listID string) (models.GroupList, error) {
	snapshot, err := tx.Get(s.listsRef(groupID).Doc(listID))
	if status.Code(err) == codes.NotFound {
		return models.GroupList{}, models.ErrListNotFound
	}

	if err != nil {
		return models.GroupList{}, err
	}

	return decodeList(snapshot, groupID)
}

func (s *GroupStore) readListItem(tx *firestore.Transaction, ref *firestore.DocumentRef, listID string) (models.ListItem, error) {
	snapshot, err := tx.Get(ref)
	if status.Code(err) == codes.NotFound {
		return models.ListItem{}, models.ErrListItemNotFound
	}

	if err != nil {
		return models.ListItem{}, err
	}

	return decodeListItem(snapshot, listID)
}

func decodeList(snapshot *firestore.DocumentSnapshot, groupID string) (models.GroupList, error) {
	var list models.GroupList
	if err := snapshot.DataTo(&list); err != nil {
		return models.GroupList{}, err
	}

	list.ID = snapshot.Ref.ID
	list.GroupID = groupID
	return list, nil
}

func decodeListItem(snapshot *firestore.DocumentSnapshot, listID string) (models.ListItem, error) {
	var item models.ListItem
	if err := snapshot.DataTo(&item); err != nil {
		return models.ListItem{}, err
	}

	item.ID = snapshot.Ref.ID
	item.ListID = listID
	if item.RotationUIDs == nil {
		item.RotationUIDs = []string{}
	}

	return item, nil
}

func newListItemData(text, createdBy, assigneeUID, assigneeName string, rotationUIDs []string) map[string]any {
	return map[string]any{
		"text":         text,
		"createdBy":    createdBy,
		"createdAt":    firestore.ServerTimestamp,
		"assigneeUID":  assigneeUID,
		"assigneeName": assigneeName,
		"rotationUIDs": rotationUIDs,
		"done":         false,
		"doneBy":       "",
		"doneByName":   "",
		"followUpID":   "",
	}
}
