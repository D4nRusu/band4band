package database_test

import (
	"errors"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestListItemsLifecycle(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)

	list, err := groups.CreateList(ctx, "alice", group.ID, models.ListInput{Name: " Shopping "})
	if err != nil || list.Name != "Shopping" || list.CreatedAt.IsZero() {
		t.Fatalf("list=%+v err=%v", list, err)
	}

	milk, err := groups.AddListItem(ctx, "alice", group.ID, list.ID, models.ListItemInput{Text: "Milk"})
	if err != nil {
		t.Fatal(err)
	}

	if err := groups.DeleteListItem(ctx, "bob", group.ID, list.ID, milk.ID); !errors.Is(err, models.ErrListItemForbidden) {
		t.Fatalf("err=%v", err)
	}

	if err := groups.SetListItemDone(ctx, "bob", group.ID, list.ID, milk.ID, true); err != nil {
		t.Fatal(err)
	}

	if err := groups.ClearDoneListItems(ctx, "charlie", group.ID, list.ID); err != nil {
		t.Fatal(err)
	}

	items, err := groups.ListItems(ctx, "charlie", group.ID, list.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 || !items[0].Done || items[0].DoneBy != "bob" || items[0].ClearedAt == nil {
		t.Fatalf("items=%+v", items)
	}

	if err := groups.SetListItemDone(ctx, "alice", group.ID, list.ID, milk.ID, false); !errors.Is(err, models.ErrListItemCleared) {
		t.Fatalf("err=%v", err)
	}

	if err := groups.DeleteList(ctx, "bob", group.ID, list.ID); !errors.Is(err, models.ErrListForbidden) {
		t.Fatalf("err=%v", err)
	}

	if err := groups.DeleteList(ctx, "alice", group.ID, list.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := groups.ListItems(ctx, "alice", group.ID, list.ID); !errors.Is(err, models.ErrListNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestRotatingChoreCreatesNextTurn(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)

	list, err := groups.CreateList(ctx, "alice", group.ID, models.ListInput{Name: "Chores"})
	if err != nil {
		t.Fatal(err)
	}

	dishes, err := groups.AddListItem(ctx, "alice", group.ID, list.ID, models.ListItemInput{
		Text: "Dishes", AssigneeUID: "alice", RotationUIDs: []string{"alice", "bob", "charlie"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := groups.AddListItem(ctx, "alice", group.ID, list.ID, models.ListItemInput{
		Text: "Bins", AssigneeUID: "alice", RotationUIDs: []string{"alice", "outsider"},
	}); !errors.Is(err, models.ErrInvalidListRotation) {
		t.Fatalf("err=%v", err)
	}

	if err := groups.SetListItemDone(ctx, "alice", group.ID, list.ID, dishes.ID, true); err != nil {
		t.Fatal(err)
	}

	items, err := groups.ListItems(ctx, "alice", group.ID, list.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 || items[0].FollowUpID != items[1].ID || items[1].AssigneeUID != "bob" || items[1].Done {
		t.Fatalf("items=%+v", items)
	}

	if err := groups.SetListItemDone(ctx, "alice", group.ID, list.ID, dishes.ID, false); err != nil {
		t.Fatal(err)
	}

	items, err = groups.ListItems(ctx, "alice", group.ID, list.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 || items[0].Done || items[0].FollowUpID != "" {
		t.Fatalf("items after reopening=%+v", items)
	}

	if err := groups.SetListItemDone(ctx, "alice", group.ID, list.ID, dishes.ID, true); err != nil {
		t.Fatal(err)
	}

	items, err = groups.ListItems(ctx, "alice", group.ID, list.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := groups.SetListItemDone(ctx, "bob", group.ID, list.ID, items[1].ID, true); err != nil {
		t.Fatal(err)
	}

	if err := groups.SetListItemDone(ctx, "alice", group.ID, list.ID, dishes.ID, false); !errors.Is(err, models.ErrListItemFollowUpDone) {
		t.Fatalf("err=%v", err)
	}
}
