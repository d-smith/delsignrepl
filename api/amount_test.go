package api

import (
	"math/big"
	"strings"
	"testing"
)

func TestParseWei(t *testing.T) {
	two256 := new(big.Int).Lsh(big.NewInt(1), 256)
	for _, bad := range []string{"-1", "007", "+3", "1e3", "", " 1", "1.0", "0x10", two256.String()} {
		if _, err := ParseWei(bad); err == nil {
			t.Errorf("ParseWei(%q) accepted", bad)
		}
	}
	max := new(big.Int).Sub(two256, big.NewInt(1)).String()
	for _, good := range []string{"0", "1", "1000000000000000", max} {
		v, err := ParseWei(good)
		if err != nil || v.String() != good {
			t.Errorf("ParseWei(%q) = %v, %v", good, v, err)
		}
	}
	if len(max) != 78 {
		t.Errorf("2^256-1 has %d digits; the 78-character cap assumes 78", len(max))
	}
}

func TestFormatEther(t *testing.T) {
	big1 := func(s string) *big.Int { v, _ := new(big.Int).SetString(s, 10); return v }
	cases := map[string]string{
		"0":                           "0",
		"1":                           "0.000000000000000001",
		"900000000000000000":          "0.9",
		"1000000000000000000":         "1",
		"1000000000000000":            "0.001",
		"123456789012345678901234567": "123456789.012345678901234567",
		MaxWei.String():               "115792089237316195423570985008687907853269984665640564039457.584007913129639935",
	}
	for in, want := range cases {
		if got := FormatEther(big1(in)); got != want {
			t.Errorf("FormatEther(%s) = %s, want %s", in, got, want)
		}
	}
	if got := FormatEther(big.NewInt(-1)); got != "-0.000000000000000001" {
		t.Errorf("negative: %s", got)
	}
}

func TestValidAddress(t *testing.T) {
	a := "0x" + strings.Repeat("aB", 20)
	for _, good := range []string{a, "0X" + a[2:]} {
		if !ValidAddress(good) {
			t.Errorf("rejected %q", good)
		}
	}
	for _, bad := range []string{"", a[2:], a + "0", a[:41], "0x" + strings.Repeat("g", 40), " " + a} {
		if ValidAddress(bad) {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestSendMessage(t *testing.T) {
	r := SendRequest{Source: "0xS", Dest: "0xD", Amount: "10", RequestID: "id", Expiry: 1800000060, Sig: "ignored"}
	if got := SendMessage(r); got != "0xS0xD10id1800000060" {
		t.Errorf("got %q", got)
	}
}
