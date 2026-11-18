// Ditado pelo navegador (Web Speech API), em português, como no Acervo (D41). No Chrome do Android o
// áudio passa pelo serviço do Google. Sem suporte, o botão do microfone não aparece.
const Reconhecimento = typeof window !== 'undefined' ? window.SpeechRecognition || window.webkitSpeechRecognition : undefined

export const temVoz = !!Reconhecimento

const MENSAGENS = {
  'not-allowed': 'O navegador não deixou usar o microfone. Libere nas permissões do site.',
  'service-not-allowed': 'O navegador não deixou usar o ditado neste site.',
  'audio-capture': 'Nenhum microfone encontrado.',
  network: 'O ditado do navegador precisa de internet.',
}

// ouvir chama aoMudar(texto) enquanto a pessoa fala e aoTerminar(texto, erro) quando para. Devolve a
// função que para de ouvir.
export function ouvir({ aoMudar, aoTerminar }) {
  const r = new Reconhecimento()
  r.lang = 'pt-BR'
  r.continuous = true
  r.interimResults = true
  let final = ''
  let ultimo = ''
  let erro = ''
  r.onresult = (ev) => {
    let provisorio = ''
    for (let i = ev.resultIndex; i < ev.results.length; i++) {
      const trecho = ev.results[i][0].transcript
      if (ev.results[i].isFinal) final += trecho
      else provisorio += trecho
    }
    ultimo = `${final} ${provisorio}`.replace(/\s+/g, ' ').trim()
    aoMudar(ultimo)
  }
  r.onerror = (ev) => {
    erro = MENSAGENS[ev.error] ?? (ev.error === 'no-speech' || ev.error === 'aborted' ? '' : `O ditado parou (${ev.error}).`)
  }
  r.onend = () => aoTerminar(ultimo, erro)
  r.start()
  return () => r.stop()
}
