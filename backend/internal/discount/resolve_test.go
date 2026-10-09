package discount

import (
	"testing"
	"time"
)

func TestApplyAmount(t *testing.T) {
	u, ok := applyAmount(1000, "fixed", 200)
	if !ok || u != 800 {
		t.Fatalf("fixed: got %d ok=%v", u, ok)
	}
	u, ok = applyAmount(1000, "percent", 10)
	if !ok || u != 900 {
		t.Fatalf("percent: got %d ok=%v", u, ok)
	}
	u, ok = applyAmount(100, "fixed", 200)
	if !ok || u != 0 {
		t.Fatalf("floor: got %d ok=%v", u, ok)
	}
}

func TestRuleMatchesEmptyMeansAlways(t *testing.T) {
	now := time.Date(2026, 10, 8, 15, 30, 0, 0, time.Local)
	r := ruleRow{}
	if !ruleMatchesNow(r, now) {
		t.Fatal("empty rule should match any time")
	}
}

func TestIsoWeekday(t *testing.T) {
	// 2026-10-08 is Thursday
	thu := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	if isoWeekday(thu) != 4 {
		t.Fatalf("want 4 got %d", isoWeekday(thu))
	}
}

func TestPickBestFeaturedBeatsCheaper(t *testing.T) {
	list := 1000
	cheap := ruleRow{ID: "c", Name: "Category −¥200", DiscountType: "fixed", Amount: 200}
	featured := ruleRow{
		ID: "f", Name: "Featured −¥50", DiscountType: "fixed", Amount: 50,
		IsFeaturedPrice: true, TargetType: "item",
	}
	best, unit := pickBest([]ruleRow{cheap, featured}, list)
	if best == nil || best.ID != "f" || unit != 950 {
		t.Fatalf("want featured 950, got %#v unit=%d", best, unit)
	}
}

func TestPickBestCheapestWithoutFeatured(t *testing.T) {
	list := 1000
	a := ruleRow{ID: "a", DiscountType: "fixed", Amount: 50}
	b := ruleRow{ID: "b", DiscountType: "fixed", Amount: 200}
	best, unit := pickBest([]ruleRow{a, b}, list)
	if best == nil || best.ID != "b" || unit != 800 {
		t.Fatalf("want cheapest b/800, got %#v unit=%d", best, unit)
	}
}
