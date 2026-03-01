package display

import (
	"strings"

	"github.com/micro-editor/micro/v2/internal/config"
	"github.com/micro-editor/micro/v2/internal/screen"
	"github.com/micro-editor/tcell/v2"
)

// CompletionPopupVisible suppresses the status bar suggestions row when a popup is shown.
var CompletionPopupVisible bool

const (
	popupMaxHeight = 10 // max visible items (excluding border and filter row)
	popupMinWidth  = 24
)

// PopupMenu is a vertical floating menu drawn over buffer content.
type PopupMenu struct {
	Items    []string // currently visible (possibly filtered) items
	AllItems []string // full unfiltered item list; populated when Filterable
	Filter   string   // current filter string; only used when Filterable

	Selected int // currently highlighted row index into Items
	Offset   int // first visible item index (for scrolling)
	Title    string

	Centered   bool // if true, center on screen; otherwise anchor to cursor
	Filterable bool // if true, capture rune/backspace input to filter Items
	AnchorX    int  // screen X of cursor (anchored mode)
	AnchorY    int  // screen Y of cursor (anchored mode)

	Visible bool

	// Callbacks (may be nil)
	OnSelect  func(idx int)
	OnAccept  func(idx int, item string)
	OnDismiss func()
}

// applyFilter rebuilds Items from AllItems using the current Filter string.
func (p *PopupMenu) applyFilter() {
	if !p.Filterable {
		return
	}
	filter := strings.ToLower(p.Filter)
	filtered := make([]string, 0, len(p.AllItems))
	for _, item := range p.AllItems {
		if filter == "" || strings.Contains(strings.ToLower(item), filter) {
			filtered = append(filtered, item)
		}
	}
	p.Items = filtered
	p.Selected = 0
	p.Offset = 0
}

// layout computes the popup rectangle (x, y, w, h).
func (p *PopupMenu) layout() (x, y, w, h int) {
	sw, sh := screen.Screen.Size()

	// Use AllItems for width when filterable so popup doesn't shrink as you type.
	source := p.Items
	if p.Filterable && len(p.AllItems) > 0 {
		source = p.AllItems
	}

	w = popupMinWidth
	for _, item := range source {
		needed := len([]rune(item)) + 4 // │ space + item + marker + │
		if needed > w {
			w = needed
		}
	}
	if len([]rune(p.Title))+4 > w {
		w = len([]rune(p.Title)) + 4
	}
	if w > sw {
		w = sw
	}

	// visible item rows: cap at popupMaxHeight, min 1 when filterable
	visible := len(p.Items)
	if visible > popupMaxHeight {
		visible = popupMaxHeight
	}
	if p.Filterable && visible < 1 {
		visible = 1
	}

	// h = top border + [filter row + separator] + item rows + bottom border
	h = visible + 2
	if p.Filterable {
		h += 2 // filter input row + separator line
	}

	if p.Centered {
		x = (sw - w) / 2
		y = (sh - h) / 2
	} else {
		x = p.AnchorX
		y = p.AnchorY + 1
		if y+h > sh {
			y = p.AnchorY - h
		}
		if x+w > sw {
			x = sw - w
		}
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
	}
	return
}

// ensureVisible adjusts Offset so that Selected is in the visible window.
func (p *PopupMenu) ensureVisible() {
	visible := popupMaxHeight
	if len(p.Items) < visible {
		visible = len(p.Items)
	}
	if p.Selected < p.Offset {
		p.Offset = p.Selected
	}
	if p.Selected >= p.Offset+visible {
		p.Offset = p.Selected - visible + 1
	}
}

// SetSelected updates the selected item and adjusts the scroll offset.
func (p *PopupMenu) SetSelected(idx int) {
	p.Selected = idx
	p.ensureVisible()
}

// CycleDown moves selection down by one item.
func (p *PopupMenu) CycleDown() {
	if len(p.Items) == 0 {
		return
	}
	p.Selected = (p.Selected + 1) % len(p.Items)
	p.ensureVisible()
	if p.OnSelect != nil {
		p.OnSelect(p.Selected)
	}
}

// CycleUp moves selection up by one item.
func (p *PopupMenu) CycleUp() {
	if len(p.Items) == 0 {
		return
	}
	p.Selected = (p.Selected - 1 + len(p.Items)) % len(p.Items)
	p.ensureVisible()
	if p.OnSelect != nil {
		p.OnSelect(p.Selected)
	}
}

