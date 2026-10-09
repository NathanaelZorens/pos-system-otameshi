package discount

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Querier is satisfied by *sql.DB and *sql.Tx.
type Querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Applied is the price snapshot for one order line.
type Applied struct {
	ListUnitPriceCents int
	UnitPriceCents     int
	RuleID             *string
	Label              *string
}

type ruleRow struct {
	ID              string
	Name            string
	TargetType      string
	TargetID        string
	DiscountType    string
	Amount          int
	IsFeaturedPrice bool
	StartTime       sql.NullString
	EndTime         sql.NullString
	Weekdays        sql.NullString
	StartsOn        sql.NullString
	EndsOn          sql.NullString
}

// ResolveUnitPrice picks at most one active rule.
// Default: largest yen savings among item + category.
// If any matching item rule has is_featured_price, pick among those (cheapest featured wins).
func ResolveUnitPrice(q Querier, menuItemID string, listPriceCents int, categoryID *string, now time.Time) (Applied, error) {
	out := Applied{ListUnitPriceCents: listPriceCents, UnitPriceCents: listPriceCents}

	candidates, err := loadMatchingRules(q, menuItemID, categoryID, now, "")
	if err != nil {
		return out, err
	}

	best, unit := pickBest(candidates, listPriceCents)
	if best == nil {
		return out, nil
	}
	out.UnitPriceCents = unit
	id := best.ID
	label := formatBadgeLabel(best, unit)
	out.RuleID = &id
	out.Label = &label
	return out, nil
}

// FeaturedConflict is an admin-time warning when featured would beat a better deal.
type FeaturedConflict struct {
	WorseDeal        bool   `json:"worse_deal"`
	ThisSavingCents  int    `json:"this_saving_cents"`
	BetterRuleName   string `json:"better_rule_name,omitempty"`
	BetterSavingCents int   `json:"better_saving_cents,omitempty"`
	ListPriceCents   int    `json:"list_price_cents"`
	Message          string `json:"message,omitempty"`
}

// CheckFeaturedConflict compares a draft item rule's savings to other rules
// currently matching that item (excludeRuleID for edits). Only meaningful for item targets.
func CheckFeaturedConflict(
	q Querier,
	menuItemID string,
	listPriceCents int,
	categoryID *string,
	discountType string,
	amount int,
	excludeRuleID string,
	now time.Time,
) (FeaturedConflict, error) {
	out := FeaturedConflict{ListPriceCents: listPriceCents}
	unit, ok := applyAmount(listPriceCents, discountType, amount)
	if !ok {
		return out, nil
	}
	thisSaving := listPriceCents - unit
	out.ThisSavingCents = thisSaving
	if thisSaving <= 0 {
		return out, nil
	}

	others, err := loadMatchingRules(q, menuItemID, categoryID, now, excludeRuleID)
	if err != nil {
		return out, err
	}
	var bestName string
	bestSaving := -1
	for _, r := range others {
		u, ok := applyAmount(listPriceCents, r.DiscountType, r.Amount)
		if !ok {
			continue
		}
		s := listPriceCents - u
		if s <= 0 {
			continue
		}
		if s > bestSaving {
			bestSaving = s
			bestName = r.Name
		}
	}
	if bestSaving > thisSaving {
		out.WorseDeal = true
		out.BetterRuleName = bestName
		out.BetterSavingCents = bestSaving
		out.Message = fmt.Sprintf(
			"This deal saves ¥%d, but “%s” would save ¥%d right now. Prioritize as Featured price will charge the guest more.",
			thisSaving, bestName, bestSaving,
		)
	}
	return out, nil
}

