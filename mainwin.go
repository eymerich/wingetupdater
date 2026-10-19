package main

import (
	"fmt"
	"sync"

	"github.com/pwiecz/go-fltk"
)

// event è un messaggio inviato dai goroutine di lavoro verso la UI.
// Viene consumato dal poller che gira sul thread di FLTK.
type event interface{ isEvent() }

// evStatus aggiorna la barra di stato.
type evStatus struct{ msg string }

// evPackages consegna l'elenco dei pacchetti aggiornabili (o un errore).
type evPackages struct {
	pkgs []Package
	err  error
}

// evInstall aggiorna l'avanzamento dell'installazione.
type evInstall struct {
	done  int
	total int
	name  string
}

// evInstallDone segnala la fine dell'installazione e fa ricaricare la lista.
type evInstallDone struct {
	err error
}

func (evStatus) isEvent()      {}
func (evPackages) isEvent()    {}
func (evInstall) isEvent()     {}
func (evInstallDone) isEvent() {}

// pollInterval è l'intervallo (in secondi) con cui la UI legge il canale eventi.
const pollInterval = 0.1

// MainWin rappresenta la finestra principale del gestore aggiornamenti.
type MainWin struct {
	win   *fltk.Window
	table *fltk.TableRow

	pkgs    []Package
	checked []bool

	// whitelist tiene traccia, per ogni riga, se l'ID è incluso nella
	// whitelist usata da "Seleziona tutto".
	whitelist []bool

	status     *fltk.Box
	header     *fltk.Box
	selectAll  *fltk.CheckButton
	updateBtn  *fltk.Button
	refreshBtn *fltk.Button

	search *ProgressWin // popup modale mostrato durante ricerca/installazione

	// events è il canale su cui i goroutine di lavoro notificano la UI.
	events chan event

	mu sync.Mutex
}

// NewMainWin costruisce e mostra la finestra principale.
func NewMainWin() *MainWin {
	w := &MainWin{events: make(chan event, 16)}
	w.build()
	return w
}

// Run mostra la finestra, avvia il poller degli eventi e il caricamento
// dei pacchetti, quindi entra nel loop eventi di fltk.
func (w *MainWin) Run() {
	w.win.Show()

	// Il poller gira sul thread di FLTK e applica gli aggiornamenti alla UI.
	w.startPoller()

	// Popup modale con barra di avanzamento indeterminata durante la ricerca.
	w.showSearchProgress()

	// Caricamento pacchetti in background per non bloccare la UI.
	go w.loadPackages()

	fltk.Run()
}

// startPoller avvia un timeout ricorrente che drena il canale eventi.
// Ogni callback viene eseguita sul thread di FLTK, quindi è sicuro
// modificare i widget direttamente.
func (w *MainWin) startPoller() {
	var tick func()
	tick = func() {
		w.drainEvents()
		// Riprogramma il prossimo tick come timeout ripetuto.
		fltk.RepeatTimeout(pollInterval, tick)
	}
	fltk.AddTimeout(pollInterval, tick)
}

// drainEvents consuma tutti gli eventi disponibili e aggiorna la UI.
func (w *MainWin) drainEvents() {
	for {
		select {
		case ev := <-w.events:
			w.handleEvent(ev)
		default:
			return
		}
	}
}

// handleEvent applica un singolo evento alla UI (sul thread di FLTK).
func (w *MainWin) handleEvent(ev event) {
	switch e := ev.(type) {
	case evStatus:
		w.status.SetLabel(e.msg)
		w.status.Redraw()
	case evPackages:
		w.applyPackages(e.pkgs, e.err)
	case evInstall:
		w.applyInstallProgress(e.done, e.total, e.name)
	case evInstallDone:
		w.finishInstall(e.err)
	}
}

// Dimensioni fisse della finestra (non ridimensionabile).
const (
	winWidth  = 960
	winHeight = 620
)

// build definisce la struttura della finestra e tutti i widget.
func (w *MainWin) build() {
	// --- Finestra principale ---
	w.win = fltk.NewWindow(winWidth, winHeight)
	w.win.SetLabel("Gestore aggiornamenti")
	w.win.SetColor(colWindowBg)

	w.buildToolbar()
	w.buildHeader()
	w.table = w.buildTable()
	w.buildFooter()

	w.win.End()

	// Finestra non ridimensionabile: min == max = dimensione fissa.
	w.win.SetSizeRange(winWidth, winHeight, winWidth, winHeight, 0, 0, false)
}

