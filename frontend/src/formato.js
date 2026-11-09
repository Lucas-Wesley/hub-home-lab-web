// Números do painel em português. Tamanhos em base 2 (GiB), como o Proxmox e o df -h mostram.
const UNIDADES = ['B', 'KiB', 'MiB', 'GiB', 'TiB']

export function bytes(n) {
  if (!n || n < 0) return '0 B'
  let i = 0
  let v = n
  while (v >= 1024 && i < UNIDADES.length - 1) {
    v /= 1024
    i++
  }
  const casas = i === 0 || v >= 10 ? 0 : 1
  return `${v.toLocaleString('pt-BR', { maximumFractionDigits: casas })} ${UNIDADES[i]}`
}

export function porcento(parte, total) {
  if (!total) return 0
  return Math.round((parte / total) * 100)
}

// duracao: "3 dias", "5 h", "12 min".
export function duracao(segundos) {
  const dias = Math.floor(segundos / 86400)
  if (dias >= 1) return `${dias} ${dias === 1 ? 'dia' : 'dias'}`
  const horas = Math.floor(segundos / 3600)
  if (horas >= 1) return `${horas} h`
  return `${Math.max(1, Math.floor(segundos / 60))} min`
}

// dataHora: "ter., 10 de nov., 04:00".
export function dataHora(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const dia = d.toLocaleDateString('pt-BR', { weekday: 'short', day: 'numeric', month: 'short', timeZone: 'America/Sao_Paulo' })
  const hora = d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit', timeZone: 'America/Sao_Paulo' })
  return `${dia}, ${hora}`
}

export const FREQUENCIAS = [
  { dias: 1, nome: 'Todo dia' },
  { dias: 2, nome: 'A cada 2 dias' },
  { dias: 3, nome: 'A cada 3 dias' },
  { dias: 7, nome: 'Toda semana' },
  { dias: 14, nome: 'A cada 2 semanas' },
  { dias: 30, nome: 'Todo mês' },
]

// Acima disto, a barra fica vermelha e o painel avisa.
export const QUASE_CHEIO = 0.85
