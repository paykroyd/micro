package action

import (
	"sort"

	"github.com/micro-editor/micro/v2/internal/display"
)

// ActivePopup is the currently shown popup, or nil if none.
var ActivePopup *display.PopupMenu

// ShowCompletionPopup opens an anchored completion popup below the cursor of h.
func ShowCompletionPopup(h *BufPane) {
	b := h.Buf
	ax, ay := h.CursorScreenPos()

	prevIdx := b.CurSuggestion

	p := &display.PopupMenu{
		Items:    b.Suggestions,
		Selected: b.CurSuggestion,
		AnchorX:  ax,
		AnchorY:  ay,
		Visible:  true,
	}
	p.OnSelect = func(idx int) {
		n := len(b.Suggestions)
		if n == 0 {
			return
		}
		if idx == (prevIdx+1)%n {
			b.CycleAutocomplete(true)
		} else {
			b.CycleAutocomplete(false)
		}
		prevIdx = b.CurSuggestion
		p.Selected = b.CurSuggestion
	}
	p.OnAccept = func(idx int, item string) {
		b.HasSuggestions = false
		DismissPopup()
	}
	p.OnDismiss = func() {
		b.HasSuggestions = false
		DismissPopup()
	}
	ActivePopup = p
	display.CompletionPopupVisible = true
}

// ShowPickerPopup opens a centered filterable picker popup.
func ShowPickerPopup(title string, items []string, onAccept func(int, string)) {
	p := &display.PopupMenu{
		Title:      title,
		AllItems:   items,
		Items:      append([]string(nil), items...),
		Centered:   true,
		Filterable: true,
		Visible:    true,
	}
	p.OnAccept = func(idx int, item string) {
		onAccept(idx, item)
		DismissPopup()
	}
	p.OnDismiss = func() {
		DismissPopup()
	}
	ActivePopup = p
}

// DismissPopup hides and removes the active popup.
func DismissPopup() {
	ActivePopup = nil
	display.CompletionPopupVisible = false
}

// SyncPopupWithBuffer dismisses the completion popup if the buffer no longer has suggestions.
func SyncPopupWithBuffer() {
	if ActivePopup == nil {
		return
	}
	if !display.CompletionPopupVisible {
		return // picker popup, don't auto-dismiss
	}
	bp := MainTab().CurPane()
	if bp != nil && !bp.Buf.HasSuggestions {
		DismissPopup()
	}
}

// CommandPalette opens a centered popup listing all available commands.
// Selecting a command that requires arguments opens an InfoBar prompt
// pre-seeded with the command name so the user can type the arguments.
func (h *BufPane) CommandPalette() bool {
	var items []string
	for name := range commands {
		items = append(items, name)
	}
	sort.Strings(items)
	ShowPickerPopup("Command Palette", items, func(_ int, cmd string) {
		if c, ok := commands[cmd]; ok && c.requiresArgs {
			InfoBar.Prompt("> ", cmd+" ", "Command", nil, func(resp string, canceled bool) {
				if !canceled {
					h.HandleCommand(resp)
				}
			})
		} else {
			h.HandleCommand(cmd)
		}
	})
	return true
}

func init() {
	BufKeyActions["CommandPalette"] = (*BufPane).CommandPalette
}
