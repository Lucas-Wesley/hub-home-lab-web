<script setup>
import { EditorContent, useEditor } from '@tiptap/vue-3'
import { computed, inject, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { api } from './api.js'
import { aplicarBloco, extensoes, filtrarBlocos, linhaSolta } from './editor.js'

// Início: a página inteira é o campo de texto, como no Notion. Salva sozinha, pouco depois de parar de digitar.
const recado = inject('recado')
const estado = ref('carregando') // carregando, pronto, erro

// ---------- Menu do "/" ----------
const menu = reactive({ aberto: false, consulta: '', de: 0, ate: 0, indice: 0, x: 0, y: 0 })
const opcoes = computed(() => filtrarBlocos(menu.consulta))

function fecharMenu() {
  menu.aberto = false
}

function escolher(bloco = opcoes.value[menu.indice]) {
  if (!bloco) return
  editor.value.chain().focus().deleteRange({ from: menu.de, to: menu.ate }).run()
  fecharMenu()
  aplicarBloco(editor.value, bloco.id)
}

// Abre o menu quando o texto antes do cursor termina em "/palavra" (no começo ou depois de um espaço).
function conferirBarra(ed) {
  const { empty, $from } = ed.state.selection
  if (!empty || !linhaSolta($from.parent)) return fecharMenu()
  const antes = $from.parent.textBetween(0, $from.parentOffset, undefined, '￼')
  const achado = antes.match(/(?:^|\s)\/([\p{L}]{0,12})$/u)
  if (!achado || !filtrarBlocos(achado[1]).length) return fecharMenu()
  const de = $from.pos - achado[1].length - 1
  if (!menu.aberto || menu.consulta !== achado[1]) menu.indice = 0
  const pos = ed.view.coordsAtPos(de)
  Object.assign(menu, {
    aberto: true,
    consulta: achado[1],
    de,
    ate: $from.pos,
    x: Math.max(8, Math.min(pos.left, window.innerWidth - 228)),
    y: pos.bottom + 6,
  })
}

const teclas = {
  aberto: () => menu.aberto,
  mover: (passo) => (menu.indice = (menu.indice + passo + opcoes.value.length) % opcoes.value.length),
  escolher: () => escolher(),
  fechar: fecharMenu,
}

// ---------- Salvar ----------
// "base" é a versão da página que este aparelho tem. O servidor só aceita salvar por cima dela: se a
// página mudou em outro aparelho (ou numa aba esquecida aberta), ele responde 409 e a versão nova
// vem para cá, em vez de a velha apagar o que foi escrito lá.
let base = ''
let salvo = '' // o documento como está no servidor, para não salvar o que não mudou
let espera = null
let salvando = null
let mudou = false
let avisou = false

function aplicar(pagina) {
  base = pagina.atualizada_em ?? ''
  if (pagina.conteudo) editor.value.commands.setContent(pagina.conteudo, { emitUpdate: false })
  salvo = JSON.stringify(editor.value.getJSON())
}

// O editor às vezes acerta o documento sozinho (ao abrir, ao focar) sem mudar o texto: isso não conta.
function semMudanca() {
  if (JSON.stringify(editor.value.getJSON()) !== salvo) return false
  mudou = false
  return true
}

function chegouOutra(pagina) {
  mudou = false
  clearTimeout(espera)
  aplicar(pagina)
  recado('A página tinha mudado em outro aparelho. Abri a versão mais nova; o que você digitou por último aqui não foi salvo.', true)
}

function corpoDoSalvar() {
  const conteudo = editor.value.getJSON()
  return { conteudo, base, texto: JSON.stringify(conteudo) }
}

async function salvar() {
  clearTimeout(espera)
  if (!mudou || !editor.value) return
  if (salvando) return salvando.then(salvar)
  if (semMudanca()) return
  mudou = false
  const { conteudo, texto } = corpoDoSalvar()
  salvando = api('/api/inicio', { metodo: 'PUT', corpo: { conteudo, base } })
    .then((p) => {
      base = p.atualizada_em
      salvo = texto
      avisou = false
    })
    .catch((e) => {
      if (e.status === 409 && e.dados) return chegouOutra(e.dados)
      mudou = true
      if (!avisou) recado(`Não salvou: ${e.message}`, true)
      avisou = true
    })
    .finally(() => (salvando = null))
  return salvando
}

// Ao sair do app (ou trocar de aba), manda o que faltou mesmo com a página fechando. Ao voltar, busca
// a página de novo: pode ter sido mudada em outro aparelho enquanto esta ficou parada.
function aoMudarVisibilidade() {
  if (!editor.value || estado.value !== 'pronto') return
  if (document.visibilityState === 'visible') return conferirSeMudou()
  if (!mudou || semMudanca()) return
  mudou = false
  const { conteudo, texto } = corpoDoSalvar()
  fetch('/api/inicio', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ conteudo, base }),
    keepalive: true,
  })
    .then(async (r) => {
      if (r.ok) {
        base = (await r.json()).atualizada_em
        salvo = texto
      }
      else if (r.status === 409) chegouOutra(await r.json())
      else mudou = true
    })
    .catch(() => (mudou = true))
}

