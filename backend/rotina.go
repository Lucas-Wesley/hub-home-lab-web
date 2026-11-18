package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"time"
)

// Rotina (H14): as atividades com a meta da semana, um modelo de semana (o padrão) e cada semana
// planejada, que nasce como cópia do modelo e depois muda à vontade (feriado, evento, remanejar).
// Como o Início, o Hub guarda o JSON e confere só a forma: quem salva por último vale.

type atividade struct {
	ID         string `json:"id"`
	Nome       string `json:"nome"`
	Cor        string `json:"cor"`
	MetaMin    int    `json:"meta_min"` // vezes por semana; 0 e 0 = sem meta (trabalho, casa)
	MetaMax    int    `json:"meta_max"`
	Prioridade int    `json:"prioridade"` // 1 alta, 2 média, 3 baixa: a ordem de remanejar
	Fixa       bool   `json:"fixa"`       // não dá para remarcar (o futebol de terça): sem aviso de falta
}

// bloco: um horário da semana. Sem atividade, é um evento com título livre ("Encontro com amigos");
// sem começo e fim, é uma tarefa do dia. Extra = não veio do modelo (voz, evento, "+ Bloco"): refazer a
// semana pelo modelo não o apaga.
type bloco struct {
	ID        string `json:"id"`
	Dia       int    `json:"dia"` // 0 segunda … 6 domingo
	Inicio    string `json:"inicio"`
	Fim       string `json:"fim"`
	Atividade string `json:"atividade"`
	Titulo    string `json:"titulo"`
	Nota      string `json:"nota"`
	Feito     bool   `json:"feito"` // só nas semanas
	Extra     bool   `json:"extra"`
}

type configRotina struct {
	Atividades []atividade `json:"atividades"`
	Modelo     []bloco     `json:"modelo"`
}

type semanaRotina struct {
	Inicio       string            `json:"inicio"` // a segunda-feira, AAAA-MM-DD
	Planejada    bool              `json:"planejada"`
	Blocos       []bloco           `json:"blocos"`
	Notas        map[string]string `json:"notas"` // "0".."6" → "Feriado", "Aniversário de namoro"…
	AtualizadaEm string            `json:"atualizada_em"`
}

