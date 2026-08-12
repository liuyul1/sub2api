package service

import (
	"strings"
	"testing"
)

func TestParseRechargeBonusTiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{"empty string is valid (no tiers)", "", 0, false},
		{"valid single tier", `[{"min":100,"bonus":10}]`, 1, false},
		{"valid tiers with multiplier", `[{"min":100,"bonus":10},{"min":200,"bonus":30,"multiplier":1.15}]`, 2, false},
		{"invalid json", `[{min:100}]`, 0, true},
		{"empty array", `[]`, 0, true},
		{"zero min", `[{"min":0,"bonus":10}]`, 0, true},
		{"negative min", `[{"min":-5,"bonus":10}]`, 0, true},
		{"negative bonus", `[{"min":100,"bonus":-1}]`, 0, true},
		{"negative multiplier", `[{"min":100,"bonus":10,"multiplier":-1}]`, 0, true},
		{"duplicate min", `[{"min":100,"bonus":1},{"min":100,"bonus":2}]`, 0, true},
		{"unsorted min", `[{"min":200,"bonus":2},{"min":100,"bonus":1}]`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tiers, err := ParseRechargeBonusTiers(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got tiers=%v", tiers)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tiers) != tt.wantLen {
				t.Fatalf("expected %d tiers, got %d", tt.wantLen, len(tiers))
			}
		})
	}
}

func TestNormalizeRechargeBonusTiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantFirst float64
		wantCount int
	}{
		{"empty falls back to defaults", "", 100, len(DefaultRechargeBonusTiers)},
		{"invalid json falls back to defaults", `not-json`, 100, len(DefaultRechargeBonusTiers)},
		{"valid config is kept", `[{"min":50,"bonus":5}]`, 50, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tiers := NormalizeRechargeBonusTiers(tt.input)
			if len(tiers) != tt.wantCount {
				t.Fatalf("expected %d tiers, got %d", tt.wantCount, len(tiers))
			}
			if len(tiers) > 0 && tiers[0].Min != tt.wantFirst {
				t.Fatalf("expected first min=%v, got %v", tt.wantFirst, tiers[0].Min)
			}
		})
	}
}

func TestMatchRechargeBonusTier(t *testing.T) {
	t.Parallel()

	tiers := []RechargeBonusTier{
		{Min: 100, Bonus: 10},
		{Min: 200, Bonus: 30, Multiplier: 1.15},
		{Min: 500, Bonus: 90, Multiplier: 1.25},
	}

	tests := []struct {
		name      string
		amount    float64
		wantBonus float64
		wantNil   bool
	}{
		{"below lowest tier", 50, 0, true},
		{"exactly lowest tier", 100, 10, false},
		{"between tiers", 150, 10, false},
		{"exactly middle tier", 200, 30, false},
		{"above all tiers", 1000, 90, false},
		{"zero amount", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MatchRechargeBonusTier(tiers, tt.amount)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected a tier for amount %v", tt.amount)
			}
			if got.Bonus != tt.wantBonus {
				t.Fatalf("expected bonus %v, got %v", tt.wantBonus, got.Bonus)
			}
		})
	}
}

func TestFormatAndSortRechargeBonusTiers(t *testing.T) {
	t.Parallel()

	tiers := []RechargeBonusTier{
		{Min: 500, Bonus: 90, Multiplier: 1.25},
		{Min: 100, Bonus: 10},
	}
	sortRechargeBonusTiers(tiers)
	if tiers[0].Min != 100 || tiers[1].Min != 500 {
		t.Fatalf("expected sorted order, got %+v", tiers)
	}

	jsonStr, err := formatRechargeBonusTiers(tiers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(jsonStr, `[{"min":100,"bonus":10`) {
		t.Fatalf("unexpected json: %s", jsonStr)
	}

	roundTripped, err := ParseRechargeBonusTiers(jsonStr)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v", err)
	}
	if len(roundTripped) != 2 {
		t.Fatalf("expected 2 tiers after round-trip, got %d", len(roundTripped))
	}
}
