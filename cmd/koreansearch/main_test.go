package main

import "testing"

func TestNormalizeArgs(t *testing.T) {
	// Build NFD (decomposed) inputs explicitly via code points, independent of
	// this file's own encoding.
	//   "사랑" as conjoining jamo: ᄉ ᅡ ᄅ ᅡ ᆼ
	loveNFD := string([]rune{0x1109, 0x1161, 0x1105, 0x1161, 0x11BC})
	//   "학교" as conjoining jamo: ᄒ ᅡ ᆨ ᄀ ᅭ
	schoolNFD := string([]rune{0x1112, 0x1161, 0x11A8, 0x1100, 0x116D})

	args := []string{"search", loveNFD, schoolNFD, "ascii"}
	normalizeArgs(args)

	if want := "사랑"; args[1] != want {
		t.Errorf("Hangul not composed to NFC: got % x, want % x", []byte(args[1]), []byte(want))
	}
	if want := "학교"; args[2] != want {
		t.Errorf("Hangul not composed to NFC: got % x, want % x", []byte(args[2]), []byte(want))
	}
	// Handler names and ASCII are untouched.
	if args[0] != "search" || args[3] != "ascii" {
		t.Errorf("ASCII args changed: %v", args)
	}
}