async function conferirSeMudou() {
  if (mudou || salvando) return
  try {
    const pagina = await api('/api/inicio')
    if (!mudou && !salvando && (pagina.atualizada_em ?? '') !== base) aplicar(pagina)
  } catch {
    // Sem conexão: fica a versão que já está na tela.
  }
}

const editor = useEditor({
  extensions: extensoes(teclas),
  editable: false,
  editorProps: { attributes: { 'aria-label': 'Início', spellcheck: 'true' } },
  onUpdate: ({ editor: ed }) => {
    conferirBarra(ed)
    mudou = true
    clearTimeout(espera)
    espera = setTimeout(salvar, 700)
  },
  onSelectionUpdate: ({ editor: ed }) => conferirBarra(ed),
  onBlur: () => setTimeout(fecharMenu, 150),
})

async function carregar() {
  estado.value = 'carregando'
  try {
    const pagina = await api('/api/inicio')
    // Espera o editor existir (o useEditor cria no onMounted).
    while (!editor.value) await new Promise((r) => setTimeout(r, 10))
    aplicar(pagina)
    editor.value.setEditable(true)
    estado.value = 'pronto'
    if (!pagina.conteudo) editor.value.commands.focus('end')
  } catch (e) {
    estado.value = 'erro'
    recado(e.message, true)
  }
}

// Tocar no espaço vazio abaixo do texto leva o cursor para o fim, como se a página toda fosse o campo.
function focarNoFim(evento) {
  if (estado.value !== 'pronto' || evento.target.closest('.ProseMirror')) return
  editor.value.commands.focus('end')
}

onMounted(() => {
  carregar()
  document.addEventListener('visibilitychange', aoMudarVisibilidade)
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', aoMudarVisibilidade)
  salvar()
})
</script>

<template>
  <div class="pagina inicio" @mousedown="focarNoFim">
    <p v-if="estado === 'erro'" class="vazio">
      Não deu para abrir a página.<br />
      <button class="botao" style="margin-top: 12px" @click="carregar">Tentar de novo</button>
    </p>
    <EditorContent v-show="estado !== 'erro'" :editor="editor" class="folha" />

    <div
      v-if="menu.aberto"
      class="menu-blocos"
      role="listbox"
      aria-label="Blocos"
      :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
      @mousedown.prevent
    >
      <button
        v-for="(bloco, i) in opcoes"
        :key="bloco.id"
        class="opcao"
        role="option"
        :aria-selected="i === menu.indice"
        @mouseenter="menu.indice = i"
        @click="escolher(bloco)"
      >
        <span>{{ bloco.nome }}</span>
        <kbd v-if="bloco.atalho">{{ bloco.atalho }}</kbd>
      </button>
    </div>
  </div>
</template>

<style scoped>
.inicio {
  min-height: 100vh;
  cursor: text;
}

.folha :deep(.ProseMirror) {
  min-height: calc(100vh - 140px);
  outline: none;
  font-size: 16px;
  line-height: 1.7;
  word-wrap: break-word;
  white-space: pre-wrap;
}

.folha :deep(p) {
  margin: 0;
  padding: 2px 0;
}

/* Título: um nível só, para separar os assuntos da página. */
.folha :deep(h1) {
  margin: 0;
  padding: 18px 0 4px;
  font-family: var(--fonte-titulo);
  font-size: 23px;
  font-weight: 500;
  line-height: 1.3;
}

.folha :deep(.ProseMirror > h1:first-child) {
  padding-top: 2px;
}

/* O texto fraco de exemplo aparece só na linha vazia onde está o cursor. */
.folha :deep(.is-empty::before) {
  content: attr(data-placeholder);
  float: left;
  height: 0;
  color: var(--texto-fraco);
  pointer-events: none;
}

