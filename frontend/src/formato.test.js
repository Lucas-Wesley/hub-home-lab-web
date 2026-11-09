import { describe, expect, it } from 'vitest'
import { bytes, dataHora, duracao, porcento } from './formato.js'

describe('bytes', () => {
  it('usa base 2 e uma casa só abaixo de 10', () => {
    expect(bytes(0)).toBe('0 B')
    expect(bytes(512)).toBe('512 B')
    expect(bytes(7.2 * 1024 * 1024)).toBe('7,2 MiB')
    expect(bytes(14_642_571_481)).toBe('14 GiB')
    expect(bytes(99 * 1024 ** 3)).toBe('99 GiB')
    expect(bytes(1.5 * 1024 ** 4)).toBe('1,5 TiB')
  })
})

describe('porcento', () => {
  it('arredonda e aguenta total zero', () => {
    expect(porcento(19, 99)).toBe(19)
    expect(porcento(1, 0)).toBe(0)
  })
})

describe('duracao', () => {
  it('escolhe a unidade maior', () => {
    expect(duracao(500000)).toBe('5 dias')
    expect(duracao(86400)).toBe('1 dia')
    expect(duracao(18000)).toBe('5 h')
    expect(duracao(30)).toBe('1 min')
  })
})

describe('dataHora', () => {
  it('mostra no horário de Brasília', () => {
    expect(dataHora('2026-11-10T07:00:00Z')).toMatch(/10 de nov\..*04:00/)
    expect(dataHora('')).toBe('')
  })
})
