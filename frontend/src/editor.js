// O editor da página Início: só texto, lista com pontinhos, checklist e toggle, como no Notion.
import { Extension, InputRule, mergeAttributes } from '@tiptap/core'
import { Details, DetailsContent, DetailsSummary } from '@tiptap/extension-details'
import { TaskItem, TaskList } from '@tiptap/extension-list'
import { Placeholder } from '@tiptap/extensions'
import { TextSelection } from '@tiptap/pm/state'
import StarterKit from '@tiptap/starter-kit'

// Troca o parágrafo do cursor por um toggle aberto: o texto vira o título e o cursor fica nele.
// Texto e título são as linhas "soltas": viram toggle, recebem o menu do "/".
export const linhaSolta = (no) => no.type.name === 'paragraph' || no.type.name === 'heading'

function podeVirarToggle($from) {
  if (!linhaSolta($from.parent)) return false
  const indice = $from.index(-1)
  return $from.node(-1).canReplaceWith(indice, indice + 1, $from.doc.type.schema.nodes.details)
}

function trocarPorToggle(tr, $from) {
  if (!podeVirarToggle($from)) return false
  const paragrafo = $from.parent
  const { schema } = tr.doc.type
  const texto = paragrafo.textContent
  const toggle = schema.nodes.details.create({ open: true }, [
    schema.nodes.detailsSummary.create(null, texto ? schema.text(texto) : null),
    schema.nodes.detailsContent.create(null, schema.nodes.paragraph.create()),
  ])
  const inicio = $from.before()
  tr.replaceWith(inicio, inicio + paragrafo.nodeSize, toggle)
  tr.setSelection(TextSelection.create(tr.doc, inicio + 2 + texto.length))
  return true
}

// O desenho do toggle é nosso: o aberto/fechado vem só do atributo "open" (que é salvo), e o CSS
// esconde o conteúdo. O do Tiptap guarda o estado também no DOM e às vezes nasce fechado.
const ConteudoDoToggle = DetailsContent.extend({
  addNodeView: null,
  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-type': 'detailsContent' }), 0]
  },
})

const Toggle = Details.extend({
  addNodeView() {
    return ({ editor, getPos, node }) => {
      const dom = document.createElement('div')
      dom.dataset.type = 'details'
      const botao = document.createElement('button')
      botao.type = 'button'
      botao.contentEditable = 'false'
      const conteudo = document.createElement('div')
      dom.append(botao, conteudo)
      const pintar = (aberto) => {
        dom.classList.toggle('is-open', aberto)
        botao.setAttribute('aria-label', aberto ? 'Fechar o toggle' : 'Abrir o toggle')
        botao.setAttribute('aria-expanded', String(aberto))
      }
      pintar(node.attrs.open)
      botao.addEventListener('mousedown', (e) => e.preventDefault())
      botao.addEventListener('click', () => {
        const pos = getPos()
        if (typeof pos !== 'number') return
        const atual = editor.state.doc.nodeAt(pos)
        if (!editor.isEditable) return pintar(!dom.classList.contains('is-open'))
        editor.view.dispatch(editor.state.tr.setNodeMarkup(pos, undefined, { ...atual.attrs, open: !atual.attrs.open }))
      })
      return {
        dom,
        contentDOM: conteudo,
        ignoreMutation: (m) => m.type !== 'selection' && (botao.contains(m.target) || !dom.contains(m.target) || dom === m.target),
        update: (novo) => {
          if (novo.type !== node.type) return false
          pintar(novo.attrs.open)
          return true
        },
      }
    }
  },
  addCommands() {
    return {
      ...this.parent?.(),
      viraToggle:
        () =>
        ({ tr, dispatch }) => {
          if (!dispatch) return podeVirarToggle(tr.selection.$from)
          return trocarPorToggle(tr, tr.selection.$from)
        },
    }
  },
  // "> " no começo da linha cria o toggle, como no Notion.
  addInputRules() {
    return [
      new InputRule({
        find: /^>\s$/,
        handler: ({ state, range }) => {
          if (!podeVirarToggle(state.doc.resolve(range.from))) return null
          const { tr } = state
          tr.delete(range.from, range.to)
          trocarPorToggle(tr, tr.doc.resolve(range.from))
        },
      }),
    ]
  },
  // Enter no título vai para a primeira linha de dentro, se ela estiver vazia, em vez de criar outra.
  addKeyboardShortcuts() {
    const doTiptap = this.parent?.() ?? {}
    return {
      ...doTiptap,
      Enter: (props) => {
        const { empty, $head } = props.editor.state.selection
        if (empty && $head.parent.type.name === 'detailsSummary') {
          const toggle = $head.node(-1)
          const primeira = toggle.child(1).firstChild
          if (toggle.attrs.open && primeira?.type.name === 'paragraph' && primeira.content.size === 0) {
            return props.editor.commands.setTextSelection($head.after() + 2)
          }
        }
        return doTiptap.Enter ? doTiptap.Enter(props) : false
      },
    }
  },
})

