// Tema da interface: "auto" segue o aparelho; "claro" e "escuro" fixam. Guardado só neste
// aparelho (conveniência), e o app funciona sem o armazenamento. (Mesma lógica da Trilha.)
const CHAVE = 'hub.tema'
export const TEMAS = ['auto', 'claro', 'escuro']

export function lerTema() {
  try {
    const t = localStorage.getItem(CHAVE)
    return TEMAS.includes(t) ? t : 'auto'
  } catch {
    return 'auto'
  }
}

export function aplicarTema(tema) {
  const raiz = document.documentElement
  if (tema === 'auto') raiz.removeAttribute('data-tema')
  else raiz.setAttribute('data-tema', tema)
  try {
    localStorage.setItem(CHAVE, tema)
  } catch {
    // sem armazenamento, o tema vale até fechar
  }
}

export function proximoTema(tema) {
  return TEMAS[(TEMAS.indexOf(tema) + 1) % TEMAS.length]
}
