package amountfast_test

import (
	"encoding/json"
	"encoding/xml"
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/wawafc/go-utils/amount"
	"github.com/wawafc/go-utils/amountfast"
	"go.mongodb.org/mongo-driver/bson"
)

// -------------------- constructors --------------------

func TestFromFloat(t *testing.T) {
	cases := []struct {
		in   float64
		want int64 // scaled
	}{
		{0, 0},
		{1, 1_000_000},
		{-1, -1_000_000},
		{1.25, 1_250_000},
		{0.1, 100_000}, // must round, not truncate, repeating binary
		{0.2, 200_000},
		{0.3, 300_000},
		{-0.1, -100_000},
		{1234567.89, 1_234_567_890_000},
	}
	for _, c := range cases {
		if got := amountfast.FromFloat(c.in).Scaled(); got != c.want {
			t.Errorf("FromFloat(%v): got scaled=%d, want %d", c.in, got, c.want)
		}
	}
}

func TestFromInt_Variants(t *testing.T) {
	if amountfast.FromInt(5).Scaled() != 5_000_000 {
		t.Error("FromInt(5)")
	}
	if amountfast.FromInt64(-3).Scaled() != -3_000_000 {
		t.Error("FromInt64(-3)")
	}
	if amountfast.FromInt32(7).Scaled() != 7_000_000 {
		t.Error("FromInt32(7)")
	}
	if amountfast.FromScaled(123).Scaled() != 123 {
		t.Error("FromScaled(123)")
	}
}

func TestFromString(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"0", 0, false},
		{"1", 1_000_000, false},
		{"-1", -1_000_000, false},
		{"+1.5", 1_500_000, false},
		{"  1.25  ", 1_250_000, false},
		{"", 0, false},
		{"0.1", 100_000, false},
		{".5", 500_000, false},
		{"1.", 1_000_000, false},
		{"1.23456789", 1_234_567, false}, // truncated past 6 places
		{"-0.000001", -1, false},
		{"abc", 0, true},
		{"1.2.3", 0, true},
	}
	for _, c := range cases {
		got, err := amountfast.FromString(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("FromString(%q): expected error, got none", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("FromString(%q): unexpected error %v", c.in, err)
			continue
		}
		if got.Scaled() != c.want {
			t.Errorf("FromString(%q): got scaled=%d, want %d", c.in, got.Scaled(), c.want)
		}
	}
}

func TestFromString_PreservesRaw(t *testing.T) {
	v, _ := amountfast.FromString("1.2500")
	if v.Raw() != "1.2500" {
		t.Errorf("Raw: got %q, want %q", v.Raw(), "1.2500")
	}
}

func TestFromString_EdgeCases(t *testing.T) {
	// bare dot → zero (matches legacy behaviour)
	if v, err := amountfast.FromString("."); err != nil || !v.IsZero() {
		t.Errorf("'.': err=%v scaled=%d, want zero", err, v.Scaled())
	}
	// whitespace-only → zero
	if v, err := amountfast.FromString("   \t\n"); err != nil || !v.IsZero() {
		t.Errorf("whitespace: err=%v scaled=%d, want zero", err, v.Scaled())
	}
	// sign only → error
	for _, s := range []string{"-", "+", "   -   ", "  +"} {
		if _, err := amountfast.FromString(s); err == nil {
			t.Errorf("%q: expected error", s)
		}
	}
	// overflow: int part > 13 digits
	if _, err := amountfast.FromString("99999999999999"); err == nil {
		t.Error("14-digit int: expected overflow error")
	}
	// overflow: total > MaxInt64 (9_223_372_036_854.775_807 is the cap)
	if _, err := amountfast.FromString("9223372036855"); err == nil {
		t.Error("value beyond MaxInt64/Scale: expected overflow error")
	}
	// boundary: just below MaxInt64/Scale, with max fractional fitting in int64
	if v, err := amountfast.FromString("9223372036854.775807"); err != nil {
		t.Errorf("MaxInt64 boundary: unexpected error %v", err)
	} else if v.Scaled() != 9223372036854775807 {
		t.Errorf("MaxInt64 boundary: got %d", v.Scaled())
	}
	// truncation past 6 decimal places
	if v, _ := amountfast.FromString("0.1234567999"); v.Scaled() != 123456 {
		t.Errorf("truncation: got scaled=%d, want 123456", v.Scaled())
	}
	// leading zeros
	if v, _ := amountfast.FromString("000123.4500"); v.Scaled() != 123_450_000 {
		t.Errorf("leading zeros: got scaled=%d", v.Scaled())
	}
	// character after dot that's not digit
	if _, err := amountfast.FromString("1.2a"); err == nil {
		t.Error("1.2a: expected error")
	}
}

