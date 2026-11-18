import { describe, expect, it } from 'vitest'
import { conflitos, diaDoMes, duracao, idDoNome, placar, rotuloSemana, somarDias, sono } from './rotina.js'

const b = (id, dia, inicio, fim, atividade = 'x', feito = false) => ({ id, dia, inicio, fim, atividade, feito })

describe('rotina', () => {
  it('conta os dias e nomeia a semana', () => {
    expect(somarDias('2026-11-30', 6)).toBe('2026-12-06')
    expect(diaDoMes('2026-11-16', 4)).toBe(20)
    expect(rotuloSemana('2026-11-16')).toBe('16 – 22 de nov.')
    expect(rotuloSemana('2026-11-30')).toBe('30 de nov. – 6 de dez.')
  })

  it('acha blocos sobrepostos só no mesmo dia', () => {
    const ids = conflitos([b('a', 0, '07:00', '08:00'), b('b', 0, '07:30', '09:00'), b('c', 0, '09:00', '10:00'), b('d', 1, '07:30', '08:00')])
    expect([...ids].sort()).toEqual(['a', 'b'])
  })

  it('calcula o sono possível até o dia seguinte', () => {
    expect(sono([b('a', 0, '21:30', '23:00')], [b('b', 1, '07:00', '08:00')])).toEqual({ de: '23:00', ate: '07:00', minutos: 480 })
    expect(sono([], [b('b', 1, '07:00', '08:00')])).toBeNull()
  })

  it('monta o placar das metas', () => {
    const atividades = [
      { id: 'prog', nome: 'Programação', meta_min: 3, meta_max: 4, prioridade: 1 },
      { id: 'ing', nome: 'Inglês', meta_min: 1, meta_max: 1, prioridade: 1 },
      { id: 'trab', nome: 'Trabalho', meta_min: 0, meta_max: 0, prioridade: 1 },
    ]
    const p = placar(atividades, [b('1', 0, '21:30', '23:00', 'prog', true), b('2', 2, '12:30', '14:00', 'prog'), b('3', 0, '07:00', '08:00', 'ing'), b('4', 1, '07:00', '08:00', 'ing')])
    expect(p.map((x) => [x.atividade.id, x.planejados, x.feitos, x.estado])).toEqual([
      ['prog', 2, 1, 'falta'],
      ['ing', 2, 0, 'passou'],
    ])
    expect(p[0].tempo).toBe(180)
    expect(duracao(p[0].tempo)).toBe('3 h')
    expect(duracao(90)).toBe('1 h 30')
  })

  it('cria um identificador a partir do nome', () => {
    expect(idDoNome('Programação', new Set())).toBe('programacao')
    expect(idDoNome('Inglês', new Set(['ingles']))).toBe('ingles-2')
  })
})
