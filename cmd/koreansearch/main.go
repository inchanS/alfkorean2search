// Command koreansearch is the single universal binary backing the Alfred
// script filter in the AlfKorean2Search workflow. It dispatches by subcommand:
//
//	koreansearch <handler> [args...]   run a search handler (search)
//	koreansearch _update-check         background: refresh update availability
//
// When the final argument is the update magic ("workflow:update"), it downloads
// and installs the latest release instead of running a handler.
package main

import (
	"os"

	"golang.org/x/text/unicode/norm"

	"github.com/inchanS/AlfKorean2Search/internal/alfred"
	"github.com/inchanS/AlfKorean2Search/internal/handlers"
	"github.com/inchanS/AlfKorean2Search/internal/update"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		return
	}

	normalizeArgs(args)

	cmd := args[0]

	// Detached background worker: refresh update info, no Alfred output.
	if cmd == "_update-check" {
		update.RunCheck()
		return
	}

	// Update magic: the query autocompletes to update.Magic, so any handler
	// invocation ending in it means "install the update".
	if args[len(args)-1] == update.Magic {
		update.Install(alfred.New())
		return
	}

	handlers.Dispatch(cmd, args[1:])
}

// normalizeArgs rewrites each argument to Unicode NFC in place. macOS delivers
// argv in NFD (decomposed) form, but stdict.korean.go.kr expects NFC
// (composed) — e.g. the Hangul "사랑" must be composed syllables, not separate
// jamo. This reproduces what the old Python library did. ASCII arguments
// (handler names, the update magic) are unaffected.
func normalizeArgs(args []string) {
	for i, a := range args {
		args[i] = norm.NFC.String(a)
	}
}
