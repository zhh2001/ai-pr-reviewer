package config

import "testing"

func TestRiskConfidenceThreshold(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want float64
	}{
		{"unset falls back to default", "", DefaultRiskConfidenceThreshold},
		{"legal mid value", "0.7", 0.7},
		{"legal zero", "0", 0},
		{"legal one", "1", 1},
		{"above range clamps to 1", "1.5", 1.0},
		{"below range clamps to 0", "-0.3", 0.0},
		{"invalid string falls back", "abc", DefaultRiskConfidenceThreshold},
		{"empty after spaces falls back", " ", DefaultRiskConfidenceThreshold},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("RISK_CONFIDENCE_THRESHOLD", tc.env)
			got := Load().RiskConfidenceThreshold
			if got != tc.want {
				t.Errorf("RiskConfidenceThreshold(env=%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}