// buildToolbar crea la barra superiore stile Mint con i pulsanti azione.
func (w *MainWin) buildToolbar() {
	tb := fltk.NewBox(fltk.FLAT_BOX, 0, 0, winWidth, 54)
	tb.SetColor(colToolbarBg)

	title := fltk.NewBox(fltk.NO_BOX, 16, 8, 460, 38, "Aggiornamenti disponibili")
	title.SetLabelFont(fltk.HELVETICA_BOLD)
	title.SetLabelSize(17)
	title.SetLabelColor(colText)
	title.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE)

	// "Seleziona tutto" seleziona/deseleziona tutte le righe.
	w.selectAll = fltk.NewCheckButton(486, 15, 130, 24, "Seleziona tutto")
	w.selectAll.SetLabelColor(colText)
	w.selectAll.SetValue(true)
	w.selectAll.SetCallback(w.onSelectAll)

	// Pulsante per ricaricare l'elenco degli aggiornamenti disponibili.
	w.refreshBtn = fltk.NewButton(624, 13, 124, 28, "Aggiorna lista")
	w.refreshBtn.SetCallback(w.reload)

	// Pulsante principale di aggiornamento.
	w.updateBtn = fltk.NewButton(756, 13, 188, 28, "Installa aggiornamenti")
	w.updateBtn.SetColor(colAccent)
	w.updateBtn.SetLabelColor(colWhite)
	w.updateBtn.SetLabelFont(fltk.HELVETICA_BOLD)
	w.updateBtn.SetCallback(func() {
		w.installSelected(w.selectedPackages())
	})
}

// buildHeader crea una fascia descrittiva sopra la tabella.
func (w *MainWin) buildHeader() {
	w.header = fltk.NewBox(fltk.FLAT_BOX, 0, 54, winWidth, 40)
	w.header.SetColor(colWhite)
	w.header.SetLabel("Ricerca degli aggiornamenti in corso…")
	w.header.SetLabelColor(colMuted)
	w.header.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE)
	w.header.SetLabelSize(13)
}

// buildTable costruisce la TableRow con le colonne richieste.
func (w *MainWin) buildTable() *fltk.TableRow {
	t := fltk.NewTableRow(0, 94, winWidth, 470)

	t.SetRowCount(0)
	t.SetColumnCount(colCount)
	t.EnableColumnHeaders()
	t.SetColumnHeaderHeight(28)
	t.SetRowHeightAll(rowHeight)
	t.SetColumnWidthAll(100)
	for i, cw := range colWidths {
		t.SetColumnWidth(i, cw)
	}
	t.SetType(fltk.SelectMulti)
	t.SetColor(colWhite)
	t.SetSelectionColor(colSelected)

	// Disegno personalizzato delle celle.
	t.SetDrawCellCallback(func(ctx fltk.TableContext, r, c, x, y, cw, ch int) {
		w.drawCell(ctx, r, c, x, y, cw, ch)
	})

	// Gestione click sul checkbox: intercettiamo PUSH/RELEASE e usiamo la
	// posizione del cursore per individuare la cella, più affidabile di
	// callback_row/callback_col.
	t.SetEventHandler(func(ev fltk.Event) bool {
		if ev != fltk.RELEASE {
			return false
		}
		r, c := t.RowAndColumnFromCursor()
		if r < 0 {
			return false
		}
		switch c {
		case colCheck:
			// Consuma l'evento per evitare che Fl_Table gestisca da sé
			// la selezione della riga quando si clicca sul checkbox.
			w.toggleRow(r)
			return true
		case colWhitelist:
			// Consuma l'evento per evitare che Fl_Table gestisca da sé
			// la selezione della riga quando si clicca sul checkbox.
			w.toggleWhitelist(r)
			return true
		}
		return false
	})

	return t
}

// buildFooter crea la barra di stato inferiore con conteggio e pulsanti.
func (w *MainWin) buildFooter() {
	f := fltk.NewBox(fltk.FLAT_BOX, 0, 564, winWidth, 56)
	f.SetColor(colToolbarBg)

	w.status = fltk.NewBox(fltk.NO_BOX, 16, 564, 700, 56, "")
	w.status.SetLabelColor(colText)
	w.status.SetLabelSize(13)
	w.status.SetAlign(fltk.ALIGN_LEFT | fltk.ALIGN_INSIDE)

	refresh := fltk.NewButton(640, 578, 100, 28, "Ricarica")
	refresh.SetCallback(w.reload)

	quit := fltk.NewButton(760, 578, 80, 28, "Chiudi")
	quit.SetCallback(func() {
		w.win.Hide()
	})
}

