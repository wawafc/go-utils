package amountfast_test

import (
	"math/rand"
	"testing"

	"github.com/wawafc/go-utils/amount"
	"github.com/wawafc/go-utils/amountfast"
)

func TestFormatString_Table(t *testing.T) {
	// FormatNumberWithPrecision and FormatNumberWithoutDecimal use
	// useComma=false — they never emit thousands separators.
	cases := []struct {
		in        string
		formatStr string // FormatString: precision 2 + commas
		formatNum string // FormatNumber: precision 2, no commas
		formatP0  string // FormatNumberWithPrecision(0)   — no commas
		formatP6  string // FormatNumberWithPrecision(6)
	}{
		{"0", "0.00", "0.00", "0", "0.000000"},
		{"1", "1.00", "1.00", "1", "1.000000"},
		{"-1", "-1.00", "-1.00", "-1", "-1.000000"},
		{"1.25", "1.25", "1.25", "1", "1.250000"},
		{"-1.25", "-1.25", "-1.25", "-1", "-1.250000"},
		{"1.5", "1.50", "1.50", "2", "1.500000"},
		{"-1.5", "-1.50", "-1.50", "-2", "-1.500000"},
		{"1234.56", "1,234.56", "1234.56", "1235", "1234.560000"},
		{"1234567.89", "1,234,567.89", "1234567.89", "1234568", "1234567.890000"},
		{"-1234567.89", "-1,234,567.89", "-1234567.89", "-1234568", "-1234567.890000"},
		{"999.999", "1,000.00", "1000.00", "1000", "999.999000"},
		{"1000", "1,000.00", "1000.00", "1000", "1000.000000"},
		{"999", "999.00", "999.00", "999", "999.000000"},
		{"-999", "-999.00", "-999.00", "-999", "-999.000000"},
		{"1000000", "1,000,000.00", "1000000.00", "1000000", "1000000.000000"},
		{"0.01", "0.01", "0.01", "0", "0.010000"},
		{"0.005", "0.01", "0.01", "0", "0.005000"},
		{"-0.005", "-0.01", "-0.01", "0", "-0.005000"},
		{"999999999.999999", "1,000,000,000.00", "1000000000.00", "1000000000", "999999999.999999"},
	}
	for _, c := range cases {
		v, err := amountfast.FromString(c.in)
		if err != nil {
			t.Fatalf("FromString(%q): %v", c.in, err)
		}
		if got := v.FormatString(); got != c.formatStr {
			t.Errorf("FormatString(%s): got %q, want %q", c.in, got, c.formatStr)
		}
		if got := v.FormatNumber(); got != c.formatNum {
			t.Errorf("FormatNumber(%s): got %q, want %q", c.in, got, c.formatNum)
		}
		if got := v.FormatNumberWithPrecision(0); got != c.formatP0 {
			t.Errorf("FormatNumberWithPrecision(%s, 0): got %q, want %q", c.in, got, c.formatP0)
		}
		if got := v.FormatNumberWithoutDecimal(); got != c.formatP0 {
			t.Errorf("FormatNumberWithoutDecimal(%s): got %q, want %q", c.in, got, c.formatP0)
		}
		if got := v.FormatNumberWithPrecision(6); got != c.formatP6 {
			t.Errorf("FormatNumberWithPrecision(%s, 6): got %q, want %q", c.in, got, c.formatP6)
		}
	}
}

// Randomised parity vs amount for non-negative values small enough to survive
// float64 round-trip. We intentionally skip negative large values because the
// legacy amount.FormatString has a known comma-placement bug (produces "-,999"
// for -999) which amountfast already corrects.
func TestFormatString_RandomParityPositive(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	const maxScaled = int64(1) << 52 // below float64 mantissa precision
	for i := 0; i < 20000; i++ {
		s := r.Int63n(maxScaled)
		fa := amountfast.FromScaled(s)
		a, err := amount.FromString(fa.String())
		if err != nil {
			t.Fatalf("amount.FromString(%q): %v", fa.String(), err)
		}
		if got, want := fa.FormatString(), a.FormatString(); got != want {
			t.Fatalf("FormatString(scaled=%d): got %q, want %q", s, got, want)
		}
		for p := 0; p <= 6; p++ {
			if got, want := fa.FormatNumberWithPrecision(p), a.FormatNumberWithPrecision(p); got != want {
				t.Fatalf("FormatNumberWithPrecision(scaled=%d, p=%d): got %q, want %q", s, p, got, want)
			}
		}
	}
}

func BenchmarkFast_FormatString(b *testing.B) {
	x := amountfast.FromFloat(1234567.89)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.FormatString()
	}
}

func BenchmarkAmount_FormatString(b *testing.B) {
	x := amount.FromFloat(1234567.89)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.FormatString()
	}
}

func BenchmarkFast_FormatNumberP6(b *testing.B) {
	x := amountfast.FromFloat(1234567.891234)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.FormatNumberWithPrecision(6)
	}
}

func BenchmarkAmount_FormatNumberP6(b *testing.B) {
	x := amount.FromFloat(1234567.891234)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.FormatNumberWithPrecision(6)
	}
}
