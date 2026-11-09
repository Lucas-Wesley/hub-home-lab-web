package main

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// uso: quanto cabe e quanto já foi usado, em bytes.
type uso struct {
	Total uint64 `json:"total"`
	Usado uint64 `json:"usado"`
	Livre uint64 `json:"livre"`
}

// lerDisco mede o sistema de arquivos onde fica o caminho. Na VM, /srv/data está no disco
// principal dela, então o resultado é o disco inteiro da VM.
func lerDisco(caminho string) (uso, error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(caminho, &s); err != nil {
		return uso{}, err
	}
	bloco := uint64(s.Bsize)
	return uso{
		Total: s.Blocks * bloco,
		Usado: (s.Blocks - s.Bfree) * bloco,
		Livre: s.Bavail * bloco,
	}, nil
}

// pasta: o que um app guarda, com o tamanho ocupado no disco (como o du).
type pasta struct {
	Nome      string  `json:"nome"`   // o nome no disco
	Titulo    string  `json:"titulo"` // como aparece no painel
	Bytes     int64   `json:"bytes"`
	Itens     []pasta `json:"itens,omitempty"`
	SemAcesso int     `json:"sem_acesso,omitempty"` // pastas que o painel não pôde ler
}

var titulosDosApps = map[string]string{
	"biblioteca": "Biblioteca Digital",
	"acervo":     "Acervo",
	"flashcards": "Flashcards",
	"streaming":  "Streaming",
	"prowlarr":   "Prowlarr",
	"caddy":      "Caddy (proxy)",
	"hub":        "Hub",
}

// Nomes conhecidos das pastas dentro de cada app; o resto aparece com o nome do disco.
var titulosDasPastas = map[string]string{
	"books":       "Livros",
	"covers":      "Capas",
	"referencias": "Referências",
	"conteudo":    "Temas",
	"revisoes":    "Revisões",
	"home":        "Claude: histórico e configuração",
	"media":       "Mídia",
	"geracoes":    "Pedidos à IA",
	"torrents":    "Torrents",
	"downloads":   "Downloads",
	"Definitions": "Definições dos indexadores",
	"logs":        "Registros",
}

const (
	itemBanco  = "Banco de dados"
	itemOutros = "Outros arquivos"
)

// medirApps mede cada pasta de /srv/data e o que há dentro dela, do maior para o menor.
func medirApps(raiz string) ([]pasta, error) {
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		return nil, err
	}
	apps := []pasta{}
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		apps = append(apps, medirApp(filepath.Join(raiz, e.Name())))
	}
	ordenar(apps)
	return apps, nil
}

func medirApp(dir string) pasta {
	nome := filepath.Base(dir)
	app := pasta{Nome: nome, Titulo: tituloOu(titulosDosApps, nome)}
	entradas, err := os.ReadDir(dir)
	if err != nil {
		app.SemAcesso = 1
		return app
	}
	// O SQLite deixa o banco em três arquivos (.db, -wal, -shm); arquivos soltos viram um item só.
	banco := pasta{Nome: "*.db", Titulo: itemBanco}
	outros := pasta{Nome: "*", Titulo: itemOutros}
	for _, e := range entradas {
		caminho := filepath.Join(dir, e.Name())
		if e.IsDir() {
			p := pasta{Nome: e.Name(), Titulo: tituloOu(titulosDasPastas, e.Name())}
			p.Bytes, p.SemAcesso = medir(caminho)
			app.Itens = append(app.Itens, p)
			continue
		}
		n, _ := medir(caminho)
		if ehBanco(e.Name()) {
			banco.Bytes += n
		} else {
			outros.Bytes += n
		}
	}
	for _, p := range []pasta{banco, outros} {
		if p.Bytes > 0 {
			app.Itens = append(app.Itens, p)
		}
	}
	for _, p := range app.Itens {
		app.Bytes += p.Bytes
		app.SemAcesso += p.SemAcesso
	}
	ordenar(app.Itens)
	return app
}

func ehBanco(nome string) bool {
	for _, fim := range []string{".db", ".db-wal", ".db-shm", ".sqlite", ".sqlite3"} {
		if strings.HasSuffix(nome, fim) {
			return true
		}
	}
	return false
}

// medir soma o espaço ocupado (blocos no disco, não o tamanho aparente) e conta as pastas que
// não puderam ser lidas. Não segue links simbólicos.
func medir(caminho string) (bytes int64, semAcesso int) {
	filepath.WalkDir(caminho, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			semAcesso++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if info, err := d.Info(); err == nil {
			bytes += ocupado(info)
		}
		return nil
	})
	return bytes, semAcesso
}

func ocupado(info fs.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return int64(st.Blocks) * 512
	}
	return info.Size()
}

func tituloOu(titulos map[string]string, nome string) string {
	if t, ok := titulos[nome]; ok {
		return t
	}
	return nome
}

func ordenar(p []pasta) {
	sort.SliceStable(p, func(i, j int) bool { return p[i].Bytes > p[j].Bytes })
}

// cacheDeMedidas: percorrer as pastas custa disco; a medida vale por um minuto.
type cacheDeMedidas struct {
	mu       sync.Mutex
	apps     []pasta
	err      error
	medidoEm time.Time
}

const validadeDasMedidas = time.Minute

func (c *cacheDeMedidas) ler(raiz string, agora time.Time, forcar bool) ([]pasta, time.Time, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if forcar || c.medidoEm.IsZero() || agora.Sub(c.medidoEm) > validadeDasMedidas {
		c.apps, c.err = medirApps(raiz)
		c.medidoEm = agora
	}
	return c.apps, c.medidoEm, c.err
}

// ---------- API ----------

type respostaArmazenamento struct {
	Disco      *uso         `json:"disco"` // disco da VM
	DiscoErro  string       `json:"disco_erro,omitempty"`
	Apps       []pasta      `json:"apps"`
	AppsErro   string       `json:"apps_erro,omitempty"`
	Docker     *usoDoDocker `json:"docker"`
	DockerErro string       `json:"docker_erro,omitempty"`
	MedidoEm   string       `json:"medido_em"`
}

// handleArmazenamento: o disco da VM, quanto cada app guarda e o que o Docker ocupa. Cada parte
// falha sozinha: sem Docker, o resto aparece igual. "?atualizar=1" mede de novo na hora.
func (a *app) handleArmazenamento(w http.ResponseWriter, r *http.Request) {
	resp := respostaArmazenamento{Apps: []pasta{}}
	if d, err := lerDisco(a.cfg.apps); err != nil {
		resp.DiscoErro = err.Error()
	} else {
		resp.Disco = &d
	}

	apps, medidoEm, err := a.medidas.ler(a.cfg.apps, a.relogio(), r.URL.Query().Get("atualizar") == "1")
	if err != nil {
		resp.AppsErro = "Não deu para ler as pastas dos apps: " + err.Error()
	} else {
		resp.Apps = apps
	}
	resp.MedidoEm = hora(medidoEm)

	if a.docker == nil {
		resp.DockerErro = "O Docker não está ligado ao painel (falta DOCKER_URL)."
	} else if d, err := a.docker.uso(r.Context()); err != nil {
		resp.DockerErro = "O Docker não respondeu: " + err.Error()
	} else {
		resp.Docker = &d
	}
	writeJSON(w, http.StatusOK, resp)
}