// onSelectAll applica lo stato del checkbox "Seleziona tutto".
// In selezione considera solo le righe in whitelist; in deselezione
// agisce su tutte le righe a prescindere dalla whitelist.
func (w *MainWin) onSelectAll() {
	w.mu.Lock()
	v := w.selectAll.Value()
	for i := range w.checked {
		if !v || w.inWhitelistLocked(i) {
			w.checked[i] = v
		}
	}
	w.mu.Unlock()
	w.refreshCounts()
	w.table.Redraw()
}

// inWhitelistLocked indica se la riga i rientra nella whitelist.
// L'assenza di whitelist significa "tutte le righe".
func (w *MainWin) inWhitelistLocked(i int) bool {
	if i < 0 || i >= len(w.whitelist) {
		return false
	}
	hasAny := false
	for _, b := range w.whitelist {
		if b {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return true
	}
	return w.whitelist[i]
}

// toggleWhitelist inverte l'appartenenza alla whitelist di una riga.
func (w *MainWin) toggleWhitelist(r int) {
	w.mu.Lock()
	if r < 0 || r >= len(w.whitelist) {
		w.mu.Unlock()
		return
	}
	w.whitelist[r] = !w.whitelist[r]
	w.mu.Unlock()

	w.persistWhitelist()
	w.refreshCounts()
	w.table.Redraw()
}

// persistWhitelist salva la whitelist corrente (ID selezionati) su settings.json.
func (w *MainWin) persistWhitelist() {
	w.mu.Lock()
	ids := make([]string, 0, len(w.pkgs))
	for i, p := range w.pkgs {
		if i < len(w.whitelist) && w.whitelist[i] {
			ids = append(ids, p.ID)
		}
	}
	w.mu.Unlock()

	if err := saveSettings(&Settings{Whitelist: ids}); err != nil {
		w.status.SetLabel(fmt.Sprintf("Impossibile salvare la whitelist: %v", err))
		w.status.Redraw()
	}
}

// toggleRow inverte lo stato di selezione di una riga.
func (w *MainWin) toggleRow(r int) {
	w.mu.Lock()
	if r < 0 || r >= len(w.checked) {
		w.mu.Unlock()
		return
	}
	w.checked[r] = !w.checked[r]
	// Il master "Seleziona tutto" riflette solo le righe in whitelist.
	all := true
	any := false
	for i, c := range w.checked {
		if !w.inWhitelistLocked(i) {
			continue
		}
		any = true
		if !c {
			all = false
			break
		}
	}
	w.selectAll.SetValue(any && all)
	w.mu.Unlock()

	w.refreshCounts()
	w.table.Redraw()
}

// reload avvia il ricaricamento della lista degli aggiornamenti.
func (w *MainWin) reload() {
	w.showSearchProgress()
	go w.loadPackages()
}

// ensureProgress crea (alla prima chiamata) il popup di avanzamento centrato.
func (w *MainWin) ensureProgress() {
	if w.search == nil {
		w.search = NewProgressWin("", w.win.X()+winWidth/2, w.win.Y()+winHeight/2)
	}
}

// showSearchProgress apre il popup con barra indeterminata per la ricerca.
func (w *MainWin) showSearchProgress() {
	w.ensureProgress()
	w.search.win.SetLabel("Ricerca aggiornamenti")
	w.search.showInfinite("Ricerca degli aggiornamenti in corso…")
}

// showInstallProgress apre il popup con barra determinata per l'installazione.
func (w *MainWin) showInstallProgress(total int) {
	w.ensureProgress()
	w.search.win.SetLabel("Installazione aggiornamenti")
	w.search.showDeterminate(fmt.Sprintf("Aggiornamento di %d pacchetti…", total), total)
}

// hideProgress chiude il popup di avanzamento, se aperto.
func (w *MainWin) hideProgress() {
	if w.search != nil {
		w.search.Close()
	}
}

// loadPackages esegue winget in un goroutine e notifica la UI via canale.
func (w *MainWin) loadPackages() {
	w.post(evStatus{"Ricerca degli aggiornamenti in corso…"})
	pkgs, err := listUpgradable()
	w.post(evPackages{pkgs: pkgs, err: err})
}

// applyPackages popola la tabella con i pacchetti ricevuti (thread di FLTK).
func (w *MainWin) applyPackages(pkgs []Package, err error) {
	// La ricerca è conclusa: chiudi il popup modale.
	w.hideProgress()

	if err != nil {
		w.status.SetLabel(fmt.Sprintf("Errore: %v", err))
		w.status.Redraw()
		w.header.SetLabel("Impossibile recuperare gli aggiornamenti.")
		w.header.Redraw()
		return
	}

	w.mu.Lock()
	w.pkgs = pkgs
	w.checked = make([]bool, len(pkgs))
	w.whitelist = make([]bool, len(pkgs))
	// Applica la whitelist salvata (match per ID).
	if s, err := loadSettings(); err == nil && len(s.Whitelist) > 0 {
		wanted := make(map[string]bool, len(s.Whitelist))
		for _, id := range s.Whitelist {
			wanted[id] = true
		}
		for i, p := range pkgs {
			w.whitelist[i] = wanted[p.ID]
		}
	}
	// Dopo il caricamento seleziona solo le righe in whitelist;
	// in assenza di whitelist seleziona tutte le righe.
	for i := range w.checked {
		w.checked[i] = w.inWhitelistLocked(i)
	}
	w.mu.Unlock()

	w.table.SetRowCount(len(pkgs))

	w.selectAll.SetValue(w.allWhitelistedLocked())
	if len(pkgs) == 0 {
		w.header.SetLabel("Il sistema è aggiornato. Nessun aggiornamento disponibile.")
		w.selectAll.SetValue(false)
	}
	w.header.Redraw()
	w.selectAll.Redraw()
	w.table.Redraw()
	w.refreshCounts()
}

// allWhitelistedLocked indica se tutte le righe in whitelist sono selezionate.
func (w *MainWin) allWhitelistedLocked() bool {
	any := false
	for i := range w.checked {
		if !w.inWhitelistLocked(i) {
			continue
		}
		any = true
		if !w.checked[i] {
			return false
		}
	}
	return any
}

// selectedPackages restituisce i pacchetti con checkbox attivo.
func (w *MainWin) selectedPackages() []Package {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []Package
	for i, p := range w.pkgs {
		if i < len(w.checked) && w.checked[i] {
			out = append(out, p)
		}
	}
	return out
}

// installSelected esegue l'aggiornamento dei pacchetti selezionati in un goroutine,
// mostrando una vera barra di avanzamento e ricaricando la lista al termine.
func (w *MainWin) installSelected(pkgs []Package) {
	if len(pkgs) == 0 {
		w.status.SetLabel("Nessun aggiornamento selezionato.")
		w.status.Redraw()
		return
	}

	// Disabilita il pulsante per evitare installazioni concorrenti.
	w.updateBtn.Deactivate()

	// Popup modale con barra determinata 0..N.
	w.showInstallProgress(len(pkgs))

	go func() {
		err := upgradePackages(pkgs, func(done, total int, pkg Package) {
			// Avanzamento: un pacchetto completato.
			w.post(evInstall{done: done, total: total, name: pkg.Name})
		})
		w.post(evInstallDone{err: err})
	}()
}

// applyInstallProgress aggiorna la barra di avanzamento (thread di FLTK).
func (w *MainWin) applyInstallProgress(done, total int, name string) {
	w.search.setProgress(done, total, fmt.Sprintf("Aggiornamento %d di %d: %s", done, total, name))
}

// finishInstall chiude il popup, riabilita il pulsante e ricarica la lista.
func (w *MainWin) finishInstall(err error) {
	if err != nil {
		w.status.SetLabel(fmt.Sprintf("Errore durante l'aggiornamento: %v", err))
		w.status.Redraw()
		w.updateBtn.Activate()
		w.hideProgress()
		return
	}

	w.updateBtn.Activate()
	// Aggiorna la lista dopo aver completato tutti gli aggiornamenti.
	w.showSearchProgress()
	go w.loadPackages()
}

// post invia un evento alla UI dal goroutine di lavoro.
func (w *MainWin) post(ev event) {
	w.events <- ev
}

// refreshCounts aggiorna l'header (selezione x su y) e la barra di stato.
func (w *MainWin) refreshCounts() {
	w.mu.Lock()
	sel := 0
	for _, c := range w.checked {
		if c {
			sel++
		}
	}
	wl := 0
	for _, b := range w.whitelist {
		if b {
			wl++
		}
	}
	total := len(w.pkgs)
	w.mu.Unlock()

	if total == 0 {
		w.header.SetLabel("Il sistema è aggiornato. Nessun aggiornamento disponibile.")
	} else if wl > 0 {
		w.header.SetLabel(fmt.Sprintf("Sono disponibili %d aggiornamenti — selezionati %d su %d (whitelist: %d).", total, sel, total, wl))
	} else {
		w.header.SetLabel(fmt.Sprintf("Sono disponibili %d aggiornamenti — selezionati %d su %d.", total, sel, total))
	}
	w.header.Redraw()

	w.status.SetLabel(fmt.Sprintf("%d di %d aggiornamenti selezionati", sel, total))
	w.status.Redraw()
}