func loadMatchingRules(q Querier, menuItemID string, categoryID *string, now time.Time, excludeID string) ([]ruleRow, error) {
	rows, err := q.Query(`
		SELECT id, name, target_type, target_id, discount_type, amount,
		       COALESCE(is_featured_price, 0),
		       start_time, end_time, weekdays, starts_on, ends_on
		FROM discount_rules
		WHERE is_active = 1
		  AND (
		    (target_type = 'item' AND target_id = ?)
		    OR (target_type = 'category' AND ? IS NOT NULL AND target_id = ?)
		  )
	`, menuItemID, categoryID, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ruleRow
	for rows.Next() {
		var r ruleRow
		var featured int
		if err := rows.Scan(
			&r.ID, &r.Name, &r.TargetType, &r.TargetID, &r.DiscountType, &r.Amount,
			&featured,
			&r.StartTime, &r.EndTime, &r.Weekdays, &r.StartsOn, &r.EndsOn,
		); err != nil {
			return nil, err
		}
		if excludeID != "" && r.ID == excludeID {
			continue
		}
		r.IsFeaturedPrice = featured == 1 && r.TargetType == "item"
		if !ruleMatchesNow(r, now) {
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func pickBest(candidates []ruleRow, listPriceCents int) (*ruleRow, int) {
	var featured []ruleRow
	for _, r := range candidates {
		if r.IsFeaturedPrice {
			featured = append(featured, r)
		}
	}
	pool := candidates
	if len(featured) > 0 {
		pool = featured
	}

	var best *ruleRow
	bestSaving := -1
	bestUnit := listPriceCents
	for i := range pool {
		r := &pool[i]
		unit, ok := applyAmount(listPriceCents, r.DiscountType, r.Amount)
		if !ok {
			continue
		}
		saving := listPriceCents - unit
		if saving <= 0 {
			continue
		}
		if best == nil || saving > bestSaving {
			best = r
			bestSaving = saving
			bestUnit = unit
		}
	}
	return best, bestUnit
}

func ruleMatchesNow(r ruleRow, now time.Time) bool {
	date := now.Format("2006-01-02")
	if r.StartsOn.Valid && r.StartsOn.String != "" && date < r.StartsOn.String {
		return false
	}
	if r.EndsOn.Valid && r.EndsOn.String != "" && date > r.EndsOn.String {
		return false
	}
	if r.Weekdays.Valid && strings.TrimSpace(r.Weekdays.String) != "" {
		want := isoWeekday(now)
		ok := false
		for _, part := range strings.Split(r.Weekdays.String, ",") {
			part = strings.TrimSpace(part)
			if part == fmt.Sprintf("%d", want) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	hasStart := r.StartTime.Valid && strings.TrimSpace(r.StartTime.String) != ""
	hasEnd := r.EndTime.Valid && strings.TrimSpace(r.EndTime.String) != ""
	if !hasStart && !hasEnd {
		return true
	}
	clock := now.Format("15:04")
	start := "00:00"
	end := "23:59"
	if hasStart {
		start = normalizeHHMM(r.StartTime.String)
	}
	if hasEnd {
		end = normalizeHHMM(r.EndTime.String)
	}
	if start <= end {
		return clock >= start && clock <= end
	}
	return clock >= start || clock <= end
}

func isoWeekday(t time.Time) int {
	w := int(t.Weekday())
	if w == 0 {
		return 7
	}
	return w
}

func normalizeHHMM(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 5 {
		return s[:5]
	}
	return s
}

func applyAmount(listCents int, discountType string, amount int) (int, bool) {
	if amount <= 0 {
		return listCents, false
	}
	switch discountType {
	case "fixed":
		u := listCents - amount
		if u < 0 {
			u = 0
		}
		return u, true
	case "percent":
		if amount > 100 {
			amount = 100
		}
		u := listCents - (listCents * amount / 100)
		if u < 0 {
			u = 0
		}
		return u, true
	default:
		return listCents, false
	}
}

func formatBadgeLabel(r *ruleRow, unitCents int) string {
	yen := formatYen(unitCents)
	switch r.DiscountType {
	case "fixed":
		return fmt.Sprintf("−¥%d=%s", r.Amount, yen)
	case "percent":
		return fmt.Sprintf("−%d%%=%s", r.Amount, yen)
	default:
		if strings.TrimSpace(r.Name) != "" {
			return r.Name
		}
		return "Discount"
	}
}

func formatYen(cents int) string {
	n := cents
	if n < 0 {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	if s != "" {
		parts = append([]string{s}, parts...)
	}
	return "¥" + strings.Join(parts, ",")
}
