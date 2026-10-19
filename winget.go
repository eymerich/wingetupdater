package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Package rappresenta un pacchetto installato con un aggiornamento disponibile.
type Package struct {
	Name      string
	ID        string
	Version   string
	Available string
	Source    string
}

// listUpgradable esegue winget e restituisce i pacchetti con aggiornamenti.
func listUpgradable() ([]Package, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "winget", "upgrade",
		"--include-unknown",
		"--accept-source-agreements",
		"--disable-interactivity",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// winget può uscire con codici non-zero anche in caso di successo parziale,
	// quindi non trattiamo l'errore di exit come fatale se abbiamo output.
	_ = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("timeout durante l'esecuzione di winget")
	}

	out := stdout.String()
	if strings.TrimSpace(out) == "" {
		return nil, fmt.Errorf("nessun output da winget: %s", strings.TrimSpace(stderr.String()))
	}

	return parseWingetOutput(out), nil
}

// colonne fisse dell'output tabellare di winget.
const (
	wingetColID        = 67
	wingetColVersion   = 106
	wingetColAvailable = 127
	wingetColSource    = 147
)

// parseWingetOutput estrae i pacchetti dall'output testuale a colonne fisse.
func parseWingetOutput(out string) []Package {
	var pkgs []Package
	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	headerSeen := false
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")

		// Salta righe vuote.
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Individua l'header per confermare la struttura e poi salta la riga di separazione.
		if !headerSeen {
			if strings.HasPrefix(line, "Nome") || strings.HasPrefix(line, "Name") {
				headerSeen = true
			}
			continue
		}

		// Salta la riga di separazione composta da trattini.
		if strings.HasPrefix(line, strings.Repeat("-", 10)) {
			continue
		}

		pkg := Package{
			Name:      strings.TrimSpace(field(line, 0, wingetColID)),
			ID:        strings.TrimSpace(field(line, wingetColID, wingetColVersion)),
			Version:   strings.TrimSpace(field(line, wingetColVersion, wingetColAvailable)),
			Available: strings.TrimSpace(field(line, wingetColAvailable, wingetColSource)),
			Source:    strings.TrimSpace(field(line, wingetColSource, len(line))),
		}

		// Righe che non hanno almeno un ID valido vengono ignorate
		// (es. messaggi finali come "X aggiornamenti disponibili").
		if pkg.ID == "" {
			continue
		}

		pkgs = append(pkgs, pkg)
	}

	return pkgs
}

// field restituisce la sottostringa di line compresa tra [start, end), gestendo
// indici oltre la lunghezza della riga.
func field(line string, start, end int) string {
	if start >= len(line) {
		return ""
	}
	if end > len(line) {
		end = len(line)
	}
	if start >= end {
		return ""
	}
	return line[start:end]
}

// upgradePackages aggiorna i pacchetti indicati eseguendo winget uno alla volta.
// onProgress, se non nil, viene chiamata dopo ogni pacchetto aggiornato con
// il conteggio dei completati, il totale e il pacchetto appena terminato.
func upgradePackages(pkgs []Package, onProgress func(done, total int, pkg Package)) error {
	total := len(pkgs)
	for i, p := range pkgs {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

		cmd := exec.CommandContext(ctx, "winget", "upgrade",
			"--id", p.ID,
			"--exact",
			"--silent",
			"--include-unknown",
			"--accept-source-agreements",
			"--accept-package-agreements",
			"--disable-interactivity",
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		cancel()

		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timeout durante l'aggiornamento di %s", p.Name)
		}
		if err != nil {
			return fmt.Errorf("aggiornamento di %s non riuscito: %v (%s)",
				p.Name, err, strings.TrimSpace(stderr.String()))
		}

		if onProgress != nil {
			onProgress(i+1, total, p)
		}
	}
	return nil
}
