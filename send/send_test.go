package send

import (
	"strings"
	"testing"
)

func TestWeiDigitsOnly(t *testing.T) {
	for _, ok := range []string{"", "0", "123", strings.Repeat("9", 78)} {
		if !weiDigitsOnly(ok, 0) {
			t.Errorf("refused %q", ok)
		}
	}
	for _, bad := range []string{"-5", "+3", "1e3", "1.5", " 1", strings.Repeat("9", 79)} {
		if weiDigitsOnly(bad, 0) {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestValidateSend(t *testing.T) {
	dest, wei, err := validateSend(src, "  "+dst+"\t", "1000")
	if err != nil || dest != dst || wei.String() != "1000" {
		t.Errorf("got %q, %v, %v", dest, wei, err)
	}
	cases := map[string][2]string{
		"empty amount":  {dst, ""},
		"zero":          {dst, "0"},
		"leading zeros": {dst, "007"},
		"bad dest":      {"0x123", "1"},
		"empty dest":    {"", "1"},
	}
	for name, c := range cases {
		if _, _, err := validateSend(src, c[0], c[1]); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, _, err := validateSend("", dst, "1"); err == nil {
		t.Error("empty source accepted")
	}
}
