package readline

import (
	"regexp"

	"github.com/evilsocket/islazy/tui"
)

// FindMode defines how the autocomplete suggestions display
type FindMode int

const (
	// HistoryFind - Searching through history
	HistoryFind = iota

	// CompletionFind - Searching through completion items
	CompletionFind
)

func (rl *Instance) backspaceTabFind() {
	if len(rl.tfLine) > 0 {
		rl.tfLine = rl.tfLine[:len(rl.tfLine)-1]
	}
	rl.updateTabFind([]rune{})
}

func (rl *Instance) updateTabFind(r []rune) {

	rl.tfLine = append(rl.tfLine, r...)

	// Depending on search type, we give different hints
	switch rl.regexpMode {
	case HistoryFind:
		rl.hintText = append([]rune("History search: "), rl.tfLine...)
	case CompletionFind:
		rl.hintText = append([]rune("Completion search: "), rl.tfLine...)
	}

	// The search regex is common to all search modes
	var err error
	rl.regexSearch, err = regexp.Compile("(?i)" + string(rl.tfLine))
	if err != nil {
		rl.hintText = []rune(tui.Red("Failed to match search regexp"))
		return
	}

	rl.clearHelpers()
	rl.getTabCompletion()
	rl.renderHelpers()
}

func (rl *Instance) resetTabFind() {
	rl.modeTabFind = false
	rl.tfLine = []rune{}
	if rl.modeAutoFind {
		rl.hintText = []rune{}
	} else {
		rl.hintText = []rune("Cancelled regexp suggestion find.")
	}

	rl.modeAutoFind = false // Added, because otherwise it gets stuck on search completions

	rl.clearHelpers()
	rl.getTabCompletion()
	rl.renderHelpers()
}
