package amountfast_test

import (
	"testing"

	"github.com/wawafc/go-utils/amount"
	"github.com/wawafc/go-utils/amountfast"
)

func BenchmarkAmount_Add(b *testing.B) {
	x := amount.FromFloat(1.25)
	y := amount.FromFloat(0.75)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = x.Add(y)
	}
}

func BenchmarkFast_Add(b *testing.B) {
	x := amountfast.FromFloat(1.25)
	y := amountfast.FromFloat(0.75)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = x.Add(y)
	}
}

func BenchmarkAmount_Mul(b *testing.B) {
	x := amount.FromFloat(1.25)
	y := amount.FromFloat(0.75)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = x.Mul(y)
		if x.IsZero() {
			x = amount.FromFloat(1.25)
		}
	}
}

func BenchmarkFast_Mul(b *testing.B) {
	x := amountfast.FromFloat(1.25)
	y := amountfast.FromFloat(0.75)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x = x.Mul(y)
		if x.IsZero() {
			x = amountfast.FromFloat(1.25)
		}
	}
}

func BenchmarkAmount_SpinLoop(b *testing.B) {
	bet := amount.FromFloat(1.0)
	mult := amount.FromFloat(1.25)
	lines := amount.FromInt(20)
	total := amount.FromInt(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payout := bet.Mul(lines).Mul(mult)
		total = total.Add(payout)
	}
}

func BenchmarkFast_SpinLoop(b *testing.B) {
	bet := amountfast.FromFloat(1.0)
	mult := amountfast.FromFloat(1.25)
	total := amountfast.Zero
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payout := bet.MulInt(20).Mul(mult)
		total = total.Add(payout)
	}
}

func BenchmarkAmount_FromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = amount.FromString("1.25")
	}
}

func BenchmarkFast_FromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = amountfast.FromString("1.25")
	}
}

func BenchmarkAmount_String(b *testing.B) {
	x := amount.FromFloat(1.25)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.String()
	}
}

func BenchmarkFast_String(b *testing.B) {
	x := amountfast.FromFloat(1.25)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.String()
	}
}

func BenchmarkAmount_Div(b *testing.B) {
	x := amount.FromFloat(100)
	y := amount.FromFloat(7)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.Div(y)
	}
}

func BenchmarkFast_Div(b *testing.B) {
	x := amountfast.FromFloat(100)
	y := amountfast.FromFloat(7)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.Div(y)
	}
}

func BenchmarkAmount_Round(b *testing.B) {
	x := amount.FromFloat(1234.56789)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.Round(2)
	}
}

func BenchmarkFast_Round(b *testing.B) {
	x := amountfast.FromFloat(1234.56789)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = x.Round(2)
	}
}
