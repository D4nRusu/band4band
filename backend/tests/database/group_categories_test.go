package database_test

import (
	"errors"
	"sync"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestGroupCategoriesMembershipAndUniqueness(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	categories, err := groups.Categories(ctx, "owner", group.ID)
	if err != nil || len(categories) != 10 {
		t.Fatalf("categories=%+v err=%v", categories, err)
	}

	if _, err := groups.Categories(ctx, "outsider", group.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider list: %v", err)
	}

	if _, err := groups.AddCategory(ctx, "outsider", group.ID, models.CategoryInput{Name: "Pets", Icon: "other"}); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider create: %v", err)
	}

	if _, err := groups.AddCategory(ctx, "owner", group.ID, models.CategoryInput{Name: " FOOD  & drinks ", Icon: "food"}); !errors.Is(err, models.ErrCategoryExists) {
		t.Fatalf("default duplicate: %v", err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Update(ctx, []firestore.Update{{Path: "memberUIDs", Value: firestore.ArrayUnion("member")}}); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, name := range []string{"Pet Care", " pet   care "} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			_, err := groups.AddCategory(ctx, "member", group.ID, models.CategoryInput{Name: name, Icon: "other"})
			results <- err
		}(name)
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, models.ErrCategoryExists) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}

	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}

	categories, err = groups.Categories(ctx, "owner", group.ID)
	if err != nil || len(categories) != 11 {
		t.Fatalf("categories=%+v err=%v", categories, err)
	}

	other, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Other group"})
	if err != nil {
		t.Fatal(err)
	}

	categories, err = groups.Categories(ctx, "owner", other.ID)
	if err != nil || len(categories) != 10 {
		t.Fatalf("categories leaked across groups: %+v %v", categories, err)
	}

	if err := groups.Leave(ctx, "member", group.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := groups.AddCategory(ctx, "member", group.ID, models.CategoryInput{Name: "Gifts", Icon: "other"}); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("former member create: %v", err)
	}
}

func TestDeleteCustomCategory(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	category, err := groups.AddCategory(ctx, "owner", group.ID, models.CategoryInput{Name: "Pets", Icon: "other"})
	if err != nil {
		t.Fatal(err)
	}

	if err := groups.DeleteCategory(ctx, "outsider", group.ID, category.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider deletion: %v", err)
	}

	if err := groups.DeleteCategory(ctx, "owner", group.ID, "default_food"); !errors.Is(err, models.ErrDefaultCategory) {
		t.Fatalf("default deletion: %v", err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Update(ctx, []firestore.Update{{Path: "memberUIDs", Value: firestore.ArrayUnion("member")}}); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := groups.DeleteCategory(ctx, "member", group.ID, category.ID); err != nil {
			t.Fatal(err)
		}
	}

	categories, err := groups.Categories(ctx, "owner", group.ID)
	if err != nil || len(categories) != 10 {
		t.Fatalf("categories=%+v err=%v", categories, err)
	}

	if _, err := groups.AddCategory(ctx, "member", group.ID, models.CategoryInput{Name: "Pets", Icon: "other"}); err != nil {
		t.Fatalf("name not reusable after deletion: %v", err)
	}

	if err := groups.Leave(ctx, "member", group.ID); err != nil {
		t.Fatal(err)
	}

	if err := groups.DeleteCategory(ctx, "member", group.ID, category.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("former member deletion: %v", err)
	}
}
