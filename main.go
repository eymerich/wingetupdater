package main

import (
	"runtime"

	"github.com/pwiecz/go-fltk"
)

// Palette ispirata al Linux Mint Update Manager.
var (
	colWindowBg  = fltk.ColorFromRgb(0xf5, 0xf6, 0xf7)
	colToolbarBg = fltk.ColorFromRgb(0xe8, 0xea, 0xed)
	colHeaderBg  = fltk.ColorFromRgb(0xdc, 0xdf, 0xe3)
	colRowAlt    = fltk.ColorFromRgb(0xf0, 0xf1, 0xf3)
	colWhite     = fltk.WHITE
	colText      = fltk.ColorFromRgb(0x2e, 0x34, 0x36)
	colMuted     = fltk.ColorFromRgb(0x7f, 0x8c, 0x8d)
	colAccent    = fltk.ColorFromRgb(0x2e, 0xa6, 0x4a) // verde Mint
	colSelected  = fltk.ColorFromRgb(0xd6, 0xec, 0xd9)
	colBorder    = fltk.ColorFromRgb(0xc8, 0xcc, 0xd1)
)

// Colonne della tabella.
const (
	colWhitelist = iota
	colCheck
	colName
	colID
	colVersion
	colAvailable
	colSource
	colCount
)

// larghezze in pixel delle colonne.
var colWidths = [colCount]int{36, 36, 264, 220, 110, 110, 90}

// altezza in pixel di ogni riga della tabella.
const rowHeight = 26

func main() {
	// FLTK deve essere gestito sempre dallo stesso thread del sistema
	// operativo: blocchiamo il goroutine principale al thread corrente,
	// altrimenti il runtime Go può spostarlo e il message loop di Windows
	// (con i suoi timer) smette di funzionare, rendendo la finestra bloccata.
	runtime.LockOSThread()

	fltk.InitStyles()
	NewMainWin().Run()
}
