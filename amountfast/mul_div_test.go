package amountfast_test

import (
	"math/bits"
	"math/rand"
	"testing"

	"github.com/wawafc/go-utils/amount"
	"github.com/wawafc/go-utils/amountfast"
)

func TestMul_TableParity(t *testing.T) {
	cases := []struct{ x, y string }{
		{"1.25", "0.75"},
		{"1000", "0.0001"},
		{"-2.5", "4"},
		{"-3.5", "-7"},
		{"0", "123.456"},
		{"0.000001", "0.000001"},
		{"999999.999999", "2"},
		{"1234567.89", "0.111111"},
	}
	for _, c := range cases {
		a, _ := amount.FromString(c.x)
		b, _ := amount.FromString(c.y)
		fa, _ := amountfast.FromString(c.x)
		fb, _ := amountfast.FromString(c.y)

		want := a.Mul(b).Truncate(6).String()
		got := fa.Mul(fb).String()
		if want != got {
			t.Errorf("Mul(%s, %s): want %s, got %s", c.x, c.y, want, got)
		}
	}
}

func TestDiv_TableParity(t *testing.T) {
	cases := []struct{ x, y string }{
		{"1", "3"},
		{"10", "4"},
		{"-7.5", "2"},
		{"-100", "-4"},
		{"0", "123"},
		{"999999", "0.000001"},
		{"1.5", "0.000002"},
	}
	for _, c := range cases {
		a, _ := amount.FromString(c.x)
		b, _ := amount.FromString(c.y)
		fa, _ := amountfast.FromString(c.x)
		fb, _ := amountfast.FromString(c.y)

		want := a.Div(b).Truncate(6).String()
		got := fa.Div(fb).String()
		if want != got {
			t.Errorf("Div(%s, %s): want %s, got %s", c.x, c.y, want, got)
		}
	}
}

// Randomised check: exercise both fast (hi==0) and slow (hi!=0) paths by
// varying magnitudes, and verify against a reference implementation that
// always uses the 128-bit slow path.
//
// Operand magnitudes are bounded to keep the quotient within int64 so neither
// bits.Div64 nor the fast path can overflow — we're testing path parity here,
// not overflow behaviour (both paths would panic in bits.Div64 on overflow).
func TestMul_FastPathParity(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	hitFast, hitSlow := 0, 0
	for i := 0; i < 50000; i++ {
		// up to 40 bits each → product up to 80 bits (hi nonzero sometimes),
		// quotient up to 60 bits (always fits int64).
		sx := randBounded(r, 40)
		sy := randBounded(r, 40)
		x := amountfast.FromScaled(sx)
		y := amountfast.FromScaled(sy)
		got := x.Mul(y).Scaled()
		want := mulRef(sx, sy)
		if got != want {
			t.Fatalf("Mul(scaled=%d, scaled=%d): got %d, want %d", sx, sy, got, want)
		}
		if _, hi := bitsMulAbs(sx, sy); hi == 0 {
			hitFast++
		} else {
			hitSlow++
		}
	}
	if hitFast == 0 || hitSlow == 0 {
		t.Fatalf("both paths must be exercised (fast=%d, slow=%d)", hitFast, hitSlow)
	}
}

func TestDiv_FastPathParity(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	hitFast, hitSlow := 0, 0
	for i := 0; i < 50000; i++ {
		// a up to 50 bits → a*Scale up to 70 bits (hi nonzero sometimes).
		// b >= 2^10 guarantees quotient = a*Scale/b < 2^60, no bits.Div64 overflow.
		sx := randBounded(r, 50)
		sy := randBounded(r, 30) // up to 30 bits
		if absU(sy) < 1<<10 {
			continue // keep divisor large enough to avoid quotient overflow
		}
		x := amountfast.FromScaled(sx)
		y := amountfast.FromScaled(sy)
		got := x.Div(y).Scaled()
		want := divRef(sx, sy)
		if got != want {
			t.Fatalf("Div(scaled=%d, scaled=%d): got %d, want %d", sx, sy, got, want)
		}
		_, hi := bitsMulAbs(sx, amountfast.Scale)
		if hi == 0 {
			hitFast++
		} else {
			hitSlow++
		}
	}
	if hitFast == 0 || hitSlow == 0 {
		t.Fatalf("both paths must be exercised (fast=%d, slow=%d)", hitFast, hitSlow)
	}
}

// randBounded returns a random signed int64 with |v| < 2^maxBits.
func randBounded(r *rand.Rand, maxBits int) int64 {
	b := r.Intn(maxBits + 1)
	mask := int64(1)<<b - 1
	if mask < 1 {
		mask = 1
	}
	v := r.Int63() & mask
	if r.Intn(2) == 0 {
		v = -v
	}
	return v
}

func bitsMulAbs(x, y int64) (lo, hi uint64) {
	hi, lo = bits.Mul64(absU(x), absU(y))
	return
}

// mulRef is a reference implementation using only the 128-bit slow path.
func mulRef(x, y int64) int64 {
	neg := (x < 0) != (y < 0)
	a := absU(x)
	b := absU(y)
	hi, lo := bits.Mul64(a, b)
	q, _ := bits.Div64(hi, lo, amountfast.Scale)
	r := int64(q)
	if neg {
		r = -r
	}
	return r
}

func divRef(x, y int64) int64 {
	if y == 0 {
		return 0
	}
	neg := (x < 0) != (y < 0)
	a := absU(x)
	b := absU(y)
	hi, lo := bits.Mul64(a, amountfast.Scale)
	q, _ := bits.Div64(hi, lo, b)
	r := int64(q)
	if neg {
		r = -r
	}
	return r
}

func absU(x int64) uint64 {
	if x < 0 {
		return uint64(-x)
	}
	return uint64(x)
}
