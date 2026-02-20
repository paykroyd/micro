VERSION = "1.0.0"

local micro = import("micro")
local config = import("micro/config")

-- State
local inFindMode = false
local findRunes = {}  -- one entry per rune, so backspace is clean
local findQuery = ""  -- assembled query string
local savedX = 0
local savedY = 0

local function rebuildQuery()
    findQuery = table.concat(findRunes)
end

local function showStatus()
    micro.InfoBar():Message("find: " .. findQuery)
end

-- Move the cursor back to where the user was when find mode was entered,
-- then run Search() so we always land on the *first* match after that point.
-- After Search(), force HighlightSearch on regardless of the hlsearch setting.
local function doSearch(bp)
    if findQuery == "" then
        bp.Buf.HighlightSearch = false
        bp.Buf.LastSearch = ""
        bp.Cursor.ResetSelection()
        bp.Cursor.X = savedX
        bp.Cursor.Y = savedY
        bp:Relocate()
        return
    end
    bp.Cursor.X = savedX
    bp.Cursor.Y = savedY
    bp.Cursor.ResetSelection()
    bp:Search(findQuery, false, true)
    bp.Buf.HighlightSearch = true
end

-- Ctrl-f: enter find mode (or toggle it off if already active).
function startFind(bp, args)
    if inFindMode then
        inFindMode = false
        micro.InfoBar():Message("")
        return
    end
    inFindMode = true
    findRunes = {}
    findQuery = ""
    savedX = bp.Cursor.X
    savedY = bp.Cursor.Y
    showStatus()
end

-- Every printable character typed goes into the query, not the buffer.
function preRune(bp, r)
    if not inFindMode then return true end
    table.insert(findRunes, r)
    rebuildQuery()
    doSearch(bp)
    showStatus()
    return false
end

-- Backspace removes the last rune from the query.
function preBackspace(bp)
    if not inFindMode then return true end
    if #findRunes > 0 then
        table.remove(findRunes)
        rebuildQuery()
    end
    doSearch(bp)
    showStatus()
    return false
end

-- Down arrow: jump to the next match.
function preCursorDown(bp)
    if not inFindMode then return true end
    if findQuery ~= "" then
        bp:FindNext()
        bp.Buf.HighlightSearch = true
        showStatus()
    end
    return false
end

-- Up arrow: jump to the previous match.
function preCursorUp(bp)
    if not inFindMode then return true end
    if findQuery ~= "" then
        bp:FindPrevious()
        bp.Buf.HighlightSearch = true
        showStatus()
    end
    return false
end

-- Escape: cancel find mode and restore the original cursor position.
function preEscape(bp)
    if not inFindMode then return true end
    inFindMode = false
    findRunes = {}
    findQuery = ""
    bp.Buf.HighlightSearch = false
    bp.Buf.LastSearch = ""
    bp.Cursor.ResetSelection()
    bp.Cursor.X = savedX
    bp.Cursor.Y = savedY
    bp:Relocate()
    micro.InfoBar():Message("")
    return false
end

-- Enter: confirm the current match and leave find mode.
-- Highlights stay on so FindNext/FindPrevious keep working afterwards.
function preInsertNewline(bp)
    if not inFindMode then return true end
    inFindMode = false
    micro.InfoBar():Message("")
    return false
end

function init()
    config.MakeCommand("findmode", startFind, config.NoComplete)
    config.TryBindKey("Ctrl-f", "command:findmode", false)
end
