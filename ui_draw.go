package main

import "github.com/pwiecz/go-fltk"

// padding interno orizzontale delle celle.
const cellPad = 8

// drawCell disegna intestazioni e celle della tabella.
func (w *MainWin) drawCell(ctx fltk.TableContext, r, c, x, y, width, height int) {
	switch ctx {
	case fltk.ContextColHeader:
		w.drawColHeader(c, x, y, width, height)
	case fltk.ContextCell:
		w.drawDataCell(r, c, x, y, width, height)
	}
}

// drawColHeader disegna l'intestazione di una colonna.
func (w *MainWin) drawColHeader(c, x, y, width, height int) {
	fltk.DrawBox(fltk.FLAT_BOX, x, y, width, height, colHeaderBg)
	// bordo inferiore
	fltk.SetDrawColor(colBorder)
	fltk.DrawLine(x, y+height-1, x+width, y+height-1)

	if c == colWhitelist {
		// colonna whitelist
		fltk.SetDrawColor(colMuted)
		fltk.SetDrawFont(fltk.HELVETICA_BOLD, 11)
		fltk.Draw("W", x, y, width, height, fltk.ALIGN_CENTER|fltk.ALIGN_INSIDE)
		return
	}
	if c == colCheck {
		// colonna del checkbox di selezione
		fltk.SetDrawColor(colMuted)
		fltk.SetDrawFont(fltk.HELVETICA_BOLD, 11)
		fltk.Draw("S", x, y, width, height, fltk.ALIGN_CENTER|fltk.ALIGN_INSIDE)
		return
	}

	fltk.SetDrawColor(colText)
	fltk.SetDrawFont(fltk.HELVETICA_BOLD, 12)
	fltk.Draw(columnTitle(c), x+cellPad, y, width-2*cellPad, height, fltk.ALIGN_LEFT|fltk.ALIGN_INSIDE)
}

// drawDataCell disegna una cella di dati.
func (w *MainWin) drawDataCell(r, c, x, y, width, height int) {
	if r < 0 || r >= len(w.pkgs) {
		return
	}

	w.mu.Lock()
	pkg := w.pkgs[r]
	checked := r < len(w.checked) && w.checked[r]
	inWhite := r < len(w.whitelist) && w.whitelist[r]
	w.mu.Unlock()

	// sfondo alternato + evidenziazione selezione.
	bg := colWhite
	if r%2 == 1 {
		bg = colRowAlt
	}
	if checked {
		bg = colSelected
	}
	fltk.DrawBox(fltk.FLAT_BOX, x, y, width, height, bg)

	// separatore inferiore leggero.
	fltk.SetDrawColor(colBorder)
	fltk.DrawLine(x, y+height-1, x+width, y+height-1)

	switch c {
	case colCheck:
		w.drawCheckbox(x, y, width, height, checked, colAccent)
	case colWhitelist:
		w.drawCheckbox(x, y, width, height, inWhite, colMuted)
	case colName:
		w.drawTextCell(pkg.Name, x, y, width, height, fltk.HELVETICA_BOLD, colText)
	case colID:
		w.drawTextCell(pkg.ID, x, y, width, height, fltk.HELVETICA, colMuted)
	case colVersion:
		w.drawTextCell(pkg.Version, x, y, width, height, fltk.HELVETICA, colText)
	case colAvailable:
		w.drawTextCell(pkg.Available, x, y, width, height, fltk.HELVETICA_BOLD, colAccent)
	case colSource:
		w.drawTextCell(pkg.Source, x, y, width, height, fltk.HELVETICA, colMuted)
	}
}

// drawTextCell disegna testo con clipping orizzontale.
func (w *MainWin) drawTextCell(text string, x, y, width, height int, font fltk.Font, col fltk.Color) {
	fltk.SetDrawColor(col)
	fltk.SetDrawFont(font, 12)
	fltk.PushClip(x+1, y, width-2, height)
	fltk.Draw(text, x+cellPad, y, width-2*cellPad, height, fltk.ALIGN_LEFT|fltk.ALIGN_INSIDE)
	fltk.PopClip()
}

// drawCheckbox disegna un checkbox centrato nella cella con il colore dato.
func (w *MainWin) drawCheckbox(x, y, width, height int, checked bool, accent fltk.Color) {
	size := 16
	cx := x + (width-size)/2
	cy := y + (height-size)/2

	if checked {
		// Sfondo pieno nel colore d'accento, poi il segno di spunta
		// bianco disegnato sopra (fl_draw_check disegna solo il simbolo).
		fltk.DrawBox(fltk.THIN_DOWN_BOX, cx, cy, size, size, colWhite)
		fltk.DrawRectfWithColor(cx+1, cy+1, size-2, size-2, accent)
		fltk.DrawCheck(cx+1, cy+1, size-2, size-2, colWhite)
	} else {
		// Riquadro vuoto.
		fltk.DrawBox(fltk.THIN_UP_BOX, cx, cy, size, size, colWhite)
	}
}

// columnTitle restituisce l'intestazione leggibile di una colonna.
func columnTitle(c int) string {
	switch c {
	case colName:
		return "Nome"
	case colID:
		return "ID"
	case colVersion:
		return "Versione"
	case colAvailable:
		return "Disponibile"
	case colSource:
		return "Origine"
	}
	return ""
}
