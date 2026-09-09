package tmp

import "testing"

func TestNormalizeCountryToAlpha2(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Already alpha-2 → pass-through (uppercased).
		{"US", "US"},
		{"us", "US"},
		{" gb ", "GB"},

		// Common alpha-3 → alpha-2 lookups.
		{"USA", "US"},
		{"GBR", "GB"},
		{"CAN", "CA"},
		{"DEU", "DE"},
		{"FRA", "FR"},
		{"JPN", "JP"},
		{"BRA", "BR"},
		{"IND", "IN"},
		{"ARE", "AE"},
		{"CHE", "CH"},

		// The traps the first-two-characters heuristic would silently
		// misclassify. These MUST round-trip through the lookup table.
		{"AUT", "AT"}, // Austria — first-two would give AU (Australia)
		{"IRL", "IE"}, // Ireland — first-two would give IR (Iran)
		{"PRT", "PT"}, // Portugal — first-two would give PR (Puerto Rico)
		{"KOR", "KR"}, // Korea (South) — first-two would give KO (unassigned)
		{"CHN", "CN"}, // China — first-two would give CH (Switzerland)
		{"SVK", "SK"}, // Slovakia — first-two would give SV (unassigned)
		{"SWZ", "SZ"}, // Eswatini — first-two would give SW (unassigned)

		// Rejected shapes.
		{"", ""},
		{"U", ""},        // 1 char
		{"US1", ""},      // 3 chars but not letters
		{"USSR", ""},     // historical, no ISO code
		{"UNITED", ""},   // 6 chars
		{"12", ""},       // 2 chars but not letters
		{"unknown3", ""}, // 8 chars
	}
	for _, tc := range cases {
		got := normalizeCountryToAlpha2(tc.in)
		if got != tc.want {
			t.Errorf("normalizeCountryToAlpha2(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
