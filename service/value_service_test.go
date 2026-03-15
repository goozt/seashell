package service_test

import (
	"testing"
)

func TestValueCalculation(t *testing.T) {
	// Pure velocity-of-money formula: price = base * (1 + k * velocity)
	tests := []struct {
		name      string
		basePrice float64
		k         float64
		volume    int
		supply    int
		wantPrice float64
	}{
		{
			name:      "zero_velocity_returns_base",
			basePrice: 100,
			k:         0.1,
			volume:    0,
			supply:    1000,
			wantPrice: 100.0,
		},
		{
			name:      "full_velocity_increases_price",
			basePrice: 100,
			k:         0.1,
			volume:    1000,
			supply:    1000,
			wantPrice: 110.0, // 100 * (1 + 0.1 * 1.0)
		},
		{
			name:      "low_velocity",
			basePrice: 50,
			k:         0.2,
			volume:    100,
			supply:    1000,
			wantPrice: 51.0, // 50 * (1 + 0.2 * 0.1)
		},
		{
			name:      "zero_supply_returns_base",
			basePrice: 75,
			k:         0.5,
			volume:    500,
			supply:    0,
			wantPrice: 75.0, // velocity = 0 when supply = 0
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var velocity float64
			if tc.supply > 0 {
				velocity = float64(tc.volume) / float64(tc.supply)
			}
			price := tc.basePrice * (1 + tc.k*velocity)
			const eps = 0.0001
			if diff := price - tc.wantPrice; diff > eps || diff < -eps {
				t.Errorf("price = %.4f, want %.4f (velocity=%.4f)", price, tc.wantPrice, velocity)
			}
		})
	}
}
