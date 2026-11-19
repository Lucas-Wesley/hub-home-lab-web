// Arrastar itens de lista e de checklist para mudar a ordem, como no Notion: uma alça (⋮⋮) aparece à
// esquerda do item sob o mouse (no celular, do item onde está o cursor). Segurar e arrastar move o
// item, com o que estiver recuado dentro dele. Pelo teclado: Alt+Shift+↑/↓.
import { Extension } from '@tiptap/core'
import { Plugin, PluginKey, TextSelection } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

// O item sendo arrastado fica apagado por uma decoração (uma classe direto no DOM o editor desfaz).
const chave = new PluginKey('arrastarItens')

const ITENS = ['listItem', 'taskItem']
const ehItem = (no) => ITENS.includes(no.type.name)

// O item de lista mais de dentro que contém a posição: { pos, no } ou null.
export function itemEm(doc, pos) {
  const $p = doc.resolve(pos)
  for (let d = $p.depth; d > 0; d--) {
    if (ehItem($p.node(d))) return { pos: $p.before(d), no: $p.node(d) }
  }
  return null
}

// Move o item que começa em "de" para antes ou depois do item que começa em "alvo". Só entre itens do
// mesmo tipo (lista com lista, checklist com checklist), e nunca para dentro dele mesmo. Devolve a
// transação, ou null se não há o que mover.
export function moverItem(state, de, alvo, lado) {
  const { doc } = state
  const no = doc.nodeAt(de)
  const destino = doc.nodeAt(alvo)
  if (!no || !destino || !ehItem(no) || destino.type !== no.type) return null
  if (alvo >= de && alvo < de + no.nodeSize) return null
  const onde = lado === 'antes' ? alvo : alvo + destino.nodeSize
  if (onde === de || onde === de + no.nodeSize) return null

  // Sendo o único item da lista, sai a lista inteira (uma lista vazia não existe).
  const $de = doc.resolve(de)
  const sozinho = $de.parent.childCount === 1
  const tr = state.tr
  if (sozinho) tr.delete($de.before(), $de.after())
  else tr.delete(de, de + no.nodeSize)
  const novo = tr.mapping.map(onde, lado === 'antes' ? -1 : 1)
  tr.insert(novo, no)
  return { tr, pos: novo }
}

function moverPeloTeclado(editor, passo) {
  const { state } = editor
  const { $from, from } = state.selection
  const item = itemEm(state.doc, from)
  if (!item) return false
  const $item = state.doc.resolve(item.pos)
  const indice = $item.index()
  const lista = $item.parent
  const vizinho = indice + passo
  if (vizinho < 0 || vizinho >= lista.childCount) return true // já está na ponta: não faz nada
  const alvo = passo < 0 ? item.pos - lista.child(vizinho).nodeSize : item.pos + item.no.nodeSize
  const r = moverItem(state, item.pos, alvo, passo < 0 ? 'antes' : 'depois')
  if (!r) return true
  const desvio = $from.pos - item.pos
  r.tr.setSelection(TextSelection.create(r.tr.doc, r.pos + desvio))
  editor.view.dispatch(r.tr.scrollIntoView())
  return true
}

// A primeira linha do item (o parágrafo), que é onde a alça se alinha e o que conta para o alvo.
const linhaDo = (li) => li.querySelector(':scope > p, :scope > div > p') ?? li

