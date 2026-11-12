package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata" // a imagem final não tem o banco de fusos; a hora da limpeza depende dele

	_ "modernc.org/sqlite"
)

var version = "dev"

//go:embed all:web/dist
var webDist embed.FS

func main() {
	dataDir := env("DATA_DIR", "./data")
	addr := env("ADDR", ":8080")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	db, err := abrirBanco(filepath.Join(dataDir, "hub.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fuso, err := time.LoadLocation(env("FUSO", "America/Sao_Paulo"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a, err := novoApp(db, fuso, time.Now, configDoAmbiente())
	if err != nil {
		log.Fatal(err)
	}
	go a.agendaDeLimpeza(ctx)
	go a.vigiar(ctx)

	web, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           newServer(a, web, version),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("hub %s em %s", version, addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		log.Print(err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func abrirBanco(caminho string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", caminho+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(esquema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// O Hub não guarda conteúdo dos outros apps: só a agenda de limpeza, o histórico das limpezas, as
// sugestões e a página Início. Horários em RFC 3339, UTC e sem fração de segundo: a ordem do texto é a ordem do tempo.
const esquema = `
CREATE TABLE IF NOT EXISTS agenda_limpeza (
	id              INTEGER PRIMARY KEY CHECK (id = 1),
	ativa           INTEGER NOT NULL DEFAULT 1,
	a_cada_dias     INTEGER NOT NULL DEFAULT 7,
	hora            INTEGER NOT NULL DEFAULT 4, -- hora local (FUSO)
	cache_de_build  INTEGER NOT NULL DEFAULT 1,
	imagens_soltas  INTEGER NOT NULL DEFAULT 1,
	proxima_em      TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS limpezas (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	origem           TEXT NOT NULL, -- 'agenda' ou 'manual'
	iniciada_em      TEXT NOT NULL,
	terminada_em     TEXT NOT NULL DEFAULT '',
	cache_liberado   INTEGER NOT NULL DEFAULT 0, -- bytes
	imagens_liberado INTEGER NOT NULL DEFAULT 0, -- bytes
	erro             TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS sugestoes (
	id            TEXT PRIMARY KEY,
	texto         TEXT NOT NULL,
	tela          TEXT NOT NULL DEFAULT '',
	estado        TEXT NOT NULL DEFAULT 'nova',
	resposta      TEXT NOT NULL DEFAULT '',
	criada_em     TEXT NOT NULL,
	atualizada_em TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS inicio (
	id            INTEGER PRIMARY KEY CHECK (id = 1),
	conteudo      TEXT NOT NULL, -- JSON do editor
	atualizada_em TEXT NOT NULL
);
`
