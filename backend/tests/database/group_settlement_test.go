package database_test

import (
	"errors"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestSettleUpUpdatesDebtGraphs(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "bob", 1000, "alice")
	graphExpense(t, groups, ctx, group.ID, "charlie", 700, "bob")

	if err := groups.SettleUp(ctx, "alice", group.ID, "alice", "bob", 300); err != nil {
		t.Fatal(err)
	}

	raw := readStoredGraph(t, db, ctx, group.ID)
	checkDebt(t, raw, "alice", "bob", 1000)
	checkDebt(t, raw, "bob", "alice", 300)
	checkDebt(t, raw, "bob", "charlie", 700)

	simplified := readSimplifiedGraph(t, db, ctx, group.ID)
	checkDebt(t, simplified, "alice", "charlie", 700)
	checkDebt(t, simplified, "alice", "bob", 0)
	checkDebt(t, simplified, "bob", "charlie", 0)
}

func TestSettlementsListsRecordedPayments(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "bob", 1000, "alice")

	if err := groups.SettleUp(ctx, "alice", group.ID, "alice", "bob", 300); err != nil {
		t.Fatal(err)
	}

	if err := groups.SettleUp(ctx, "bob", group.ID, "charlie", "bob", 200); err == nil {
		t.Fatal("recipient recorded a payment from a third member")
	}

	if err := groups.SettleUp(ctx, "bob", group.ID, "alice", "bob", 100); err != nil {
		t.Fatal(err)
	}

	settlements, err := groups.Settlements(ctx, "charlie", group.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(settlements) != 2 || settlements[0].AmountMinor != 100 || settlements[1].AmountMinor != 300 {
		t.Fatalf("settlements=%+v", settlements)
	}

	first := settlements[1]
	if first.FromUID != "alice" || first.ToUID != "bob" || first.CreatedBy != "alice" || first.Currency != "EUR" || first.CreatedAt.IsZero() {
		t.Fatalf("settlement=%+v", first)
	}

	if _, err := groups.Settlements(ctx, "outsider", group.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("err=%v", err)
	}
}

func TestSettleUpRejectsInvalidPayments(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "bob", 1000, "alice")

	if err := groups.SettleUp(ctx, "alice", group.ID, "alice", "outsider", 10); !errors.Is(err, models.ErrInvalidDebtUser) {
		t.Fatalf("err=%v", err)
	}

	if err := groups.SettleUp(ctx, "charlie", group.ID, "alice", "bob", 10); !errors.Is(err, models.ErrSettlementNotParty) {
		t.Fatalf("err=%v", err)
	}
}

func TestSettleUpRecordedByRecipient(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "bob", 1000, "alice")

	if err := groups.SettleUp(ctx, "bob", group.ID, "alice", "bob", 400); err != nil {
		t.Fatal(err)
	}

	simplified := readSimplifiedGraph(t, db, ctx, group.ID)
	checkDebt(t, simplified, "alice", "bob", 600)
}

func TestSettleUpAllowsPaymentsBeyondDebt(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "bob", 1000, "alice")

	if err := groups.SettleUp(ctx, "alice", group.ID, "alice", "bob", 1500); err != nil {
		t.Fatal(err)
	}

	if err := groups.SettleUp(ctx, "alice", group.ID, "alice", "charlie", 200); err != nil {
		t.Fatal(err)
	}

	simplified := readSimplifiedGraph(t, db, ctx, group.ID)
	checkDebt(t, simplified, "alice", "bob", 0)
	checkDebt(t, simplified, "bob", "alice", 500)
	checkDebt(t, simplified, "charlie", "alice", 200)
}

func TestUserOwedSummaryUsesSimplifiedGraph(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "bob", 1000, "alice")
	graphExpense(t, groups, ctx, group.ID, "charlie", 700, "bob")

	summary, err := groups.UserOwedSummary(ctx, "alice", group.ID, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if summary.UserUID != "alice" || summary.TotalAmountMinor != 700 || summary.Currency != "EUR" {
		t.Fatalf("summary=%+v", summary)
	}

	if len(summary.Debts) != 1 || summary.Debts[0].ToUID != "charlie" || summary.Debts[0].AmountMinor != 700 {
		t.Fatalf("summary=%+v", summary)
	}
}

func TestUserReceivableSummaryUsesSimplifiedGraph(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "bob", 1000, "alice")
	graphExpense(t, groups, ctx, group.ID, "charlie", 700, "bob")

	summary, err := groups.UserReceivableSummary(ctx, "alice", group.ID, "charlie")
	if err != nil {
		t.Fatal(err)
	}

	if summary.UserUID != "charlie" || summary.TotalAmountMinor != 700 || summary.Currency != "EUR" {
		t.Fatalf("summary=%+v", summary)
	}

	if len(summary.Debts) != 1 || summary.Debts[0].FromUID != "alice" || summary.Debts[0].AmountMinor != 700 {
		t.Fatalf("summary=%+v", summary)
	}
}