func TestFromDecimal(t *testing.T) {
	d, _ := decimal.NewFromString("1234.56789012") // >6 fractional digits
	v := amountfast.FromDecimal(d)
	if v.Scaled() != 1_234_567_890 {
		t.Errorf("FromDecimal: got %d, want 1234567890", v.Scaled())
	}
}

// -------------------- accessors --------------------

func TestAccessors(t *testing.T) {
	x, _ := amountfast.FromString("-1234.75")
	if got, want := x.Float64(), -1234.75; got != want {
		t.Errorf("Float64: got %v, want %v", got, want)
	}
	if got, want := x.Int64(), int64(-1234); got != want {
		t.Errorf("Int64: got %v, want %v", got, want)
	}
	if got, want := x.Int32(), int32(-1234); got != want {
		t.Errorf("Int32: got %v, want %v", got, want)
	}
	if got, want := x.Int(), -1234; got != want {
		t.Errorf("Int: got %v, want %v", got, want)
	}
	if got, want := x.Scaled(), int64(-1234_750_000); got != want {
		t.Errorf("Scaled: got %v, want %v", got, want)
	}
	if got, want := x.Decimal().String(), "-1234.75"; got != want {
		t.Errorf("Decimal: got %v, want %v", got, want)
	}
}

// -------------------- arithmetic --------------------

func TestAddSub(t *testing.T) {
	a := amountfast.FromFloat(1.25)
	b := amountfast.FromFloat(0.75)
	if got := a.Add(b).String(); got != "2" {
		t.Errorf("Add: got %s, want 2", got)
	}
	if got := a.Sub(b).String(); got != "0.5" {
		t.Errorf("Sub: got %s, want 0.5", got)
	}
	if got := b.Sub(a).String(); got != "-0.5" {
		t.Errorf("Sub negative: got %s, want -0.5", got)
	}
}

func TestMulInt(t *testing.T) {
	x := amountfast.FromFloat(1.25)
	if got := x.MulInt(4).String(); got != "5" {
		t.Errorf("MulInt(4): got %s", got)
	}
	if got := x.MulInt(-2).String(); got != "-2.5" {
		t.Errorf("MulInt(-2): got %s", got)
	}
}

func TestDivInt(t *testing.T) {
	x := amountfast.FromFloat(10)
	if got := x.DivInt(4).String(); got != "2.5" {
		t.Errorf("DivInt(4): got %s", got)
	}
	if got := x.DivInt(0).String(); got != "0" {
		t.Errorf("DivInt(0) should return zero, got %s", got)
	}
}

func TestDiv_ByZero(t *testing.T) {
	x := amountfast.FromFloat(10)
	z := amountfast.Zero
	if got := x.Div(z).String(); got != "0" {
		t.Errorf("Div by zero: got %s", got)
	}
}

func TestNegAbsCopy(t *testing.T) {
	x := amountfast.FromFloat(-5.25)
	if x.Neg().String() != "5.25" {
		t.Error("Neg")
	}
	if x.Abs().String() != "5.25" {
		t.Error("Abs")
	}
	y := x.Copy()
	if y.Scaled() != x.Scaled() {
		t.Error("Copy")
	}
}

// -------------------- comparisons --------------------