/* Lista com pontinhos. */
.folha :deep(ul:not([data-type])) {
  margin: 0;
  padding-left: 1.6em;
}

.folha :deep(li::marker) {
  color: var(--texto-suave);
}

/* Checklist: a caixinha à esquerda; marcada, o texto fica riscado. */
.folha :deep(ul[data-type='taskList']) {
  margin: 0;
  padding: 0;
  list-style: none;
}

.folha :deep(li[data-checked]) {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.folha :deep(li[data-checked] > label) {
  flex: none;
  display: flex;
  align-items: center;
  height: calc(1.6em + 4px);
  user-select: none;
}

.folha :deep(li[data-checked] > div) {
  flex: 1;
  min-width: 0;
}

.folha :deep(input[type='checkbox']) {
  width: 16px;
  height: 16px;
  margin: 0;
  accent-color: var(--link);
  cursor: pointer;
}

.folha :deep(li[data-checked='true'] > div > p) {
  color: var(--texto-fraco);
  text-decoration: line-through;
}

/* Toggle: o triângulo abre e fecha; o que está dentro fica recuado. */
.folha :deep([data-type='details']) {
  display: flex;
  align-items: flex-start;
  gap: 2px;
}

.folha :deep([data-type='details'] > button) {
  flex: none;
  display: grid;
  place-items: center;
  width: 24px;
  height: calc(1.6em + 4px);
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--texto-suave);
  cursor: pointer;
}

.folha :deep([data-type='details'] > button::before) {
  content: '';
  border-top: 5px solid transparent;
  border-bottom: 5px solid transparent;
  border-left: 7px solid currentColor;
  transition: transform 0.15s ease;
}

.folha :deep([data-type='details'].is-open > button::before) {
  transform: rotate(90deg);
}

.folha :deep([data-type='details'] > button:hover) {
  background: var(--superficie);
  color: var(--texto);
}

.folha :deep([data-type='details'] > div) {
  flex: 1;
  min-width: 0;
}

.folha :deep(summary) {
  display: block;
  padding: 2px 0;
  list-style: none;
  cursor: text;
}

.folha :deep(summary::-webkit-details-marker) {
  display: none;
}

.folha :deep([data-type='details']:not(.is-open) > div > [data-type='detailsContent']) {
  display: none;
}

/* Alça para arrastar item de lista e de checklist (arrastar.js). Aparece só no item sob o mouse ou,
   no celular, no item onde está o cursor. */
.folha {
  position: relative;
}

.folha :deep(.alca-item) {
  position: absolute;
  z-index: 5;
  display: grid;
  place-items: center;
  width: 20px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--texto-fraco);
  cursor: grab;
  opacity: 0;
  pointer-events: none;
  touch-action: none;
  transition: opacity var(--transicao);
}

.folha :deep(.alca-item.visivel) {
  opacity: 1;
  pointer-events: auto;
}

.folha :deep(.alca-item:hover) {
  background: var(--superficie);
  color: var(--texto-suave);
}

.folha :deep(.alca-item svg) {
  width: 10px;
  height: 16px;
  fill: currentColor;
}

.folha :deep(li.arrastando) {
  opacity: 0.4;
}

.folha :deep(.linha-soltar) {
  position: absolute;
  z-index: 4;
  height: 2px;
  border-radius: 1px;
  background: var(--link);
  pointer-events: none;
  visibility: hidden;
}

.folha :deep(.linha-soltar.visivel) {
  visibility: visible;
}

:global(body.arrastando-item),
:global(body.arrastando-item *) {
  cursor: grabbing !important;
  user-select: none;
}

/* Menu do "/". */
.menu-blocos {
  position: fixed;
  z-index: 30;
  display: grid;
  width: 220px;
  padding: 5px;
  border: 1px solid var(--borda);
  border-radius: var(--raio);
  background: var(--superficie);
}

.opcao {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 34px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--raio-p);
  background: transparent;
  font-size: 0.9rem;
  text-align: left;
  cursor: pointer;
}

.opcao[aria-selected='true'] {
  background: var(--ativo);
}

@media (pointer: coarse) {
  .opcao {
    min-height: 42px;
  }

  .folha :deep(input[type='checkbox']) {
    width: 18px;
    height: 18px;
  }

  .folha :deep(.alca-item) {
    width: 24px;
    height: 32px;
    color: var(--texto-suave);
  }
}
</style>
