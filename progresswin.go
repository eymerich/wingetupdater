package main

import "github.com/pwiecz/go-fltk"

// Dimensioni del popup di avanzamento.
const (
	progressWinW = 360
	progressWinH = 120
)

// Parametri dell'animazione indeterminata (barra che va avanti e indietro).
const (
	animSteps    = 40
	animInterval = 0.03
)

// ProgressWin è un popup modale con una barra di avanzamento, usato sia per
// la ricerca (avanzamento indeterminato) sia per l'installazione
// (avanzamento determinato 0..N).
type ProgressWin struct {
	win      *fltk.Window
	message  *fltk.Box
	progress *fltk.Progress

	// stato dell'animazione indeterminata
	step int
	dir  int
	anim bool
}

// NewProgressWin costruisce il popup centrato attorno alla posizione data.
func NewProgressWin(title string, centerX, centerY int) *ProgressWin {
	p := &ProgressWin{dir: 1}

	x := centerX - progressWinW/2
	y := centerY - progressWinH/2

	p.win = fltk.NewWindowWithPosition(x, y, progressWinW, progressWinH)
	p.win.SetLabel(title)
	p.win.SetColor(colWindowBg)
	p.win.SetModal()

	p.message = fltk.NewBox(fltk.NO_BOX, 16, 14, progressWinW-32, 24, "")
	p.message.SetLabelColor(colText)
	p.message.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE)
	p.message.SetLabelSize(13)

	p.progress = fltk.NewProgress(16, 50, progressWinW-32, 22)
	p.progress.SetMinimum(0)
	p.progress.SetMaximum(1)
	p.progress.SetValue(0)
	p.progress.SetSelectionColor(colAccent)

	p.win.End()
	return p
}

// showInfinite mostra il popup con avanzamento indeterminato (ping-pong).
func (p *ProgressWin) showInfinite(initialMessage string) {
	p.anim = true
	p.step = 0
	p.dir = 1
	p.message.SetLabel(initialMessage)
	p.progress.SetMinimum(0)
	p.progress.SetMaximum(animSteps)
	p.progress.SetValue(0)
	p.win.Show()
	p.tick()
}

// showDeterminate mostra il popup con una vera barra di avanzamento 0..total.
func (p *ProgressWin) showDeterminate(initialMessage string, total int) {
	p.anim = false
	p.message.SetLabel(initialMessage)
	p.progress.SetMinimum(0)
	p.progress.SetMaximum(float64(total))
	p.progress.SetValue(0)
	p.win.Show()
	p.message.Redraw()
	p.progress.Redraw()
}

// setProgress aggiorna posizione e testo della barra determinata.
func (p *ProgressWin) setProgress(done, total int, message string) {
	p.progress.SetMaximum(float64(total))
	p.progress.SetValue(float64(done))
	p.message.SetLabel(message)
	p.message.Redraw()
	p.progress.Redraw()
}

// Close ferma l'animazione e nasconde il popup.
func (p *ProgressWin) Close() {
	p.anim = false
	p.win.Hide()
}

// tick avanza l'animazione indeterminata di un passo e la riprogramma.
func (p *ProgressWin) tick() {
	if !p.anim {
		return
	}

	p.step += p.dir
	if p.step >= animSteps {
		p.step = animSteps
		p.dir = -1
	} else if p.step <= 0 {
		p.step = 0
		p.dir = 1
	}
	p.progress.SetValue(float64(p.step))
	p.progress.Redraw()

	fltk.RepeatTimeout(animInterval, p.tick)
}