func TestComparisons(t *testing.T) {
	zero := amountfast.Zero
	pos := amountfast.FromInt(5)
	neg := amountfast.FromInt(-5)

	if !zero.IsZero() || pos.IsZero() {
		t.Error("IsZero")
	}
	if !pos.IsPositive() || neg.IsPositive() || zero.IsPositive() {
		t.Error("IsPositive")
	}
	if !pos.IsPositiveOrZero() || !zero.IsPositiveOrZero() || neg.IsPositiveOrZero() {
		t.Error("IsPositiveOrZero")
	}
	if !neg.IsNegative() || pos.IsNegative() || zero.IsNegative() {
		t.Error("IsNegative")
	}
	if !neg.IsNegativeOrZero() || !zero.IsNegativeOrZero() || pos.IsNegativeOrZero() {
		t.Error("IsNegativeOrZero")
	}
	if !pos.Equal(amountfast.FromInt(5)) || pos.Equal(neg) {
		t.Error("Equal")
	}
	if !pos.GreaterThan(neg) || pos.GreaterThan(pos) {
		t.Error("GreaterThan")
	}
	if !pos.GreaterThanOrEqual(pos) || !pos.GreaterThanOrEqual(neg) || neg.GreaterThanOrEqual(pos) {
		t.Error("GreaterThanOrEqual")
	}
	if !neg.LessThan(pos) || pos.LessThan(pos) {
		t.Error("LessThan")
	}
	if !neg.LessThanOrEqual(neg) || pos.LessThanOrEqual(neg) {
		t.Error("LessThanOrEqual")
	}
	if pos.Cmp(neg) != 1 || neg.Cmp(pos) != -1 || pos.Cmp(pos) != 0 {
		t.Error("Cmp")
	}
	if pos.Sign() != 1 || neg.Sign() != -1 || zero.Sign() != 0 {
		t.Error("Sign")
	}
}

// -------------------- rounding --------------------

type roundCase struct {
	in    string
	round string
	bank  string
	ceil  string
	floor string
	up    string
	down  string
}

func TestRounding_Precision2(t *testing.T) {
	cases := []roundCase{
		{"1.234", "1.23", "1.23", "1.24", "1.23", "1.24", "1.23"},
		{"1.235", "1.24", "1.24", "1.24", "1.23", "1.24", "1.23"}, // half: round→up; bank→even(4)
		{"1.245", "1.25", "1.24", "1.25", "1.24", "1.25", "1.24"}, // bank: half→even(4)
		{"-1.235", "-1.24", "-1.24", "-1.23", "-1.24", "-1.24", "-1.23"},
		{"1.230", "1.23", "1.23", "1.23", "1.23", "1.23", "1.23"}, // exact: no rounding
		{"-1.230", "-1.23", "-1.23", "-1.23", "-1.23", "-1.23", "-1.23"},
		{"1.999", "2", "2", "2", "1.99", "2", "1.99"},
		{"-1.999", "-2", "-2", "-1.99", "-2", "-2", "-1.99"},
	}
	for _, c := range cases {
		v, _ := amountfast.FromString(c.in)
		if got := v.Round(2).String(); got != c.round {
			t.Errorf("Round(%s, 2): got %s, want %s", c.in, got, c.round)
		}
		if got := v.RoundBank(2).String(); got != c.bank {
			t.Errorf("RoundBank(%s, 2): got %s, want %s", c.in, got, c.bank)
		}
		if got := v.RoundCeil(2).String(); got != c.ceil {
			t.Errorf("RoundCeil(%s, 2): got %s, want %s", c.in, got, c.ceil)
		}
		if got := v.RoundFloor(2).String(); got != c.floor {
			t.Errorf("RoundFloor(%s, 2): got %s, want %s", c.in, got, c.floor)
		}
		if got := v.RoundUp(2).String(); got != c.up {
			t.Errorf("RoundUp(%s, 2): got %s, want %s", c.in, got, c.up)
		}
		if got := v.RoundDown(2).String(); got != c.down {
			t.Errorf("RoundDown(%s, 2): got %s, want %s", c.in, got, c.down)
		}
	}
}

func TestRounding_PrecisionBounds(t *testing.T) {
	v, _ := amountfast.FromString("1.234567")
	// precision >= 6 is a no-op
	if v.Round(6).Scaled() != v.Scaled() {
		t.Error("Round(6) should be no-op")
	}
	if v.RoundDown(10).Scaled() != v.Scaled() {
		t.Error("RoundDown(10) should be no-op")
	}
	// negative precision clamped to 0
	if v.Round(-1).String() != "1" {
		t.Errorf("Round(-1): got %s", v.Round(-1).String())
	}
}

func TestTruncate(t *testing.T) {
	v, _ := amountfast.FromString("1.234567")
	if got := v.Truncate(2).String(); got != "1.23" {
		t.Errorf("Truncate(2): got %s", got)
	}
	if got := v.Truncate(5).String(); got != "1.23456" {
		t.Errorf("Truncate(5): got %s", got)
	}
}

