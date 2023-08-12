package completion

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/term"
)

// group is used to structure different types of completions with different
// display types, autosuffix removal matchers, under their tag heading.
type group struct {
	tag             string        // Printed on top of the group's completions
	rows            [][]Candidate // Values are grouped by aliases/rows, with computed paddings.
	noSpace         SuffixMatcher // Suffixes to remove if a space or non-nil character is entered after the completion.
	columnsWidth    []int         // Computed width for each column of completions, when aliases
	listSeparator   string        // This is used to separate completion candidates from their descriptions.
	list            bool          // Force completions to be listed instead of grided
	noSort          bool          // Don't sort completions
	aliased         bool          // Are their aliased completions
	preserveEscapes bool          // Preserve escape sequences in the completion inserted values.
	isCurrent       bool          // Currently cycling through this group, for highlighting choice
	maxLength       int           // Each group can be limited in the number of comps offered
	tcMaxLength     int           // Used when display is map/list, for determining message width
	maxDescWidth    int
	maxCellLength   int

	// Selectors (position/bounds) management
	posX int
	posY int
	maxX int
	maxY int
}

func (e *Engine) group(comps Values) {
	// Compute hints once we found either any errors,
	// or if we have no completions but usage strings.
	defer func() {
		e.hintCompletions(comps)
	}()

	// Nothing else to do if no completions
	if len(comps.values) == 0 {
		return
	}

	// Apply the prefix to the completions, and filter out any
	// completions that don't match, optionally ignoring case.
	matchCase := e.config.GetBool("completion-ignore-case")
	comps.values = comps.values.FilterPrefix(e.prefix, !matchCase)

	comps.values.EachTag(func(tag string, values RawValues) {
		// Separate the completions that have a description and
		// those which don't, and devise if there are aliases.
		vals, noDescVals, aliased := e.groupValues(&comps, values)

		// Create a "first" group with the "first" grouped values
		e.newGroup(comps, tag, vals, aliased)

		// If we have a remaining group of values without descriptions,
		// we will print and use them in a separate, anonymous group.
		if len(noDescVals) > 0 {
			e.newGroup(comps, "", noDescVals, false)
		}
	})

	e.justifyGroups(comps)
}

// groupValues separates values based on whether they have descriptions, or are aliases of each other.
func (e *Engine) groupValues(comps *Values, values RawValues) (vals, noDescVals RawValues, aliased bool) {
	var descriptions []string

	prefix := ""
	if e.prefix != "\"\"" && e.prefix != "''" {
		prefix = e.prefix
	}

	for _, val := range values {
		// Ensure all values have a display string.
		if val.Display == "" {
			val.Display = val.Value
		}

		// Currently this is because errors are passed as completions.
		if strings.HasPrefix(val.Value, prefix+"ERR") && val.Value == prefix+"_" {
			comps.Messages.Add(color.FgRed + val.Display + val.Description)

			continue
		}

		// Grid completions
		if val.Description == "" {
			noDescVals = append(noDescVals, val)

			continue
		}

		// List/map completions.
		if stringInSlice(val.Description, descriptions) {
			aliased = true
		}

		descriptions = append(descriptions, val.Description)
		vals = append(vals, val)
	}

	// if no candidates have a description, swap
	if len(vals) == 0 {
		vals = noDescVals
		noDescVals = make(RawValues, 0)
	}

	return vals, noDescVals, aliased
}

func (e *Engine) justifyGroups(values Values) {
	commandGroups := make([]*group, 0)
	maxCellLength := 0

	for _, group := range e.groups {
		// Skip groups that are not to be justified
		if _, justify := values.Pad[group.tag]; !justify {
			if _, all := values.Pad["*"]; !all {
				continue
			}
		}

		// Skip groups that are aliased or have more than one column
		if group.aliased || len(group.columnsWidth) > 1 {
			continue
		}

		commandGroups = append(commandGroups, group)

		if group.tcMaxLength > maxCellLength {
			maxCellLength = group.tcMaxLength
		}
	}

	for _, group := range commandGroups {
		group.tcMaxLength = maxCellLength
	}
}

