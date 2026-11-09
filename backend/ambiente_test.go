package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

// ambiente: o servidor do Hub com banco próprio, um Docker e um Proxmox falsos, uma pasta de
// apps e um /proc inventados, e um relógio que o teste avança à mão.
type ambiente struct {
	t       *testing.T
	srv     *httptest.Server
	app     *app
	apps    string
	docker  *dockerFalso
	proxmox *httptest.Server
	agora   time.Time
	fuso    *time.Location
}

const tokenDeTeste = "hub@pve!painel=00000000-0000-0000-0000-000000000000"

func novoAmbiente(t *testing.T) *ambiente {
	t.Helper()
	db, err := abrirBanco(filepath.Join(t.TempDir(), "teste.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	fuso, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	e := &ambiente{t: t, fuso: fuso, agora: time.Date(2026, 11, 9, 15, 0, 0, 0, fuso)}

	e.apps = t.TempDir()
	escrever(t, filepath.Join(e.apps, "biblioteca", "biblioteca.db"), 40000)
	escrever(t, filepath.Join(e.apps, "biblioteca", "biblioteca.db-wal"), 8000)
	escrever(t, filepath.Join(e.apps, "biblioteca", "books", "livro.epub"), 300000)
	escrever(t, filepath.Join(e.apps, "acervo", "home", ".claude", "projects", "a.jsonl"), 20000)
	escrever(t, filepath.Join(e.apps, "acervo", "notas.txt"), 100)

	proc := t.TempDir()
	os.WriteFile(filepath.Join(proc, "meminfo"), []byte("MemTotal:        8000000 kB\nMemFree:          600000 kB\nMemAvailable:    6500000 kB\n"), 0o644)
	os.WriteFile(filepath.Join(proc, "loadavg"), []byte("0.15 0.20 0.10 1/300 1234\n"), 0o644)
	os.WriteFile(filepath.Join(proc, "uptime"), []byte("18000.50 70000.00\n"), 0o644)

	e.docker = novoDockerFalso()
	dockerSrv := httptest.NewServer(e.docker)
	t.Cleanup(dockerSrv.Close)

	e.proxmox = httptest.NewTLSServer(proxmoxFalso(t))
	t.Cleanup(e.proxmox.Close)

	cfg := config{
		apps:   e.apps,
		proc:   proc,
		docker: dockerSrv.URL,
		proxmox: configProxmox{
			url: e.proxmox.URL, no: "pve", token: tokenDeTeste, impressao: impressaoDe(e.proxmox),
		},
	}
	a, err := novoApp(db, fuso, func() time.Time { return e.agora }, cfg)
	if err != nil {
		t.Fatal(err)
	}
	e.app = a
	web := fstest.MapFS{"index.html": {Data: []byte(`<div id="app"></div>`)}}
	e.srv = httptest.NewServer(newServer(a, web, "teste"))
	t.Cleanup(e.srv.Close)
	return e
}

func escrever(t *testing.T, caminho string, tamanho int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, bytes.Repeat([]byte("x"), tamanho), 0o644); err != nil {
		t.Fatal(err)
	}
}

// impressaoDe: o SHA-256 do certificado do servidor de teste, no formato que o Proxmox mostra.
func impressaoDe(srv *httptest.Server) string {
	soma := sha256.Sum256(srv.Certificate().Raw)
	h := strings.ToUpper(hex.EncodeToString(soma[:]))
	partes := []string{}
	for i := 0; i < len(h); i += 2 {
		partes = append(partes, h[i:i+2])
	}
	return strings.Join(partes, ":")
}

// dockerFalso responde como o Docker atrás do docker-socket-proxy e anota as limpezas pedidas.
type dockerFalso struct {
	mu      sync.Mutex
	pedidos []string
	cache   int64
	falhar  bool
	esperar chan struct{} // se não for nil, o prune do cache espera por ele
}

func novoDockerFalso() *dockerFalso {
	return &dockerFalso{cache: 14_642_571_481}
}

func (d *dockerFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	d.pedidos = append(d.pedidos, r.Method+" "+r.URL.RequestURI())
	cache, falhar, esperar := d.cache, d.falhar, d.esperar
	d.mu.Unlock()
	switch r.Method + " " + r.URL.Path {
	case "GET /v1.47/system/df":
		fmt.Fprintf(w, `{
			"LayersSize": 1946260943,
			"Images": [
				{"RepoTags": ["flashcards:abc"], "Size": 750002124, "SharedSize": 731627799, "Containers": 1},
				{"RepoTags": ["flashcards:velha"], "Size": 750000000, "SharedSize": 731627799, "Containers": 0},
				{"RepoTags": [], "Size": 99770000, "SharedSize": -1, "Containers": 0}
			],
			"Containers": [
				{"Names": ["/caddy"], "Image": "caddy:2-alpine", "State": "running", "Status": "Up 5 hours", "SizeRw": 4096},
				{"Names": ["/flashcards-flashcards-1"], "Image": "flashcards:abc", "State": "running", "Status": "Up 2 days", "SizeRw": 0}
			],
			"BuildCache": [
				{"Size": %d, "InUse": false, "Shared": false},
				{"Size": 1000, "InUse": true, "Shared": false},
				{"Size": 5000, "InUse": false, "Shared": true}
			]
		}`, cache)
	case "POST /v1.47/build/prune":
		if esperar != nil {
			<-esperar
		}
		if falhar {
			http.Error(w, `{"message":"falhou"}`, http.StatusInternalServerError)
			return
		}
		d.mu.Lock()
		d.cache = 0
		d.mu.Unlock()
		fmt.Fprintf(w, `{"CachesDeleted": ["a"], "SpaceReclaimed": %d}`, cache)
	case "POST /v1.47/images/prune":
		fmt.Fprint(w, `{"ImagesDeleted": [], "SpaceReclaimed": 99770000}`)
	default:
		http.Error(w, `{"message":"proibido"}`, http.StatusForbidden)
	}
}

func (d *dockerFalso) limpezas() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	var l []string
	for _, p := range d.pedidos {
		if strings.HasPrefix(p, "POST") {
			l = append(l, p)
		}
	}
	return l
}