// Display draws the popup menu to the screen.
func (p *PopupMenu) Display() {
	if !p.Visible {
		return
	}
	if len(p.Items) == 0 && !p.Filterable {
		return
	}

	x, y, w, h := p.layout()
	visible := h - 2
	if p.Filterable {
		visible -= 2 // filter input row + separator line
	}

	// Styles
	popupStyle := config.DefStyle.Reverse(true)
	if style, ok := config.Colorscheme["completion.popup"]; ok {
		popupStyle = style
	} else if style, ok := config.Colorscheme["statusline"]; ok {
		popupStyle = style.Reverse(true)
	}

	// Items use the editor's default style (dark bg, light text) so the popup
	// blends with the editor rather than floating on a harsh white background.
	itemStyle := config.DefStyle
	if style, ok := config.Colorscheme["completion.item"]; ok {
		itemStyle = style
	}

	// Selected item uses the statusline colors (un-reversed: typically light bg,
	// dark text) to create strong contrast against the dark item background.
	selectedStyle := config.DefStyle.Reverse(true)
	if style, ok := config.Colorscheme["completion.selected"]; ok {
		selectedStyle = style
	} else if style, ok := config.Colorscheme["statusline"]; ok {
		selectedStyle = style
	}

	set := func(px, py int, r rune, style tcell.Style) {
		screen.SetContent(px, py, r, nil, style)
	}

	innerW := w - 2

	// Top border: ┌─ Title ─...─┐
	set(x, y, '┌', popupStyle)
	titleRunes := []rune(p.Title)
	if len(titleRunes) > 0 && len(titleRunes)+2 <= innerW {
		set(x+1, y, '─', popupStyle)
		for i, r := range titleRunes {
			set(x+2+i, y, r, popupStyle)
		}
		for i := 1 + len(titleRunes); i < innerW; i++ {
			set(x+1+i, y, '─', popupStyle)
		}
	} else {
		for i := 0; i < innerW; i++ {
			set(x+1+i, y, '─', popupStyle)
		}
	}
	set(x+w-1, y, '┐', popupStyle)

	// Filter input row + separator (only when Filterable)
	itemRowStart := y + 1
	if p.Filterable {
		py := y + 1
		itemRowStart = y + 3 // past filter row and separator
		set(x, py, '│', popupStyle)
		// Draw "> filter_text"
		set(x+1, py, '>', itemStyle)
		set(x+2, py, ' ', itemStyle)
		col := 3
		filterRunes := []rune(p.Filter)
		for _, r := range filterRunes {
			if col >= innerW {
				break
			}
			set(x+col, py, r, itemStyle)
			col++
		}
		for col < innerW {
			set(x+col, py, ' ', itemStyle)
			col++
		}
		set(x+w-1, py, '│', popupStyle)
		// Position terminal cursor at end of filter text
		cursorCol := 3 + len(filterRunes)
		if cursorCol > innerW-1 {
			cursorCol = innerW - 1
		}
		screen.ShowCursor(x+cursorCol, py)

		// Separator between filter input and item list
		sepY := y + 2
		set(x, sepY, '├', popupStyle)
		for i := 1; i < w-1; i++ {
			set(x+i, sepY, '─', popupStyle)
		}
		set(x+w-1, sepY, '┤', popupStyle)
	}

	// Item rows
	hasAbove := p.Offset > 0
	hasBelow := p.Offset+visible < len(p.Items)

	for row := 0; row < visible; row++ {
		py := itemRowStart + row
		idx := p.Offset + row
		isSelected := idx == p.Selected && idx < len(p.Items)

		style := itemStyle
		if isSelected {
			style = selectedStyle
		}

		set(x, py, '│', popupStyle)

		textW := innerW - 1
		var itemRunes []rune
		if idx < len(p.Items) {
			itemRunes = []rune(p.Items[idx])
		}
		for i := 0; i < textW; i++ {
			if i < len(itemRunes) {
				set(x+1+i, py, itemRunes[i], style)
			} else {
				set(x+1+i, py, ' ', style)
			}
		}

		lastCol := x + 1 + textW
		if row == 0 && hasAbove {
			set(lastCol, py, '▲', popupStyle)
		} else if row == visible-1 && hasBelow {
			set(lastCol, py, '▼', popupStyle)
		} else {
			set(lastCol, py, ' ', style)
		}

		set(x+w-1, py, '│', popupStyle)
	}

	// Bottom border: └─...─┘
	set(x, y+h-1, '└', popupStyle)
	for i := 1; i < w-1; i++ {
		set(x+i, y+h-1, '─', popupStyle)
	}
	set(x+w-1, y+h-1, '┘', popupStyle)
}

// HandleEvent processes a tcell event for the popup.
// Returns true if the event was consumed by the popup.
func (p *PopupMenu) HandleEvent(e tcell.Event) bool {
	ke, ok := e.(*tcell.EventKey)
	if !ok {
		return false
	}

	switch ke.Key() {
	case tcell.KeyUp:
		p.CycleUp()
		return true
	case tcell.KeyDown:
		p.CycleDown()
		return true
	case tcell.KeyEnter:
		if p.OnAccept != nil && p.Selected < len(p.Items) {
			p.OnAccept(p.Selected, p.Items[p.Selected])
		}
		p.Visible = false
		return true
	case tcell.KeyEscape:
		if p.OnDismiss != nil {
			p.OnDismiss()
		}
		p.Visible = false
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if p.Filterable {
			runes := []rune(p.Filter)
			if len(runes) > 0 {
				p.Filter = string(runes[:len(runes)-1])
				p.applyFilter()
			}
			return true
		}
	case tcell.KeyRune:
		if p.Filterable {
			p.Filter += string(ke.Rune())
			p.applyFilter()
			return true
		}
	}
	return false
}
