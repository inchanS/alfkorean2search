package urlx

import "testing"

func TestQuote(t *testing.T) {
	cases := map[string]string{
		"hello":  "hello",
		"a b":    "a%20b",     // space -> %20 (not '+')
		"a/b":    "a%2Fb",     // '/' is encoded (safe='')
		"-_.~":   "-_.~",      // unreserved untouched
		"005930": "005930",    // digits untouched
		"c++":    "c%2B%2B",   // reserved encoded
		"한":      "%ED%95%9C", // UTF-8 bytes, uppercase hex
		"사랑":     "%EC%82%AC%EB%9E%91",
	}
	for in, want := range cases {
		if got := Quote(in); got != want {
			t.Errorf("Quote(%q) = %q, want %q", in, got, want)
		}
	}
}
