// Package urlx replicates Python's urllib.parse.quote(text, safe='') so the
// display/action URLs are percent-encoded consistently.
package urlx

import "strings"

const upperhex = "0123456789ABCDEF"

// isUnreserved reports whether c is an RFC 3986 unreserved character, which
// quote(safe='') leaves untouched. Everything else (including '/') is encoded.
func isUnreserved(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '-' || c == '_' || c == '.' || c == '~':
		return true
	}
	return false
}

// Quote percent-encodes text like Python's quote(text, safe=''): UTF-8 bytes
// outside the unreserved set become %XX (uppercase hex), spaces become %20.
func Quote(text string) string {
	var b strings.Builder
	// Worst case every byte expands to 3 chars.
	b.Grow(len(text) * 3)
	for i := 0; i < len(text); i++ {
		c := text[i]
		if isUnreserved(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(upperhex[c>>4])
		b.WriteByte(upperhex[c&0x0f])
	}
	return b.String()
}