func (e *Engine) newGroup(comps Values, tag string, vals RawValues, aliased bool) {
	grp := &group{
		tag:          tag,
		noSpace:      comps.NoSpace,
		posX:         -1,
		posY:         -1,
		aliased:      aliased,
		columnsWidth: []int{0},
	}

	// Check that all comps have a display value,
	// and begin computing some parameters.
	vals = grp.checkDisplays(vals)

	// Set sorting options, various display styles, etc.
	grp.setOptions(e, &comps, tag, vals)

	// Keep computing/devising some parameters and constraints.
	// This does not do much when we have aliased completions.
	grp.computeCells(e, vals)

	// Rearrange all candidates into a matrix of completions,
	// based on most parameters computed above.
	grp.makeMatrix(vals)

	e.groups = append(e.groups, grp)
}

func (g *group) checkDisplays(vals RawValues) RawValues {
	if g.aliased {
		return vals
	}

	if len(vals) == 0 {
		g.columnsWidth[0] = term.GetWidth() - 1
	}

	// Otherwise update the size of the longest candidate
	for _, val := range vals {
		valLen := utf8.RuneCountInString(val.Display)
		if valLen > g.columnsWidth[0] {
			g.columnsWidth[0] = valLen
		}
	}

	return vals
}

func (g *group) setOptions(eng *Engine, comps *Values, tag string, vals RawValues) {
	// Override grid/list displays
	_, g.list = comps.ListLong[tag]
	if _, all := comps.ListLong["*"]; all && len(comps.ListLong) == 1 {
		g.list = true
	}

	// Description list separator
	listSep, err := strconv.Unquote(eng.config.GetString("completion-list-separator"))
	if err != nil {
		g.listSeparator = "--"
	} else {
		g.listSeparator = listSep
	}

	// Strip escaped characters in the value component.
	preserve, yes := comps.Escapes[g.tag]
	if !yes {
		preserve, _ = comps.Escapes["*"]
	}
	g.preserveEscapes = preserve

	// Always list long commands when they have descriptions.
	if strings.HasSuffix(g.tag, "commands") && len(vals) > 0 && vals[0].Description != "" {
		g.list = true
	}

	// Description list separator
	listSep, found := comps.ListSep[tag]
	if !found {
		if allSep, found := comps.ListSep["*"]; found {
			g.listSeparator = allSep
		}
	} else {
		g.listSeparator = listSep
	}

	// Override sorting or sort if needed
	_, g.noSort = comps.NoSort[tag]
	if _, all := comps.NoSort["*"]; all && len(comps.NoSort) == 1 {
		g.noSort = true
	}

	if !g.noSort {
		sort.Slice(vals, func(i, j int) bool {
			return vals[i].Display < vals[j].Display
		})
	}
}

func (g *group) computeCells(eng *Engine, vals RawValues) {
	// Aliases will compute themselves individually, later.
	if g.aliased {
		return
	}

	if len(vals) == 0 {
		return
	}

	g.tcMaxLength = g.columnsWidth[0]

	// Each value first computes the total amount of space
	// it is going to take in a row (including the description)
	for _, val := range vals {
		candidate := g.displayTrimmed(color.Strip(val.Display))
		pad := strings.Repeat(" ", g.tcMaxLength-len(candidate))
		desc := g.descriptionTrimmed(color.Strip(val.Description))
		display := fmt.Sprintf("%s%s%s", candidate, pad+" ", desc)
		valLen := utf8.RuneCountInString(display)

		if valLen > g.maxCellLength {
			g.maxCellLength = valLen
		}
	}

	// Adapt the maximum cell size based on inputrc settings.
	compDisplayWidth := g.setMaxCellLength(eng)

	// We now have the length of the longest completion for this group,
	// so we devise how many columns we can use to display completions.
	g.setColumnsWidth(&vals, compDisplayWidth)
}

func (g *group) setMaxCellLength(eng *Engine) int {
	termWidth := term.GetWidth()

	compDisplayWidth := eng.config.GetInt("completion-display-width")
	if compDisplayWidth > termWidth {
		compDisplayWidth = -1
	}

	if compDisplayWidth > 0 && compDisplayWidth < termWidth {
		if g.maxCellLength < compDisplayWidth {
			g.maxCellLength = compDisplayWidth
		} else {
			g.maxCellLength = termWidth
		}
	}

	return compDisplayWidth
}