// RoundCash parity with shopspring/decimal
func TestRoundCash_Parity(t *testing.T) {
	intervals := []uint8{5, 10, 25, 50, 100}
	inputs := []string{"1.23", "1.255", "1.275", "-1.255", "7.89"}
	for _, interval := range intervals {
		for _, s := range inputs {
			d, _ := decimal.NewFromString(s)
			v, _ := amountfast.FromString(s)
			want := d.RoundCash(interval).String()
			got := v.RoundCash(interval).String()
			if got != want {
				t.Errorf("RoundCash(%s, %d): got %s, want %s", s, interval, got, want)
			}
		}
	}
}

// -------------------- String / trailing zeros --------------------

func TestString_Formatting(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0", "0"},
		{"1", "1"},
		{"-1", "-1"},
		{"1.25", "1.25"},
		{"1.200", "1.2"},   // trailing zeros stripped
		{"1.000000", "1"},  // all zeros stripped with no dot
		{"-0.000001", "-0.000001"},
	}
	for _, c := range cases {
		v, _ := amountfast.FromString(c.in)
		if got := v.String(); got != c.want {
			t.Errorf("String(%s): got %s, want %s", c.in, got, c.want)
		}
	}
}

// -------------------- JSON --------------------

func TestJSON_RoundTrip(t *testing.T) {
	v, _ := amountfast.FromString("1234.567")
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(b) != "1234.567" {
		t.Errorf("Marshal: got %s", string(b))
	}
	var w amountfast.Value
	if err := json.Unmarshal(b, &w); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if w.Scaled() != v.Scaled() {
		t.Errorf("round-trip: got %d, want %d", w.Scaled(), v.Scaled())
	}
}

func TestJSON_UnmarshalNull(t *testing.T) {
	var v amountfast.Value
	if err := json.Unmarshal([]byte(`null`), &v); err != nil {
		t.Fatalf("Unmarshal null: %v", err)
	}
	if !v.IsZero() {
		t.Errorf("null should be zero, got %d", v.Scaled())
	}
}

func TestJSON_UnmarshalQuoted(t *testing.T) {
	var v amountfast.Value
	if err := json.Unmarshal([]byte(`"12.5"`), &v); err != nil {
		t.Fatalf("Unmarshal quoted: %v", err)
	}
	if v.String() != "12.5" {
		t.Errorf("quoted: got %s, want 12.5", v.String())
	}
}

// -------------------- XML --------------------

type xmlDoc struct {
	XMLName xml.Name          `xml:"doc"`
	Amount  amountfast.Value  `xml:"amt"`
}

func TestXML_RoundTrip(t *testing.T) {
	in := xmlDoc{Amount: amountfast.FromFloat(12.5)}
	b, err := xml.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out xmlDoc
	if err := xml.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Amount.Scaled() != in.Amount.Scaled() {
		t.Errorf("round-trip: got %d, want %d", out.Amount.Scaled(), in.Amount.Scaled())
	}
}

// -------------------- Text --------------------

func TestText_RoundTrip(t *testing.T) {
	v := amountfast.FromFloat(12.5)
	b, _ := v.MarshalText()
	if string(b) != "12.5" {
		t.Errorf("MarshalText: got %s", b)
	}
	var w amountfast.Value
	if err := w.UnmarshalText([]byte("12.5")); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if w.Scaled() != v.Scaled() {
		t.Errorf("mismatch")
	}
}

// -------------------- YAML --------------------

// Drives UnmarshalYAML directly with mock unmarshalers, avoiding an external
// yaml package dependency. Covers every branch of the type switch.
func TestYAML_Unmarshal(t *testing.T) {
	cases := []struct {
		name string
		feed interface{}
		want int64
	}{
		{"string", "12.5", 12_500_000},
		{"int", int(42), 42_000_000},
		{"int64", int64(-7), -7_000_000},
		{"float64", float64(2.5), 2_500_000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var v amountfast.Value
			unmarshal := func(dst interface{}) error {
				p := dst.(*interface{})
				*p = c.feed
				return nil
			}
			if err := v.UnmarshalYAML(unmarshal); err != nil {
				t.Fatalf("UnmarshalYAML: %v", err)
			}
			if v.Scaled() != c.want {
				t.Errorf("got %d, want %d", v.Scaled(), c.want)
			}
		})
	}

	t.Run("Marshal", func(t *testing.T) {
		v := amountfast.FromFloat(12.5)
		got, err := v.MarshalYAML()
		if err != nil {
			t.Fatalf("MarshalYAML: %v", err)
		}
		if got != "12.5" {
			t.Errorf("got %v, want 12.5", got)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		var v amountfast.Value
		unmarshal := func(dst interface{}) error {
			p := dst.(*interface{})
			*p = []int{1, 2, 3}
			return nil
		}
		if err := v.UnmarshalYAML(unmarshal); err == nil {
			t.Error("expected error for unsupported type")
		}
	})
}

