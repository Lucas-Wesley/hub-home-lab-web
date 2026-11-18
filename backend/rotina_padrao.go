package main

import "fmt"

// rotinaPadrao: a rotina com que a tela nasce, montada a partir do rascunho do caderno do Lucas
// (18/11/2026). Trabalho 8–12 e 14–18; Muay Thai seg, qua e sex; academia ter, qui e sáb; futebol
// ter às 19h30 (fixo: não remarca, então não avisa falta); quarta e sexta à noite com a namorada; sábado para mercado e comida da semana;
// domingo livre, com o planejamento da semana à noite. Sono de ~7 h (23h30–6h30).
func rotinaPadrao() configRotina {
	atividades := []atividade{
		{"ingles", "Inglês", "azul", 7, 7, 1, false},
		{"trabalho", "Trabalho", "cinza", 0, 0, 1, false},
		{"programacao", "Programação", "violeta", 3, 4, 1, false},
		{"comida", "Preparar comida", "amarelo", 1, 1, 1, false},
		{"planejar", "Planejar a semana", "turquesa", 1, 1, 1, false},
		{"namorada", "Namorada", "rosa", 0, 0, 1, false},
		{"muaythai", "Muay Thai", "laranja", 2, 3, 2, false},
		{"academia", "Academia", "laranja", 2, 3, 2, false},
		{"futebol", "Futebol", "laranja", 1, 1, 2, true},
		{"leitura", "Leitura", "verde", 5, 7, 2, false},
		{"mercado", "Mercado", "amarelo", 1, 1, 2, false},
		{"portugues", "Português", "turquesa", 2, 2, 3, false},
		{"filme", "Filme", "rosa", 1, 2, 3, false},
		{"almoco", "Almoço", "cinza", 0, 0, 3, false},
		{"casa", "Banho, janta e casa", "cinza", 0, 0, 3, false},
	}

	type b struct {
		dias             []int
		inicio, fim, ati string
		nota             string
	}
	semana := []int{0, 1, 2, 3, 4}
	blocos := []b{
		{semana, "07:00", "08:00", "ingles", ""},
		{semana, "08:00", "12:00", "trabalho", "+ 1 item de estudo"},
		{semana, "14:00", "18:00", "trabalho", "+ Anki nas pausas"},

		// Segunda
		{[]int{0}, "12:00", "14:00", "almoco", "Almoço, banho, rosto e dente"},
		{[]int{0}, "18:00", "18:45", "leitura", ""},
		{[]int{0}, "19:00", "20:00", "muaythai", ""},
		{[]int{0}, "20:00", "21:30", "casa", ""},
		{[]int{0}, "21:30", "23:00", "programacao", ""},

		// Terça: futebol às 19h30
		{[]int{1}, "12:00", "13:00", "academia", ""},
		{[]int{1}, "13:00", "14:00", "almoco", "Almoço, banho, rosto e dente"},
		{[]int{1}, "18:00", "19:00", "portugues", ""},
		{[]int{1}, "19:00", "21:00", "futebol", "Jogo às 19h30, com ida e volta"},
		{[]int{1}, "21:00", "22:30", "casa", ""},
		{[]int{1}, "22:30", "23:15", "leitura", ""},

		// Quarta: noite com a namorada
		{[]int{2}, "12:00", "12:30", "almoco", ""},
		{[]int{2}, "12:30", "14:00", "programacao", ""},
		{[]int{2}, "18:00", "19:00", "muaythai", ""},
		{[]int{2}, "19:00", "23:30", "namorada", ""},

		// Quinta
		{[]int{3}, "12:00", "13:00", "academia", ""},
		{[]int{3}, "13:00", "14:00", "almoco", "Almoço, banho, rosto e dente"},
		{[]int{3}, "18:00", "19:30", "programacao", ""},
		{[]int{3}, "19:30", "21:00", "casa", ""},
		{[]int{3}, "21:00", "22:00", "portugues", ""},
		{[]int{3}, "22:15", "23:00", "leitura", ""},

		// Sexta: noite com a namorada, com filme
		{[]int{4}, "12:00", "12:30", "almoco", ""},
		{[]int{4}, "12:30", "14:00", "programacao", ""},
		{[]int{4}, "18:00", "19:00", "muaythai", ""},
		{[]int{4}, "19:00", "20:30", "namorada", ""},
		{[]int{4}, "20:30", "22:30", "filme", "Com ela"},

		// Sábado
		{[]int{5}, "08:00", "09:00", "ingles", ""},
		{[]int{5}, "09:00", "12:00", "trabalho", ""},
		{[]int{5}, "12:00", "13:00", "academia", ""},
		{[]int{5}, "13:00", "14:00", "almoco", ""},
		{[]int{5}, "14:00", "15:00", "mercado", ""},
		{[]int{5}, "15:00", "18:00", "comida", "Café, almoço, lanche da tarde e janta da semana"},
		{[]int{5}, "20:00", "22:30", "filme", ""},
		{[]int{5}, "22:30", "23:15", "leitura", ""},

		// Domingo: livre
		{[]int{6}, "09:00", "09:30", "ingles", "Leve: Anki e revisão"},
		{[]int{6}, "20:30", "21:00", "planejar", "Ver a semana que passou e ajustar a próxima"},
		{[]int{6}, "21:30", "22:15", "leitura", ""},
	}

	c := configRotina{Atividades: atividades, Modelo: []bloco{}}
	for _, x := range blocos {
		for _, d := range x.dias {
			c.Modelo = append(c.Modelo, bloco{
				ID: fmt.Sprintf("m%d", len(c.Modelo)+1), Dia: d, Inicio: x.inicio, Fim: x.fim, Atividade: x.ati, Nota: x.nota,
			})
		}
	}
	return c
}