func (g *group) setColumnsWidth(vals *RawValues, compDisplayWidth int) {
	g.maxX = term.GetWidth() / (g.maxCellLength)
	if g.maxX < 1 {
		g.maxX = 1 // avoid a divide by zero error
	}

	if g.maxX > len(*vals) {
		g.maxX = len(*vals)
	}

	if g.list || compDisplayWidth == 0 {
		g.maxX = 1
	}

	if g.maxX > compDisplayWidth && compDisplayWidth > 0 {
		g.maxX = compDisplayWidth
	}

	// We also have the width for each column
	g.columnsWidth = make([]int, g.maxX)
	for i := 0; i < g.maxX; i++ {
		g.columnsWidth[i] = g.maxCellLength
	}
}

func (g *group) makeMatrix(vals RawValues) {
nextValue:
	for _, val := range vals {
		valLen := utf8.RuneCountInString(val.Display)

		// If we have an alias, and we must get the right
		// column and the right padding for this column.
		if g.aliased {
			for i, row := range g.rows {
				if row[0].Description == val.Description {
					g.rows[i] = append(row, val)
					g.columnsWidth = getColumnPad(g.columnsWidth, valLen, len(g.rows[i]))

					continue nextValue
				}
			}
		}

		// Else, either add it to the current row if there is still room
		// on it for this candidate, or add a new one. We only do that when
		// we know we don't have aliases, or when we don't have to display list.
		if !g.aliased && g.canFitInRow(term.GetWidth()) && !g.list {
			g.rows[len(g.rows)-1] = append(g.rows[len(g.rows)-1], val)
		} else {
			// Else create a new row, and update the row pad.
			g.rows = append(g.rows, []Candidate{val})
			if g.columnsWidth[0] < valLen+1 {
				g.columnsWidth[0] = valLen + 1
			}
		}
	}

	if g.aliased {
		g.maxX = len(g.columnsWidth)
		g.tcMaxLength = sum(g.columnsWidth) + len(g.columnsWidth) + 1
	}

	g.maxY = len(g.rows)
	if g.maxY > g.maxLength && g.maxLength != 0 {
		g.maxY = g.maxLength
	}
}

func (g *group) canFitInRow(termWidth int) bool {
	if len(g.rows) == 0 {
		return false
	}

	if termWidth/(g.maxCellLength)-1 < len(g.rows[len(g.rows)-1]) {
		return false
	}

	return true
}

//
// Usage-time functions (selecting/writing) -----------------------------------------------------------------
//

// updateIsearch - When searching through all completion groups (whether it be command history or not),
// we ask each of them to filter its own items and return the results to the shell for aggregating them.
// The rx parameter is passed, as the shell already checked that the search pattern is valid.
func (g *group) updateIsearch(eng *Engine) {
	if eng.IsearchRegex == nil {
		return
	}

	suggs := make([]Candidate, 0)

	for i := range g.rows {
		row := g.rows[i]

		for _, val := range row {
			if eng.IsearchRegex.MatchString(val.Value) {
				suggs = append(suggs, val)
			} else if val.Description != "" && eng.IsearchRegex.MatchString(val.Description) {
				suggs = append(suggs, val)
			}
		}
	}

	// Reset the group parameters
	g.rows = make([][]Candidate, 0)
	g.posX = -1
	g.posY = -1
	g.columnsWidth = []int{0}

	// Assign the filtered values: we don't need to check
	// for a separate set of non-described values, as the
	// completions have already been triaged when generated.
	vals, _, aliased := eng.groupValues(nil, suggs)
	g.aliased = aliased

	if len(vals) == 0 {
		return
	}

	// And perform the usual initialization routines.
	vals = g.checkDisplays(vals)
	g.computeCells(eng, vals)
	g.makeMatrix(vals)
}

func (g *group) firstCell() {
	g.posX = 0
	g.posY = 0
}

func (g *group) lastCell() {
	g.posY = len(g.rows) - 1
	g.posX = len(g.columnsWidth) - 1

	if g.aliased {
		g.findFirstCandidate(0, -1)
	} else {
		g.posX = len(g.rows[g.posY]) - 1
	}
}

func (g *group) selected() (comp Candidate) {
	defer func() {
		if !g.preserveEscapes {
			comp.Value = color.Strip(comp.Value)
		}
	}()

	if g.posY == -1 || g.posX == -1 {
		return g.rows[0][0]
	}

	return g.rows[g.posY][g.posX]
}

