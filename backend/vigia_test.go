package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// H9: um problema avisa na segunda conferência seguida, uma vez só, e de novo se voltar.
func TestVigiaAvisaUmaVezDepoisDeConfirmar(t *testing.T) {
	var v vigia
	disco := problema{chave: "disco-vm", titulo: "Disco"}
	parado := problema{chave: "conteiner-acervo", titulo: "Parado"}

	if n := v.novos([]problema{disco, parado}); len(n) != 0 {
		t.Fatalf("na primeira vez não avisa (pode ser um deploy): %v", n)
	}
	if n := v.novos([]problema{disco}); len(n) != 1 || n[0].chave != "disco-vm" {
		t.Fatalf("segunda vez seguida: %v", n)
	}
	if n := v.novos([]problema{disco, parado}); len(n) != 0 {
		t.Fatalf("já avisado, e o contêiner voltou do zero: %v", n)
	}
	v.novos(nil) // resolvido
	v.novos([]problema{disco})
	if n := v.novos([]problema{disco}); len(n) != 1 {
		t.Fatalf("voltou depois de resolvido: %v", n)
	}
}

func TestProblemasDoServidor(t *testing.T) {
	if p := discoQuaseCheio("d", "Disco", uso{Total: 100, Usado: 84, Livre: 16}); p != nil {
		t.Errorf("84%% não é problema: %+v", p)
	}
	if p := discoQuaseCheio("d", "Disco", uso{Total: 100e9, Usado: 90e9, Livre: 10e9}); p == nil || p.titulo != "Disco em 90%" || p.texto[:15] != "Livre: 10,0 GB." {
		t.Errorf("90%%: %+v", p)
	}

	if p := discoFisicoComProblema(discoFisico{Caminho: "/dev/nvme0n1", Saude: "PASSED", VidaUtil: 97}); p != nil {
		t.Errorf("disco bom: %+v", p)
	}
	if p := discoFisicoComProblema(discoFisico{Caminho: "/dev/nvme0n1", Saude: "FAILED", VidaUtil: -1}); p == nil {
		t.Error("SMART falhou e não avisou")
	}
	if p := discoFisicoComProblema(discoFisico{Caminho: "/dev/nvme0n1", Saude: "PASSED", VidaUtil: 8}); p == nil {
		t.Error("vida útil acabando e não avisou")
	}

	if p := conteinerComProblema(container{Nome: "acervo-acervo-1", Estado: "running", Status: "Up 2 hours (healthy)"}); p != nil {
		t.Errorf("contêiner bom: %+v", p)
	}
	if p := conteinerComProblema(container{Nome: "acervo-acervo-1", Estado: "exited", Status: "Exited (1) 5 minutes ago"}); p == nil {
		t.Error("parado e não avisou")
	}
	if p := conteinerComProblema(container{Nome: "acervo-acervo-1", Estado: "running", Status: "Up 2 hours (unhealthy)"}); p == nil {
		t.Error("sem saúde e não avisou")
	}
}

func TestBackupAtrasado(t *testing.T) {
	agora := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	arquivo := filepath.Join(t.TempDir(), "ultimo")

	if p := backupComProblema(arquivo, agora); p == nil || p.titulo != "Backup sem registro" {
		t.Errorf("sem arquivo: %+v", p)
	}
	os.WriteFile(arquivo, []byte("2026-11-11T06:00:41+00:00 ok homelab-2026-11-11.tar.gz 31M\n"), 0o644)
	if p := backupComProblema(arquivo, agora); p != nil {
		t.Errorf("rodou hoje: %+v", p)
	}
	os.WriteFile(arquivo, []byte("2026-11-11T06:00:41+00:00 FALHOU (linha 42)\n"), 0o644)
	if p := backupComProblema(arquivo, agora); p != nil {
		t.Errorf("a falha de hoje o próprio script avisa: %+v", p)
	}
	if p := backupComProblema(arquivo, agora.Add(25*time.Hour)); p == nil || p.titulo != "Backup atrasado" {
		t.Errorf("há 31 horas: %+v", p)
	}
}