function alcaView(view) {
  // O editor nasce fora da tela e só depois o Vue o põe na página: a caixa é buscada na hora.
  const caixa = () => {
    const c = view.dom.parentElement
    if (c && alca.parentElement !== c) c.append(alca, linha)
    return c
  }
  const alca = document.createElement('button')
  alca.type = 'button'
  alca.className = 'alca-item'
  alca.setAttribute('aria-label', 'Arrastar para mudar a ordem')
  alca.title = 'Arrastar para mudar a ordem (Alt+Shift+↑/↓)'
  alca.tabIndex = -1
  alca.contentEditable = 'false'
  alca.innerHTML = '<svg viewBox="0 0 10 16" aria-hidden="true"><circle cx="2.5" cy="3" r="1.5"/><circle cx="7.5" cy="3" r="1.5"/><circle cx="2.5" cy="8" r="1.5"/><circle cx="7.5" cy="8" r="1.5"/><circle cx="2.5" cy="13" r="1.5"/><circle cx="7.5" cy="13" r="1.5"/></svg>'
  const linha = document.createElement('div')
  linha.className = 'linha-soltar'

  const toque = window.matchMedia('(pointer: coarse)')
  let atual = null // o <li> com a alça
  let arraste = null // { li, pos, alvo, lado }

  const esconder = () => {
    atual = null
    alca.classList.remove('visivel')
  }

  function mostrar(li) {
    if (!li || !view.editable) return esconder()
    atual = li
    const c = caixa().getBoundingClientRect()
    const l = linhaDo(li).getBoundingClientRect()
    // Na lista com pontinhos, a alça fica antes do pontinho (que está no recuo da <ul>).
    const esquerda = li.hasAttribute('data-checked') ? li.getBoundingClientRect().left : li.parentElement.getBoundingClientRect().left
    const altura = parseFloat(getComputedStyle(linhaDo(li)).lineHeight) || 24
    alca.style.left = `${Math.max(2 - c.left, esquerda - c.left - alca.offsetWidth - 2)}px`
    alca.style.top = `${l.top - c.top + Math.min(l.height, altura + 4) / 2 - alca.offsetHeight / 2}px`
    alca.classList.add('visivel')
  }

  const liDe = (el) => {
    const li = el?.closest?.('li')
    return li && view.dom.contains(li) ? li : null
  }

  function posDo(li) {
    try {
      const pos = view.posAtDOM(linhaDo(li), 0)
      return itemEm(view.state.doc, pos)
    } catch {
      return null
    }
  }

  // No celular não há "passar o mouse": a alça segue o item onde está o cursor.
  function pelaSelecao() {
    if (!toque.matches || arraste) return
    if (!view.hasFocus()) return esconder()
    const { node } = view.domAtPos(view.state.selection.from)
    mostrar(liDe(node.nodeType === 3 ? node.parentElement : node))
  }

  function aoMover(e) {
    if (arraste || toque.matches || e.pointerType === 'touch') return
    if (e.target === alca || alca.contains(e.target)) return
    const li = liDe(e.target)
    if (li !== atual) mostrar(li)
  }

  // Durante o arraste: o item mais perto do dedo, do mesmo tipo e fora do que está sendo movido.
  function alvoEm(y) {
    const fonte = view.nodeDOM(arraste.pos)
    if (!fonte) return null
    const tipo = fonte.hasAttribute('data-checked')
    let melhor = null
    for (const li of view.dom.querySelectorAll('li')) {
      if (li.hasAttribute('data-checked') !== tipo || fonte.contains(li)) continue
      if (li.closest('[data-type="details"]:not(.is-open) [data-type="detailsContent"]')) continue
      const r = linhaDo(li).getBoundingClientRect()
      const dist = y < r.top ? r.top - y : y > r.bottom ? y - r.bottom : 0
      if (!melhor || dist < melhor.dist) melhor = { li, dist, lado: y < r.top + r.height / 2 ? 'antes' : 'depois' }
    }
    return melhor
  }

  function desenharLinha(a) {
    if (!a) return linha.classList.remove('visivel')
    const c = caixa().getBoundingClientRect()
    const r = a.li.getBoundingClientRect()
    linha.style.left = `${r.left - c.left}px`
    linha.style.width = `${r.width}px`
    linha.style.top = `${(a.lado === 'antes' ? r.top : r.bottom) - c.top - 1}px`
    linha.classList.add('visivel')
  }

  function comecar(e) {
    if (!atual || !view.editable) return
    e.preventDefault()
    const item = posDo(atual)
    if (!item) return
    alca.setPointerCapture(e.pointerId)
    arraste = { pos: item.pos, alvo: null, inicioY: e.clientY, andou: false }
  }

  function arrastar(e) {
    if (!arraste) return
    if (!arraste.andou && Math.abs(e.clientY - arraste.inicioY) < 4) return
    if (!arraste.andou) view.dispatch(view.state.tr.setMeta(chave, arraste.pos))
    arraste.andou = true
    document.body.classList.add('arrastando-item')
    if (e.clientY < 48) window.scrollBy(0, -12)
    else if (e.clientY > window.innerHeight - 48) window.scrollBy(0, 12)
    arraste.alvo = alvoEm(e.clientY)
    desenharLinha(arraste.alvo)
  }

  function soltar() {
    if (!arraste) return
    const { pos, alvo, andou } = arraste
    arraste = null
    document.body.classList.remove('arrastando-item')
    linha.classList.remove('visivel')
    if (andou) view.dispatch(view.state.tr.setMeta(chave, null))
    if (!andou || !alvo) return
    const destino = posDo(alvo.li)
    if (!destino) return
    const r = moverItem(view.state, pos, destino.pos, alvo.lado)
    if (!r) return
    r.tr.setSelection(TextSelection.create(r.tr.doc, Math.min(r.pos + 2, r.tr.doc.content.size)))
    view.dispatch(r.tr)
    view.focus()
  }

  function cancelar() {
    if (!arraste) return
    arraste.alvo = null
    soltar()
  }

  const sair = (e) => {
    if (!arraste && e.relatedTarget !== alca && !alca.contains(e.relatedTarget) && !view.dom.contains(e.relatedTarget)) esconder()
  }

  view.dom.addEventListener('pointermove', aoMover)
  view.dom.addEventListener('pointerleave', sair)
  alca.addEventListener('pointerleave', sair)
  alca.addEventListener('pointerdown', comecar)
  alca.addEventListener('pointermove', arrastar)
  alca.addEventListener('pointerup', soltar)
  alca.addEventListener('pointercancel', cancelar)
  alca.addEventListener('mousedown', (e) => e.preventDefault())

  return {
    update() {
      if (arraste) return
      if (toque.matches) return pelaSelecao()
      if (atual && !atual.isConnected) esconder()
      else if (atual) mostrar(atual)
    },
    destroy() {
      view.dom.removeEventListener('pointermove', aoMover)
      view.dom.removeEventListener('pointerleave', sair)
      alca.remove()
      linha.remove()
    },
  }
}

export const ArrastarItens = Extension.create({
  name: 'arrastarItens',
  addKeyboardShortcuts() {
    return {
      'Alt-Shift-ArrowUp': () => moverPeloTeclado(this.editor, -1),
      'Alt-Shift-ArrowDown': () => moverPeloTeclado(this.editor, 1),
    }
  },
  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: chave,
        state: {
          init: () => null,
          apply: (tr, pos) => (tr.getMeta(chave) !== undefined ? tr.getMeta(chave) : pos === null ? null : tr.mapping.map(pos)),
        },
        props: {
          decorations(state) {
            const pos = chave.getState(state)
            const no = pos === null ? null : state.doc.nodeAt(pos)
            return no ? DecorationSet.create(state.doc, [Decoration.node(pos, pos + no.nodeSize, { class: 'arrastando' })]) : null
          },
        },
        view: alcaView,
      }),
    ]
  },
})
