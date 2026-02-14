// Package tui implements the Poly terminal user interface.
//
// The TUI is decomposed into the following files:
//   - model.go      : Model struct, New(), Init()
//   - messages.go   : Message types (StreamMsg, TableRondeStreamMsg, etc.)
//   - update.go     : Update(), key handling, message routing
//   - views.go      : View(), renderChat(), renderHeader(), renderInput(), renderStatusBar(), renderMessage()
//   - dialogs.go    : Splash, help, control room, add provider dialogs
//   - streaming.go  : sendToProvider(), sendTableRonde(), startNextRound(), stream event reading
//   - commands.go   : handleCommand(), slash commands
//   - clipboard.go  : getClipboardContent(), getClipboardImage()
//   - palette.go    : Command palette overlay (Ctrl+K)
//   - modelpicker.go: Enhanced model picker with filter and grouping
//   - keys.go       : KeyMap and key bindings
package tui