func (g *group) moveSelector(x, y int) (done, next bool) {
	// When the group has not yet been used, adjust
	if g.posX == -1 && g.posY == -1 {
		if x != 0 {
			g.posY++
		} else {
			g.posX++
		}
	}

	g.posX += x
	g.posY += y
	reverse := (x < 0 || y < 0)

	// 1) Ensure columns is minimum one, if not, either
	// go to previous row, or go to previous group.
	if g.posX < 0 {
		if g.posY == 0 && reverse {
			g.posX = 0
			g.posY = 0

			return true, false
		}

		g.posY--
		g.posX = len(g.rows[g.posY]) - 1
	}

	// 2) If we are reverse-cycling and currently on the first candidate,
	// we are done with this group. Stay on those coordinates still.
	if g.posY < 0 {
		if g.posX == 0 {
			g.posX = 0
			g.posY = 0

			return true, false
		}

		g.posY = len(g.rows) - 1
		g.posX--
	}

	// 3) If we are on the last row, we might have to move to
	// the next column, if there is another one.
	if g.posY > g.maxY-1 {
		g.posY = 0
		if g.posX < g.maxX-1 {
			g.posX++
		} else {
			return true, true
		}
	}

	// 4) If we are on the last column, go to next row or next group
	if g.posX > len(g.rows[g.posY])-1 {
		if g.aliased {
			return g.findFirstCandidate(x, y)
		}

		g.posX = 0

		if g.posY < g.maxY-1 {
			g.posY++
		} else {
			return true, true
		}
	}

	// By default, come back to this group for next item.
	return false, false
}

// Check that there is indeed a completion in the column for a given row,
// otherwise loop in the direction wished until one is found, or go next/
// previous column, and so on.
func (g *group) findFirstCandidate(x, y int) (done, next bool) {
	for g.posX > len(g.rows[g.posY])-1 {
		g.posY += y
		g.posY += x

		// Previous column or group
		if g.posY < 0 {
			if g.posX == 0 {
				g.posX = 0
				g.posY = 0

				return true, false
			}

			g.posY = len(g.rows) - 1
			g.posX--
		}

		// Next column or group
		if g.posY > g.maxY-1 {
			g.posY = 0
			if g.posX < len(g.columnsWidth)-1 {
				g.posX++
			} else {
				return true, true
			}
		}
	}

	return
}

func (g *group) writeComps(eng *Engine) (comp string) {
	if len(g.rows) == 0 {
		return
	}

	if g.tag != "" {
		comp += fmt.Sprintf("%s%s%s %s", color.Bold, color.FgYellow, g.tag, color.Reset) + term.ClearLineAfter + term.NewlineReturn
		eng.usedY++
	}

	// Base parameters
	var columns, rows int

	for range g.rows {
		// Generate the completion string for this row (comp/aliases
		// and/or descriptions), and apply any styles and isearch
		// highlighting with pattern replacement,
		comp += g.writeRow(eng, columns)

		columns++
		rows++

		if rows > g.maxY {
			break
		}
	}

	eng.usedY += rows

	return comp
}

func (g *group) writeRow(eng *Engine, row int) (comp string) {
	current := g.rows[row]

	writeDesc := func(val Candidate, x, y, pad int) string {
		desc := g.highlightDescription(eng, val, y, x)
		descPad := g.padDescription(current, val, pad)

		if descPad > 0 {
			desc += strings.Repeat(" ", descPad)
		}

		return desc
	}

	for col, val := range current {
		// Generate the highlighted candidate with its padding
		isSelected := row == g.posY && col == g.posX && g.isCurrent
		cell, pad := g.padCandidate(current, val, col)
		comp += g.highlightCandidate(eng, val, cell, pad, isSelected) + " "

		// And append the description only if at the end of the row,
		// or if there are no aliases, in which case all comps are described.
		if !g.aliased || col == len(current)-1 {
			comp += writeDesc(val, col, row, len(cell)+len(pad))
		}
	}

	comp += term.ClearLineAfter + term.NewlineReturn

	return
}