// Dentro de uma lista, "[] " e "- " trocam o tipo só da linha atual, como no Notion.
const TrocaDeLista = Extension.create({
  name: 'trocaDeLista',
  addInputRules() {
    // Tira a linha da lista e põe na outra, em passos separados (juntos, a lista inteira mudava).
    const trocar = (find, de, para) =>
      new InputRule({
        find,
        handler: ({ state, range }) => {
          if (state.doc.resolve(range.from).node(-1)?.type.name !== de) return null
          state.tr.delete(range.from, range.to)
          queueMicrotask(() => {
            this.editor.chain().liftListItem(de).run()
            this.editor.chain()[para]().run()
          })
        },
      })
    return [trocar(/^\[( |x)?\]\s$/, 'listItem', 'toggleTaskList'), trocar(/^[-*+]\s$/, 'taskItem', 'toggleBulletList')]
  },
})

// Backspace no começo de um título volta a ser texto, como no Notion.
const TituloVoltaATexto = Extension.create({
  name: 'tituloVoltaATexto',
  addKeyboardShortcuts() {
    return {
      Backspace: ({ editor }) => {
        const { empty, $from } = editor.state.selection
        if (!empty || $from.parent.type.name !== 'heading' || $from.parentOffset !== 0) return false
        return editor.commands.setParagraph()
      },
    }
  },
})

// Enquanto o menu do "/" está aberto, as setas, o Enter e o Esc são dele.
export function teclasDoMenu(menu) {
  return Extension.create({
    name: 'teclasDoMenu',
    priority: 1000,
    addKeyboardShortcuts() {
      const se = (acao) => () => (menu.aberto() ? (acao(), true) : false)
      return {
        ArrowDown: se(() => menu.mover(1)),
        ArrowUp: se(() => menu.mover(-1)),
        Enter: se(() => menu.escolher()),
        Tab: se(() => menu.escolher()),
        Escape: se(() => menu.fechar()),
      }
    },
  })
}

export function extensoes(menu) {
  return [
    StarterKit.configure({
      heading: { levels: [1] }, // um nível só; "# " no começo da linha
      blockquote: false,
      codeBlock: false,
      code: false,
      orderedList: false,
      horizontalRule: false,
      link: false,
    }),
    TaskList,
    TaskItem.configure({ nested: true, a11y: { checkboxLabel: (node) => `Marcar ${node.textContent || 'item vazio'}` } }),
    Toggle.configure({ persist: true }),
    DetailsSummary,
    ConteudoDoToggle,
    Placeholder.configure({
      includeChildren: true,
      placeholder: ({ node }) =>
        ({ detailsSummary: 'Toggle', heading: 'Título' })[node.type.name] ?? 'Escreva, ou digite / para escolher um bloco',
    }),
    TrocaDeLista,
    TituloVoltaATexto,
    teclasDoMenu(menu),
  ]
}

// O que o menu do "/" oferece. Os atalhos são os mesmos do Notion.
export const BLOCOS = [
  { id: 'texto', nome: 'Texto', atalho: '' },
  { id: 'titulo', nome: 'Título', atalho: '#' },
  { id: 'lista', nome: 'Lista', atalho: '-' },
  { id: 'checklist', nome: 'Checklist', atalho: '[]' },
  { id: 'toggle', nome: 'Toggle', atalho: '>' },
]

const semAcento = (s) => s.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase()

export function filtrarBlocos(consulta) {
  const q = semAcento(consulta)
  return BLOCOS.filter((b) => semAcento(b.nome).includes(q))
}

// Transforma o bloco do cursor no tipo escolhido.
export function aplicarBloco(editor, id) {
  const sairDaLista = (c) => {
    if (editor.isActive('taskList')) c = c.toggleTaskList()
    else if (editor.isActive('bulletList')) c = c.toggleBulletList()
    return c
  }
  if (id === 'texto') sairDaLista(editor.chain().focus()).setParagraph().run()
  else if (id === 'titulo') {
    sairDaLista(editor.chain().focus()).run()
    editor.chain().focus().setHeading({ level: 1 }).run()
  } else if (id === 'lista' && !editor.isActive('bulletList')) {
    sairDaLista(editor.chain().focus()).setParagraph().run()
    editor.chain().focus().toggleBulletList().run()
  } else if (id === 'checklist' && !editor.isActive('taskList')) {
    sairDaLista(editor.chain().focus()).setParagraph().run()
    editor.chain().focus().toggleTaskList().run()
  }
  else if (id === 'toggle') {
    sairDaLista(editor.chain().focus()).run()
    editor.chain().focus().viraToggle().run()
  }
}
