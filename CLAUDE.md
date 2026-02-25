# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Micro is a terminal-based text editor written in Go. It produces a static binary with no runtime dependencies. The project targets Go 1.19+.

## Build Commands

```sh
make build          # Full build: generate runtime assets, then compile
make build-quick    # Compile only (skip asset generation, faster iteration)
make build-dbg      # Build with debug symbols (enables logging to ./log.txt)
make install        # Install to $GOPATH/bin

make generate       # Regenerate embedded runtime assets (syntax, colorschemes)
```

The binary is built with `CGO_ENABLED=0` by default (static linking). macOS forces `CGO_ENABLED=1` for Info.plist support.

## Test Commands

```sh
make test                         # Run all tests
go test ./internal/buffer/...     # Run a specific package
go test -run TestName ./internal/buffer/  # Run a single test
make bench                        # Run benchmarks (3 iterations)
make bench-compare                # Compare against saved baseline
```

Buffer operation tests are partially generated from VSCode test specs via `make testgen` (requires TypeScript compiler).

## Debug / Profiling

```sh
./micro -debug      # Logs to ./log.txt at runtime
./micro -profile    # CPU profiling to ./micro.prof (use go tool pprof)
```

## Architecture

The codebase is organized under `internal/` with strict separation of concerns:

### Core Packages

- **`internal/buffer`** — Text model: `Buffer`, `LineArray`, `Cursor`, undo/redo stack, diff gutter, multi-cursor, file encoding, backup/recovery. `Loc` (Row, Col) is the fundamental position type.

- **`internal/action`** — User input and command execution. `BufPane` is the main editor pane; it handles keyboard/mouse events, dispatches actions (insert, delete, navigate, select), manages keybindings, and runs macros. Platform-specific action files exist for posix/other/darwin.

- **`internal/display`** — Terminal rendering. `BufWindow` renders a buffer section; `StatusLine` is the bottom bar; `TabWindow` is the tab bar. Handles soft-wrap, line numbers, gutter, scrollbar.

- **`internal/views`** — Split management as a binary tree of `Node`s. Each node is either a vertical (`STVert`) or horizontal (`STHoriz`) split. Handles proportional scaling on resize.

- **`internal/config`** — Configuration loading (XDG paths, `~/.config/micro/`), color scheme management, plugin loading and package manager, settings validation.

- **`internal/lua`** — Lua 5.1 VM (`gopher-lua`) for plugins. Exposes Go APIs to Lua via `gopher-luar`. Plugins live in `runtime/plugins/` (built-in) or `~/.config/micro/plugins/` (user).

- **`internal/screen`** — tcell wrapper. Thread-safe screen locking, event polling goroutine, raw escape handling.

- **`internal/shell`** — Shell command execution, job management, foreground/background, signal handling.

- **`internal/clipboard`** — Clipboard abstraction supporting internal, primary, and system clipboards via the `clipper` library.

- **`pkg/highlight`** — Regex-based syntax highlighting engine. Parses `.yaml` syntax definition files.

### Embedded Assets (`runtime/`)

Runtime assets (syntax definitions, colorschemes, help files, default plugins) are embedded via `go:generate` in `runtime/`. Running `make generate` regenerates the Go source files that embed these assets. The 130+ syntax definitions are in `runtime/syntax/`.

### Buffer Types

Buffers have a type (`BTDefault`, `BTHelp`, `BTLog`, `BTScratch`, `BTInfo`, `BTStdout`) that controls behavior like read-only access, save eligibility, and display.

### Input Flow

```
Terminal events (tcell)
  → screen.Screen
  → action.BufPane (keybinding dispatch)
  → buffer.Buffer (text mutations)
  → display.BufWindow (re-render)
  → screen.Screen (flush)
```

Lua plugin callbacks can intercept at the action layer.

## Key Dependencies

- `github.com/micro-editor/tcell/v2` — forked terminal library (not the upstream zyedidia/tcell)
- `github.com/yuin/gopher-lua` + `layeh.com/gopher-luar` — Lua VM and Go interop
- `github.com/sergi/go-diff` — diff engine for the gutter
- `golang.org/x/text` — Unicode and encoding support