var (
	reHora = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$|^24:00$`)
	reID   = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)
	cores  = map[string]bool{"azul": true, "violeta": true, "verde": true, "laranja": true, "amarelo": true, "rosa": true, "turquesa": true, "cinza": true}
)

func novoID(prefixo string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefixo + hex.EncodeToString(b)
}

// ---------- Validação ----------

// validarBlocos: com `existe` (o modelo), todo bloco tem uma atividade do catálogo e horário. Sem ele
// (uma semana), vale também evento com título e tarefa sem horário.
func validarBlocos(blocos []bloco, existe map[string]bool) error {
	ids := map[string]bool{}
	for _, b := range blocos {
		if b.ID == "" || len(b.ID) > 40 || ids[b.ID] {
			return errors.New("Bloco sem identificador ou repetido.")
		}
		ids[b.ID] = true
		if b.Dia < 0 || b.Dia > 6 {
			return errors.New("Dia inválido.")
		}
		if len(b.Titulo) > 80 {
			return errors.New("Título longo demais.")
		}
		if len(b.Nota) > 300 {
			return errors.New("Nota longa demais.")
		}
		if existe != nil {
			if !existe[b.Atividade] {
				return fmt.Errorf("Atividade desconhecida: %q.", b.Atividade)
			}
		} else if b.Atividade == "" && b.Titulo == "" {
			return errors.New("O bloco precisa de uma atividade ou de um título.")
		}
		if b.Inicio == "" && b.Fim == "" && existe == nil {
			continue // tarefa do dia
		}
		if !reHora.MatchString(b.Inicio) || !reHora.MatchString(b.Fim) || b.Inicio == "24:00" {
			return errors.New("Horário inválido (use HH:MM).")
		}
		if b.Fim <= b.Inicio {
			return fmt.Errorf("O bloco das %s termina antes de começar.", b.Inicio)
		}
	}
	return nil
}

func validarAtividades(lista []atividade) (map[string]bool, error) {
	if len(lista) == 0 || len(lista) > 60 {
		return nil, errors.New("A rotina precisa de pelo menos uma atividade.")
	}
	existe := map[string]bool{}
	for _, a := range lista {
		if !reID.MatchString(a.ID) || existe[a.ID] {
			return nil, errors.New("Atividade sem identificador ou repetida.")
		}
		existe[a.ID] = true
		if a.Nome == "" || len(a.Nome) > 60 {
			return nil, errors.New("Toda atividade precisa de um nome.")
		}
		if !cores[a.Cor] {
			return nil, fmt.Errorf("Cor inválida: %q.", a.Cor)
		}
		if a.MetaMin < 0 || a.MetaMax < a.MetaMin || a.MetaMax > 21 {
			return nil, fmt.Errorf("Meta inválida em %s.", a.Nome)
		}
		if a.Prioridade < 1 || a.Prioridade > 3 {
			return nil, fmt.Errorf("Prioridade inválida em %s.", a.Nome)
		}
	}
	return existe, nil
}

func (c configRotina) validar() error {
	existe, err := validarAtividades(c.Atividades)
	if err != nil {
		return err
	}
	for _, b := range c.Modelo {
		if b.Feito || b.Titulo != "" {
			return errors.New("O modelo só tem atividades, sem feito nem eventos.")
		}
	}
	return validarBlocos(c.Modelo, existe)
}

// segunda confere que a data é uma segunda-feira.
func segunda(s string) (time.Time, bool) {
	d, err := time.Parse("2006-01-02", s)
	return d, err == nil && d.Weekday() == time.Monday
}

// semanaDe: a segunda-feira e o dia (0–6) de uma data.
func semanaDe(d time.Time) (string, int) {
	dia := (int(d.Weekday()) + 6) % 7
	return d.AddDate(0, 0, -dia).Format("2006-01-02"), dia
}

// choques: os blocos com horário que se sobrepõem a `b` no mesmo dia.
func choques(blocos []bloco, b bloco) []bloco {
	if b.Inicio == "" {
		return nil
	}
	var r []bloco
	for _, x := range blocos {
		if x.ID != b.ID && x.Dia == b.Dia && x.Inicio != "" && x.Inicio < b.Fim && b.Inicio < x.Fim {
			r = append(r, x)
		}
	}
	return r
}

// ---------- Banco ----------

func (a *app) lerConfigRotina(ctx context.Context) (configRotina, error) {
	var c configRotina
	var texto string
	err := a.db.QueryRowContext(ctx, `SELECT conteudo FROM rotina WHERE id = 1`).Scan(&texto)
	if errors.Is(err, sql.ErrNoRows) {
		return rotinaPadrao(), nil
	}
	if err != nil {
		return c, err
	}
	err = json.Unmarshal([]byte(texto), &c)
	return c, err
}

func (a *app) lerSemana(ctx context.Context, inicio string) (semanaRotina, error) {
	s := semanaRotina{Inicio: inicio, Blocos: []bloco{}, Notas: map[string]string{}}
	var texto string
	err := a.db.QueryRowContext(ctx, `SELECT conteudo, atualizada_em FROM rotina_semanas WHERE inicio = ?`, inicio).Scan(&texto, &s.AtualizadaEm)
	if errors.Is(err, sql.ErrNoRows) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	return s, desembrulhar(&s, texto)
}

func desembrulhar(s *semanaRotina, texto string) error {
	var guardada struct {
		Blocos []bloco           `json:"blocos"`
		Notas  map[string]string `json:"notas"`
	}
	if err := json.Unmarshal([]byte(texto), &guardada); err != nil {
		return err
	}
	s.Planejada = true
	if guardada.Blocos != nil {
		s.Blocos = guardada.Blocos
	}
	if guardada.Notas != nil {
		s.Notas = guardada.Notas
	}
	return nil
}

func (a *app) gravarSemana(ctx context.Context, s *semanaRotina) error {
	conteudo, err := json.Marshal(map[string]any{"blocos": s.Blocos, "notas": s.Notas})
	if err != nil {
		return err
	}
	s.AtualizadaEm = a.agora()
	s.Planejada = true
	_, err = a.db.ExecContext(ctx, `INSERT INTO rotina_semanas (inicio, conteudo, atualizada_em) VALUES (?, ?, ?)
		ON CONFLICT (inicio) DO UPDATE SET conteudo = excluded.conteudo, atualizada_em = excluded.atualizada_em`,
		s.Inicio, string(conteudo), s.AtualizadaEm)
	return err
}

// copiaDoModelo: os blocos do modelo com identificadores novos, para uma semana.
func copiaDoModelo(c configRotina) []bloco {
	r := make([]bloco, len(c.Modelo))
	copy(r, c.Modelo)
	return r
}

// ---------- Handlers ----------

// GET /api/rotina: as atividades, o modelo e o dia de hoje no fuso de casa (para a tela abrir na
// semana e no dia certos, mesmo com o relógio do aparelho errado).
func (a *app) handleRotina(w http.ResponseWriter, r *http.Request) {
	c, err := a.lerConfigRotina(r.Context())
	if err != nil {
		falha(w, err)
		return
	}
	hoje := a.relogio().In(a.fuso)
	semanaAtual, dia := semanaDe(hoje)
	writeJSON(w, http.StatusOK, map[string]any{
		"atividades":   c.Atividades,
		"modelo":       c.Modelo,
		"hoje":         hoje.Format("2006-01-02"),
		"dia_de_hoje":  dia,
		"semana_atual": semanaAtual,
		"ia":           a.ia != nil,
	})
}

func (a *app) handleSalvarRotina(w http.ResponseWriter, r *http.Request) {
	var c configRotina
	if !lerCorpo(w, r, &c) {
		return
	}
	if c.Modelo == nil {
		c.Modelo = []bloco{}
	}
	if err := c.validar(); err != nil {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	conteudo, _ := json.Marshal(c)
	_, err := a.db.ExecContext(r.Context(), `INSERT INTO rotina (id, conteudo, atualizada_em) VALUES (1, ?, ?)
		ON CONFLICT (id) DO UPDATE SET conteudo = excluded.conteudo, atualizada_em = excluded.atualizada_em`,
		string(conteudo), a.agora())
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// GET /api/rotina/semanas?de=AAAA-MM-DD&ate=AAAA-MM-DD: as semanas planejadas no intervalo (o mês).
func (a *app) handleSemanas(w http.ResponseWriter, r *http.Request) {
	de, ate := r.URL.Query().Get("de"), r.URL.Query().Get("ate")
	if _, err := time.Parse("2006-01-02", de); err != nil {
		erroJSON(w, http.StatusBadRequest, "Data inicial inválida.")
		return
	}
	if _, err := time.Parse("2006-01-02", ate); err != nil {
		erroJSON(w, http.StatusBadRequest, "Data final inválida.")
		return
	}
	linhas, err := a.db.QueryContext(r.Context(), `SELECT inicio, conteudo, atualizada_em FROM rotina_semanas WHERE inicio >= ? AND inicio <= ? ORDER BY inicio`, de, ate)
	if err != nil {
		falha(w, err)
		return
	}
	defer linhas.Close()
	lista := []semanaRotina{}
	for linhas.Next() {
		s := semanaRotina{Blocos: []bloco{}, Notas: map[string]string{}}
		var texto string
		if err := linhas.Scan(&s.Inicio, &texto, &s.AtualizadaEm); err != nil {
			falha(w, err)
			return
		}
		if err := desembrulhar(&s, texto); err != nil {
			falha(w, err)
			return
		}
		lista = append(lista, s)
	}
	writeJSON(w, http.StatusOK, lista)
}

func (a *app) handleSemana(w http.ResponseWriter, r *http.Request) {
	inicio := r.PathValue("inicio")
	if _, ok := segunda(inicio); !ok {
		erroJSON(w, http.StatusBadRequest, "A semana começa numa segunda-feira (AAAA-MM-DD).")
		return
	}
	s, err := a.lerSemana(r.Context(), inicio)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// POST /api/rotina/semanas/{inicio}: planeja a semana copiando o modelo. Uma semana já planejada
// só é refeita com ?refazer=1: volta ao modelo, mas os eventos e blocos extras ficam.
func (a *app) handlePlanejarSemana(w http.ResponseWriter, r *http.Request) {
	inicio := r.PathValue("inicio")
	if _, ok := segunda(inicio); !ok {
		erroJSON(w, http.StatusBadRequest, "A semana começa numa segunda-feira (AAAA-MM-DD).")
		return
	}
	atual, err := a.lerSemana(r.Context(), inicio)
	if err != nil {
		falha(w, err)
		return
	}
	if atual.Planejada && r.URL.Query().Get("refazer") != "1" {
		erroJSON(w, http.StatusConflict, "Esta semana já foi planejada.")
		return
	}
	c, err := a.lerConfigRotina(r.Context())
	if err != nil {
		falha(w, err)
		return
	}
	s := semanaRotina{Inicio: inicio, Blocos: copiaDoModelo(c), Notas: map[string]string{}}
	for _, b := range atual.Blocos {
		if b.Extra {
			s.Blocos = append(s.Blocos, b)
		}
	}
	if err := a.gravarSemana(r.Context(), &s); err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (a *app) handleSalvarSemana(w http.ResponseWriter, r *http.Request) {
	inicio := r.PathValue("inicio")
	if _, ok := segunda(inicio); !ok {
		erroJSON(w, http.StatusBadRequest, "A semana começa numa segunda-feira (AAAA-MM-DD).")
		return
	}
	var corpo struct {
		Blocos []bloco           `json:"blocos"`
		Notas  map[string]string `json:"notas"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	if corpo.Blocos == nil {
		corpo.Blocos = []bloco{}
	}
	if corpo.Notas == nil {
		corpo.Notas = map[string]string{}
	}
	// A semana pode citar uma atividade que saiu do catálogo depois: não confere o catálogo.
	if err := validarBlocos(corpo.Blocos, nil); err != nil {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	for dia, nota := range corpo.Notas {
		if len(dia) != 1 || dia < "0" || dia > "6" || len(nota) > 120 {
			erroJSON(w, http.StatusBadRequest, "Nota do dia inválida.")
			return
		}
	}
	s := semanaRotina{Inicio: inicio, Blocos: corpo.Blocos, Notas: corpo.Notas}
	if err := a.gravarSemana(r.Context(), &s); err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (a *app) handleApagarSemana(w http.ResponseWriter, r *http.Request) {
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM rotina_semanas WHERE inicio = ?`, r.PathValue("inicio")); err != nil {
		falha(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Eventos com data (da voz, do mês) ----------

type eventoNovo struct {
	Data      string `json:"data"` // AAAA-MM-DD
	Inicio    string `json:"inicio"`
	Fim       string `json:"fim"`
	Atividade string `json:"atividade"`
	Titulo    string `json:"titulo"`
	Nota      string `json:"nota"`
}

type eventoAdicionado struct {
	eventoNovo
	ID           string   `json:"id"`
	Semana       string   `json:"semana"`
	Dia          int      `json:"dia"`
	SemanaNova   bool     `json:"semana_nova"` // a semana ainda não tinha plano e nasceu do modelo
	ChocaCom     []string `json:"choca_com"`   // os nomes do que já estava no horário
	NomeDoEvento string   `json:"nome"`
}

// adicionarEventos põe cada evento na semana dele, sem tirar nada: se já há algo no horário, os dois
// ficam e a tela marca o dia com "!". Uma semana sem plano nasce do modelo antes.
func (a *app) adicionarEventos(ctx context.Context, eventos []eventoNovo) ([]eventoAdicionado, error) {
	c, err := a.lerConfigRotina(ctx)
	if err != nil {
		return nil, err
	}
	nomes := map[string]string{}
	for _, at := range c.Atividades {
		nomes[at.ID] = at.Nome
	}
	nomeDe := func(b bloco) string {
		if b.Titulo != "" {
			return b.Titulo
		}
		if n, ok := nomes[b.Atividade]; ok {
			return n
		}
		return "atividade apagada"
	}

	// Confere todos antes de gravar qualquer um.
	for i, ev := range eventos {
		d, err := time.Parse("2006-01-02", ev.Data)
		if err != nil {
			return nil, fmt.Errorf("Data inválida: %q.", ev.Data)
		}
		_, dia := semanaDe(d)
		if _, ok := nomes[ev.Atividade]; !ok {
			eventos[i].Atividade = ""
		}
		b := bloco{ID: "x", Dia: dia, Inicio: ev.Inicio, Fim: ev.Fim, Atividade: eventos[i].Atividade, Titulo: ev.Titulo, Nota: ev.Nota}
		if err := validarBlocos([]bloco{b}, nil); err != nil {
			return nil, err
		}
	}

	semanas := map[string]*semanaRotina{}
	novas := map[string]bool{}
	var feitos []eventoAdicionado
	for _, ev := range eventos {
		d, _ := time.Parse("2006-01-02", ev.Data)
		inicio, dia := semanaDe(d)
		s, ok := semanas[inicio]
		if !ok {
			lida, err := a.lerSemana(ctx, inicio)
			if err != nil {
				return nil, err
			}
			if !lida.Planejada {
				lida.Blocos = copiaDoModelo(c)
				novas[inicio] = true
			}
			s = &lida
			semanas[inicio] = s
		}
		b := bloco{ID: novoID("e"), Dia: dia, Inicio: ev.Inicio, Fim: ev.Fim, Atividade: ev.Atividade, Titulo: ev.Titulo, Nota: ev.Nota, Extra: true}
		chocam := []string{}
		for _, x := range choques(s.Blocos, b) {
			chocam = append(chocam, nomeDe(x))
		}
		s.Blocos = append(s.Blocos, b)
		feitos = append(feitos, eventoAdicionado{eventoNovo: ev, ID: b.ID, Semana: inicio, Dia: dia, SemanaNova: novas[inicio], ChocaCom: chocam, NomeDoEvento: nomeDe(b)})
	}
	chaves := make([]string, 0, len(semanas))
	for k := range semanas {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	for _, k := range chaves {
		if err := a.gravarSemana(ctx, semanas[k]); err != nil {
			return nil, err
		}
	}
	return feitos, nil
}

// POST /api/rotina/eventos {eventos: [...]}
func (a *app) handleAdicionarEventos(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Eventos []eventoNovo `json:"eventos"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	if len(corpo.Eventos) == 0 || len(corpo.Eventos) > 30 {
		erroJSON(w, http.StatusBadRequest, "Nenhum evento para adicionar.")
		return
	}
	feitos, err := a.adicionarEventos(r.Context(), corpo.Eventos)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"eventos": feitos})
}
