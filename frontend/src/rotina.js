// Contas da tela Rotina: datas da semana, conflitos de horário, sono e o placar das metas.
// As datas são "AAAA-MM-DD" em UTC ao meio-dia, para o fuso e o horário de verão não mudarem o dia.

export const DIAS = ['Segunda', 'Terça', 'Quarta', 'Quinta', 'Sexta', 'Sábado', 'Domingo']
export const LETRAS = ['S', 'T', 'Q', 'Q', 'S', 'S', 'D']
export const CORES = ['azul', 'violeta', 'verde', 'laranja', 'amarelo', 'rosa', 'turquesa', 'cinza']
export const PRIORIDADES = { 1: 'Alta', 2: 'Média', 3: 'Baixa' }

function data(iso) {
  return new Date(`${iso}T12:00:00Z`)
}

export function somarDias(iso, n) {
  const d = data(iso)
  d.setUTCDate(d.getUTCDate() + n)
  return d.toISOString().slice(0, 10)
}

export function diaDoMes(inicio, dia) {
  return data(somarDias(inicio, dia)).getUTCDate()
}

// "16 – 22 de nov." ou "30 de nov. – 6 de dez."
export function rotuloSemana(inicio) {
  const a = data(inicio)
  const b = data(somarDias(inicio, 6))
  const mes = (d) => d.toLocaleDateString('pt-BR', { month: 'short', timeZone: 'UTC' })
  if (a.getUTCMonth() === b.getUTCMonth()) return `${a.getUTCDate()} – ${b.getUTCDate()} de ${mes(b)}`
  return `${a.getUTCDate()} de ${mes(a)} – ${b.getUTCDate()} de ${mes(b)}`
}

export function minutos(hhmm) {
  const [h, m] = hhmm.split(':').map(Number)
  return h * 60 + m
}

// "1 h 30", "45 min", "4 h"
export function duracao(min) {
  if (min < 60) return `${min} min`
  const h = Math.floor(min / 60)
  const m = min % 60
  return m ? `${h} h ${String(m).padStart(2, '0')}` : `${h} h`
}

export function doDia(blocos, dia) {
  return blocos.filter((b) => b.dia === dia).sort((a, b) => a.inicio.localeCompare(b.inicio) || a.fim.localeCompare(b.fim))
}

// Os blocos que se sobrepõem a outro no mesmo dia.
export function conflitos(blocos) {
  const ids = new Set()
  for (let d = 0; d < 7; d++) {
    const lista = doDia(blocos, d).filter((b) => b.inicio)
    for (let i = 0; i < lista.length; i++) {
      for (let j = i + 1; j < lista.length && lista[j].inicio < lista[i].fim; j++) {
        ids.add(lista[i].id)
        ids.add(lista[j].id)
      }
    }
  }
  return ids
}

// Horas livres entre o último bloco do dia e o primeiro do dia seguinte (o sono possível).
// `seguinte` são os blocos do dia seguinte (do domingo, os da segunda do modelo).
export function sono(doDiaAtual, seguinte) {
  doDiaAtual = doDiaAtual.filter((b) => b.inicio)
  seguinte = seguinte.filter((b) => b.inicio)
  if (!doDiaAtual.length || !seguinte.length) return null
  const fim = Math.max(...doDiaAtual.map((b) => minutos(b.fim)))
  const comeco = Math.min(...seguinte.map((b) => minutos(b.inicio)))
  return { de: doDiaAtual.reduce((a, b) => (b.fim > a ? b.fim : a), '00:00'), ate: seguinte.reduce((a, b) => (b.inicio < a ? b.inicio : a), '24:00'), minutos: 24 * 60 - fim + comeco }
}

// Placar: quantas vezes cada atividade com meta está na semana e quantas foram feitas.
// Duas sessões no mesmo dia contam como duas.
export function placar(atividades, blocos) {
  return atividades
    .filter((a) => a.meta_max > 0)
    .map((a) => {
      const dela = blocos.filter((b) => b.atividade === a.id)
      const planejados = dela.length
      const feitos = dela.filter((b) => b.feito).length
      const tempo = dela.filter((b) => b.inicio).reduce((s, b) => s + minutos(b.fim) - minutos(b.inicio), 0)
      let estado = 'ok'
      // Fixa (o futebol de terça): se não aconteceu, não dá para remarcar, então não avisa.
      if (planejados < a.meta_min) estado = a.fixa ? 'fixa' : 'falta'
      else if (planejados > a.meta_max) estado = 'passou'
      return { atividade: a, planejados, feitos, tempo, estado, falta: estado === 'falta' ? a.meta_min - planejados : 0 }
    })
    .sort((x, y) => x.atividade.prioridade - y.atividade.prioridade || (x.estado === 'falta' ? -1 : 0) - (y.estado === 'falta' ? -1 : 0))
}

export function meta(a) {
  if (!a.meta_max) return ''
  return a.meta_min === a.meta_max ? `${a.meta_max}×` : `${a.meta_min}–${a.meta_max}×`
}

export function novoId(prefixo = 'b') {
  return `${prefixo}${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`
}

// "Programação" → "programacao"
export function idDoNome(nome, usados) {
  const base =
    nome
      .normalize('NFD')
      .replace(/[̀-ͯ]/g, '')
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '')
      .slice(0, 32) || 'atividade'
  let id = base
  for (let i = 2; usados.has(id); i++) id = `${base}-${i}`
  return id
}

// Os dias (0–6) que têm blocos se chocando: o "!" do dia.
export function diasComChoque(blocos) {
  const ids = conflitos(blocos)
  return new Set(blocos.filter((b) => ids.has(b.id)).map((b) => b.dia))
}

// O nome que aparece: o título do evento ou o nome da atividade.
export function nomeDoBloco(b, porId) {
  return b.titulo || porId[b.atividade]?.nome || 'Atividade apagada'
}

// semanaDe("2026-11-19") → { inicio: "2026-11-16", dia: 3 }
export function semanaDe(iso) {
  const dia = (data(iso).getUTCDay() + 6) % 7
  return { inicio: somarDias(iso, -dia), dia }
}

// As semanas (segundas) que cobrem um mês, para a grade do calendário. mes: "2026-11".
export function semanasDoMes(mes) {
  const primeiro = `${mes}-01`
  const d = data(primeiro)
  d.setUTCMonth(d.getUTCMonth() + 1, 0)
  const ultimo = d.toISOString().slice(0, 10)
  const semanas = []
  for (let s = semanaDe(primeiro).inicio; s <= ultimo; s = somarDias(s, 7)) semanas.push(s)
  return semanas
}

export function somarMeses(mes, n) {
  const d = data(`${mes}-01`)
  d.setUTCMonth(d.getUTCMonth() + n)
  return d.toISOString().slice(0, 7)
}

export function nomeDoMes(mes) {
  const t = data(`${mes}-01`).toLocaleDateString('pt-BR', { month: 'long', year: 'numeric', timeZone: 'UTC' })
  return t.charAt(0).toUpperCase() + t.slice(1)
}

// "qui., 19 de nov."
export function dataCurta(iso) {
  return data(iso).toLocaleDateString('pt-BR', { weekday: 'short', day: 'numeric', month: 'short', timeZone: 'UTC' })
}