func (g *group) highlightCandidate(eng *Engine, val Candidate, cell, pad string, selected bool) (candidate string) {
	reset := color.Fmt(val.Style)
	candidate = g.displayTrimmed(val.Display)

	if eng.IsearchRegex != nil && eng.isearchBuf.Len() > 0 {
		match := eng.IsearchRegex.FindString(candidate)
		match = color.Fmt(color.Bg+"244") + match + color.Reset + reset
		candidate = eng.IsearchRegex.ReplaceAllLiteralString(candidate, match)
	}

	switch {
	// If the comp is currently selected, overwrite any highlighting already applied.
	case selected:
		userStyle := color.UnquoteRC(eng.config.GetString("completion-selection-style"))
		selectionHighlightStyle := color.Fmt(color.Bg+"255") + userStyle
		candidate = selectionHighlightStyle + g.displayTrimmed(color.Strip(val.Display))
		if g.aliased {
			candidate += cell + color.Reset
		}

	default:
		// Highlight the prefix if any and configured for it.
		if eng.config.GetBool("colored-completion-prefix") && eng.prefix != "" {
			if prefixMatch, err := regexp.Compile(fmt.Sprintf("^%s", eng.prefix)); err == nil {
				candidate = prefixMatch.ReplaceAllString(candidate, color.Bold+color.FgBlue+eng.prefix+color.BoldReset+color.FgDefault+reset)
			}
		}

		candidate = reset + candidate + color.Reset + cell
	}

	return candidate + pad
}

func (g *group) highlightDescription(eng *Engine, val Candidate, row, col int) (desc string) {
	if val.Description == "" {
		return color.Reset
	}

	desc = g.descriptionTrimmed(val.Description)

	if eng.IsearchRegex != nil && eng.isearchBuf.Len() > 0 {
		match := eng.IsearchRegex.FindString(desc)
		match = color.Fmt(color.Bg+"244") + match + color.Reset + color.Dim
		desc = eng.IsearchRegex.ReplaceAllLiteralString(desc, match)
	}

	// If the comp is currently selected, overwrite any highlighting already applied.
	// Replace all background reset escape sequences in it, to ensure correct display.
	if row == g.posY && col == g.posX && g.isCurrent && !g.aliased {
		userDescStyle := color.UnquoteRC(eng.config.GetString("completion-selection-style"))
		selectionHighlightStyle := color.Fmt(color.Bg+"255") + userDescStyle
		desc = strings.ReplaceAll(desc, color.BgDefault, userDescStyle)
		desc = selectionHighlightStyle + desc
	}

	compDescStyle := color.UnquoteRC(eng.config.GetString("completion-description-style"))

	return compDescStyle + desc + color.Reset
}

func (g *group) padCandidate(row []Candidate, val Candidate, col int) (cell, pad string) {
	var cellLen, padLen int
	valLen := utf8.RuneCountInString(val.Display)

	if !g.aliased {
		padLen = g.tcMaxLength - valLen
		if padLen < 0 {
			padLen = 0
		}

		return "", strings.Repeat(" ", padLen)
	}

	cellLen = g.columnsWidth[col] - valLen

	if len(row) == 1 {
		padLen = sum(g.columnsWidth) + len(g.columnsWidth) - g.columnsWidth[col] - 1
	}

	return strings.Repeat(" ", cellLen), strings.Repeat(" ", padLen)
}

func (g *group) padDescription(row []Candidate, val Candidate, valPad int) (pad int) {
	if g.aliased {
		return 1
	}

	displayNoEscapes := g.displayTrimmed(color.Strip(val.Display))
	descNoEscapes := g.descriptionTrimmed(color.Strip(val.Description))

	candidateLen := len(displayNoEscapes) + valPad + 1
	individualRest := (term.GetWidth() % g.maxCellLength) / (g.maxX + len(row))
	pad = g.maxCellLength - candidateLen - len(descNoEscapes) + individualRest

	if pad > 1 {
		pad--
	}

	return pad
}

func (g *group) displayTrimmed(val string) string {
	termWidth := term.GetWidth()
	if g.tcMaxLength > termWidth-1 {
		g.tcMaxLength = termWidth - 1
	}

	if len(val) > g.tcMaxLength {
		val = val[:g.tcMaxLength-3] + "..."
	}

	val = sanitizer.Replace(val)

	return val
}

func (g *group) descriptionTrimmed(desc string) string {
	if desc == "" {
		return desc
	}

	termWidth := term.GetWidth()
	if g.tcMaxLength > termWidth {
		g.tcMaxLength = termWidth
	}

	g.maxDescWidth = termWidth - g.tcMaxLength - len(g.listSeparator) - 1

	if len(desc) >= g.maxDescWidth {
		offset := 4
		if !g.aliased {
			offset++
		}

		desc = desc[:g.maxDescWidth-offset] + "..."
	}

	desc = g.listSeparator + " " + sanitizer.Replace(desc)

	return desc
}