// proxmoxFalso responde as quatro leituras do painel com dados parecidos com os do M920q.
func proxmoxFalso(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "PVEAPIToken="+tokenDeTeste {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var data string
		switch r.URL.Path {
		case "/api2/json/nodes/pve/status":
			data = `{"cpu": 0.031, "cpuinfo": {"cpus": 12, "model": "Intel(R) Core(TM) i7-8700T CPU @ 2.40GHz"},
				"memory": {"total": 16642998272, "used": 9000000000, "free": 7642998272},
				"swap": {"total": 8589930496, "used": 0, "free": 8589930496},
				"rootfs": {"total": 72694968320, "used": 10000000000, "avail": 62694968320},
				"uptime": 500000, "pveversion": "pve-manager/9.2.24/abc"}`
		case "/api2/json/nodes/pve/storage":
			data = `[{"storage": "local-lvm", "type": "lvmthin", "content": "rootdir,images", "total": 151610736640, "used": 20000000000, "avail": 131610736640, "active": 1},
				{"storage": "local", "type": "dir", "content": "iso,backup", "total": 72694968320, "used": 10000000000, "avail": 62694968320, "active": 1}]`
		case "/api2/json/cluster/resources":
			data = `[{"vmid": 100, "name": "apps", "type": "qemu", "status": "running", "node": "pve", "cpu": 0.05, "maxcpu": 4,
				"mem": 2000000000, "maxmem": 8589934592, "disk": 0, "maxdisk": 107374182400, "uptime": 400000}]`
		case "/api2/json/nodes/pve/disks/list":
			// O token sem Sys.Audit não lê os discos: o resto da visão vem com um aviso.
			w.WriteHeader(http.StatusForbidden)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, `{"data": %s}`, data)
	})
}

func (e *ambiente) pedir(metodo, caminho string, corpo any, destino any) int {
	e.t.Helper()
	var leitor *bytes.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(b)
	} else {
		leitor = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(metodo, e.srv.URL+caminho, leitor)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	if destino != nil {
		if err := json.NewDecoder(resp.Body).Decode(destino); err != nil {
			e.t.Fatalf("%s %s: %v", metodo, caminho, err)
		}
	}
	return resp.StatusCode
}
