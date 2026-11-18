package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
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
}

type bloco struct {
	ID        string `json:"id"`
	Dia       int    `json:"dia"` // 0 segunda … 6 domingo
	Inicio    string `json:"inicio"`
	Fim       string `json:"fim"`
	Atividade string `json:"atividade"`
	Nota      string `json:"nota"`
	Feito     bool   `json:"feito"` // só nas semanas
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

// ---------- Validação ----------

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
		if !reHora.MatchString(b.Inicio) || !reHora.MatchString(b.Fim) || b.Inicio == "24:00" {
			return errors.New("Horário inválido (use HH:MM).")
		}
		if b.Fim <= b.Inicio {
			return fmt.Errorf("O bloco das %s termina antes de começar.", b.Inicio)
		}
		if existe != nil && !existe[b.Atividade] {
			return fmt.Errorf("Atividade desconhecida: %q.", b.Atividade)
		}
		if len(b.Nota) > 300 {
			return errors.New("Nota longa demais.")
		}
	}
	return nil
}

func (c configRotina) validar() error {
	if len(c.Atividades) == 0 || len(c.Atividades) > 60 {
		return errors.New("A rotina precisa de pelo menos uma atividade.")
	}
	existe := map[string]bool{}
	for _, a := range c.Atividades {
		if !reID.MatchString(a.ID) || existe[a.ID] {
			return errors.New("Atividade sem identificador ou repetida.")
		}
		existe[a.ID] = true
		if a.Nome == "" || len(a.Nome) > 60 {
			return errors.New("Toda atividade precisa de um nome.")
		}
		if !cores[a.Cor] {
			return fmt.Errorf("Cor inválida: %q.", a.Cor)
		}
		if a.MetaMin < 0 || a.MetaMax < a.MetaMin || a.MetaMax > 21 {
			return fmt.Errorf("Meta inválida em %s.", a.Nome)
		}
		if a.Prioridade < 1 || a.Prioridade > 3 {
			return fmt.Errorf("Prioridade inválida em %s.", a.Nome)
		}
	}
	for _, b := range c.Modelo {
		if b.Feito {
			return errors.New("O modelo não tem blocos feitos.")
		}
	}
	return validarBlocos(c.Modelo, existe)
}

// segunda confere que a data é uma segunda-feira.
func segunda(s string) (time.Time, bool) {
	d, err := time.Parse("2006-01-02", s)
	return d, err == nil && d.Weekday() == time.Monday
}

// ---------- Banco ----------

func (a *app) lerConfigRotina(r *http.Request) (configRotina, error) {
	var c configRotina
	var texto string
	err := a.db.QueryRowContext(r.Context(), `SELECT conteudo FROM rotina WHERE id = 1`).Scan(&texto)
	if errors.Is(err, sql.ErrNoRows) {
		return rotinaPadrao(), nil
	}
	if err != nil {
		return c, err
	}
	err = json.Unmarshal([]byte(texto), &c)
	return c, err
}

func (a *app) lerSemana(r *http.Request, inicio string) (semanaRotina, error) {
	s := semanaRotina{Inicio: inicio, Blocos: []bloco{}, Notas: map[string]string{}}
	var texto string
	err := a.db.QueryRowContext(r.Context(), `SELECT conteudo, atualizada_em FROM rotina_semanas WHERE inicio = ?`, inicio).Scan(&texto, &s.AtualizadaEm)
	if errors.Is(err, sql.ErrNoRows) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	var guardada struct {
		Blocos []bloco           `json:"blocos"`
		Notas  map[string]string `json:"notas"`
	}
	if err := json.Unmarshal([]byte(texto), &guardada); err != nil {
		return s, err
	}
	s.Planejada = true
	if guardada.Blocos != nil {
		s.Blocos = guardada.Blocos
	}
	if guardada.Notas != nil {
		s.Notas = guardada.Notas
	}
	return s, nil
}

func (a *app) gravarSemana(r *http.Request, s *semanaRotina) error {
	conteudo, err := json.Marshal(map[string]any{"blocos": s.Blocos, "notas": s.Notas})
	if err != nil {
		return err
	}
	s.AtualizadaEm = a.agora()
	s.Planejada = true
	_, err = a.db.ExecContext(r.Context(), `INSERT INTO rotina_semanas (inicio, conteudo, atualizada_em) VALUES (?, ?, ?)
		ON CONFLICT (inicio) DO UPDATE SET conteudo = excluded.conteudo, atualizada_em = excluded.atualizada_em`,
		s.Inicio, string(conteudo), s.AtualizadaEm)
	return err
}

// ---------- Handlers ----------

// GET /api/rotina: as atividades, o modelo e o dia de hoje no fuso de casa (para a tela abrir na
// semana e no dia certos, mesmo com o relógio do aparelho errado).
func (a *app) handleRotina(w http.ResponseWriter, r *http.Request) {
	c, err := a.lerConfigRotina(r)
	if err != nil {
		falha(w, err)
		return
	}
	hoje := a.relogio().In(a.fuso)
	dia := (int(hoje.Weekday()) + 6) % 7
	writeJSON(w, http.StatusOK, map[string]any{
		"atividades":   c.Atividades,
		"modelo":       c.Modelo,
		"hoje":         hoje.Format("2006-01-02"),
		"dia_de_hoje":  dia,
		"semana_atual": hoje.AddDate(0, 0, -dia).Format("2006-01-02"),
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

func (a *app) handleSemana(w http.ResponseWriter, r *http.Request) {
	inicio := r.PathValue("inicio")
	if _, ok := segunda(inicio); !ok {
		erroJSON(w, http.StatusBadRequest, "A semana começa numa segunda-feira (AAAA-MM-DD).")
		return
	}
	s, err := a.lerSemana(r, inicio)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// POST /api/rotina/semanas/{inicio}: planeja a semana copiando o modelo. Uma semana já planejada
// só é refeita com ?refazer=1 (o que estava nela se perde).
func (a *app) handlePlanejarSemana(w http.ResponseWriter, r *http.Request) {
	inicio := r.PathValue("inicio")
	if _, ok := segunda(inicio); !ok {
		erroJSON(w, http.StatusBadRequest, "A semana começa numa segunda-feira (AAAA-MM-DD).")
		return
	}
	atual, err := a.lerSemana(r, inicio)
	if err != nil {
		falha(w, err)
		return
	}
	if atual.Planejada && r.URL.Query().Get("refazer") != "1" {
		erroJSON(w, http.StatusConflict, "Esta semana já foi planejada.")
		return
	}
	c, err := a.lerConfigRotina(r)
	if err != nil {
		falha(w, err)
		return
	}
	s := semanaRotina{Inicio: inicio, Blocos: make([]bloco, len(c.Modelo)), Notas: map[string]string{}}
	copy(s.Blocos, c.Modelo)
	if err := a.gravarSemana(r, &s); err != nil {
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
	if err := a.gravarSemana(r, &s); err != nil {
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