// -------------------- BSON --------------------

func TestBSON_RoundTrip(t *testing.T) {
	type doc struct {
		Amt amountfast.Value `bson:"amt"`
	}
	in := doc{Amt: amountfast.FromFloat(12.5)}
	raw, err := bson.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out doc
	if err := bson.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Amt.Scaled() != in.Amt.Scaled() {
		t.Errorf("round-trip: got %d, want %d", out.Amt.Scaled(), in.Amt.Scaled())
	}
}

// Exercises Int32 / Int64 / Double branches of UnmarshalBSONValue by marshaling
// documents with scalar types and decoding into Value.
func TestBSON_ScalarTypes(t *testing.T) {
	type doc struct {
		Amt amountfast.Value `bson:"amt"`
	}
	cases := []struct {
		name string
		in   interface{}
		want int64
	}{
		{"int32", bson.M{"amt": int32(5)}, 5_000_000},
		{"int64", bson.M{"amt": int64(-7)}, -7_000_000},
		{"double", bson.M{"amt": float64(2.5)}, 2_500_000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw, err := bson.Marshal(c.in)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var out doc
			if err := bson.Unmarshal(raw, &out); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if out.Amt.Scaled() != c.want {
				t.Errorf("got %d, want %d", out.Amt.Scaled(), c.want)
			}
		})
	}
}

// -------------------- SQL driver --------------------

func TestSQL_Value(t *testing.T) {
	v := amountfast.FromFloat(12.5)
	got, err := v.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if got != "12.5" {
		t.Errorf("Value: got %v", got)
	}
}

func TestSQL_Scan(t *testing.T) {
	cases := []struct {
		in   interface{}
		want int64
	}{
		{"12.5", 12_500_000},
		{[]byte("12.5"), 12_500_000},
		{int64(3), 3_000_000},
		{float64(2.5), 2_500_000},
	}
	for _, c := range cases {
		var v amountfast.Value
		if err := v.Scan(c.in); err != nil {
			t.Fatalf("Scan(%v): %v", c.in, err)
		}
		if v.Scaled() != c.want {
			t.Errorf("Scan(%v): got %d, want %d", c.in, v.Scaled(), c.want)
		}
	}
	var v amountfast.Value
	if err := v.Scan(struct{}{}); err == nil {
		t.Error("Scan(struct): expected error")
	}
}

// -------------------- Arithmetic parity vs amount --------------------

// Values truncated to 6 decimal places to match amountfast's scale.
func TestArithmetic_ParityVsAmount(t *testing.T) {
	cases := [][2]string{
		{"1.25", "0.75"},
		{"100", "7"},
		{"-2.5", "4"},
		{"1234.567", "0.89"},
		{"-1000.123456", "2.5"},
		{"0.000001", "1000000"},
	}
	for _, c := range cases {
		ax, _ := amount.FromString(c[0])
		ay, _ := amount.FromString(c[1])
		fx, _ := amountfast.FromString(c[0])
		fy, _ := amountfast.FromString(c[1])

		if got, want := fx.Add(fy).String(), ax.Add(ay).Truncate(6).String(); got != want {
			t.Errorf("Add(%s, %s): got %s, want %s", c[0], c[1], got, want)
		}
		if got, want := fx.Sub(fy).String(), ax.Sub(ay).Truncate(6).String(); got != want {
			t.Errorf("Sub(%s, %s): got %s, want %s", c[0], c[1], got, want)
		}
		if got, want := fx.Mul(fy).String(), ax.Mul(ay).Truncate(6).String(); got != want {
			t.Errorf("Mul(%s, %s): got %s, want %s", c[0], c[1], got, want)
		}
		if !fy.IsZero() {
			if got, want := fx.Div(fy).String(), ax.Div(ay).Truncate(6).String(); got != want {
				t.Errorf("Div(%s, %s): got %s, want %s", c[0], c[1], got, want)
			}
		}
	}
}

// -------------------- overflow / edge --------------------

func TestFromFloat_NaN(t *testing.T) {
	// NaN: behavior is platform-defined; just ensure no panic.
	_ = amountfast.FromFloat(math.NaN())
}
