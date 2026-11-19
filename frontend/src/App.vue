<script setup>
import { computed, onBeforeUnmount, onMounted, provide, ref } from 'vue'
import Ajustes from './Ajustes.vue'
import Aplicacoes from './Aplicacoes.vue'
import Armazenamento from './Armazenamento.vue'
import Claude from './Claude.vue'
import Design from './Design.vue'
import Icone from './Icone.vue'
import Inicio from './Inicio.vue'
import Limpeza from './Limpeza.vue'
import Rede from './Rede.vue'
import Rotina from './Rotina.vue'
import Servidor from './Servidor.vue'
import { api } from './api.js'
import { aplicarTema, lerTema, proximoTema } from './tema.js'

// Rotas no hash, como nos outros apps: o "voltar" do celular volta para a tela anterior.
const hash = ref(location.hash)
const recado = ref(null) // { texto, erro } — aviso rápido
const versao = ref('')

const rota = computed(() => {
  const h = decodeURIComponent(hash.value.replace(/^#\/?/, ''))
  const [nome, ...resto] = h.split('/')
  return { nome: nome || 'inicio', param: resto.join('/') }
})

const MENU = [
  { rota: 'inicio', nome: 'Início', icone: 'inicio' },
  { rota: 'rotina', nome: 'Rotina', icone: 'rotina' },
  { rota: 'aplicacoes', nome: 'Aplicações', curto: 'Apps', icone: 'apps' },
  { rota: 'armazenamento', nome: 'Armazenamento', curto: 'Disco', icone: 'disco' },
  { rota: 'servidor', nome: 'Servidor', icone: 'servidor' },
  { rota: 'limpeza', nome: 'Limpeza', icone: 'limpeza' },
  { rota: 'rede', nome: 'Rede', icone: 'rede' },
  { rota: 'claude', nome: 'Claude', icone: 'claude' },
  { rota: 'ajustes', nome: 'Ajustes', icone: 'ajustes' },
]

function ir(destino) {
  location.hash = `#/${destino}`
}

// A tela de onde se foi para Ajustes: vai junto com a sugestão, para dar contexto.
const telaAnterior = ref('')

function aoMudarHash() {
  if (location.hash.startsWith('#/ajustes') && !hash.value.startsWith('#/ajustes')) telaAnterior.value = hash.value || '#/inicio'
  hash.value = location.hash
}

const tema = ref(lerTema())
const NOMES_TEMA = { auto: 'Tema do aparelho', claro: 'Tema claro', escuro: 'Tema escuro' }
const ICONES_TEMA = { auto: 'auto', claro: 'sol', escuro: 'lua' }
function escolherTema(t) {
  tema.value = t
  aplicarTema(t)
}
const trocarTema = () => escolherTema(proximoTema(tema.value))

let tempoDoRecado
function mostrarRecado(texto, erro = false) {
  recado.value = { texto, erro }
  clearTimeout(tempoDoRecado)
  tempoDoRecado = setTimeout(() => (recado.value = null), 4000)
}
provide('recado', mostrarRecado)

onMounted(async () => {
  aplicarTema(tema.value)
  window.addEventListener('hashchange', aoMudarHash)
  try {
    versao.value = (await api('/api/health', { tempo: 4000 })).version
  } catch {
    versao.value = ''
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('hashchange', aoMudarHash)
})
</script>

<template>
  <div class="moldura">
    <aside class="menu" aria-label="Menu">
      <a class="marca" href="#/inicio">
        <span class="marca-icone"><Icone nome="hub" :tamanho="18" /></span>
        <span class="marca-texto">
          <span class="marca-sobre">Homelab</span>
          <span class="marca-nome">Hub</span>
        </span>
      </a>
      <nav>
        <button
          v-for="item in MENU"
          :key="item.rota"
          class="nav-item"
          :class="{ ativo: rota.nome === item.rota }"
          :aria-current="rota.nome === item.rota ? 'page' : undefined"
          @click="ir(item.rota)"
        >
          <Icone :nome="item.icone" :tamanho="18" />
          <span>{{ item.nome }}</span>
        </button>
      </nav>
      <div class="rodape">
        <span>{{ versao ? `versão ${versao.slice(0, 7)}` : 'sem conexão' }}</span>
        <button class="botao discreto icone" :title="NOMES_TEMA[tema]" :aria-label="NOMES_TEMA[tema]" @click="trocarTema">
          <Icone :nome="ICONES_TEMA[tema]" />
        </button>
      </div>
    </aside>

    <main class="conteudo">
      <Rotina v-if="rota.nome === 'rotina'" />
      <Servidor v-else-if="rota.nome === 'servidor'" />
      <Limpeza v-else-if="rota.nome === 'limpeza'" />
      <Rede v-else-if="rota.nome === 'rede'" />
      <Claude v-else-if="rota.nome === 'claude'" />
      <Ajustes v-else-if="rota.nome === 'ajustes'" :tela="telaAnterior" :secao="rota.param" :tema="tema" @tema="escolherTema" />
      <Design v-else-if="rota.nome === 'design'" />
      <Armazenamento v-else-if="rota.nome === 'armazenamento'" />
      <Aplicacoes v-else-if="rota.nome === 'aplicacoes'" />
      <Inicio v-else />
    </main>
  </div>

  <Transition name="recado">
    <div v-if="recado" class="recado" :class="{ erro: recado.erro }" role="status">{{ recado.texto }}</div>
  </Transition>

  <nav class="abas-celular" aria-label="Seções">
    <a
      v-for="item in MENU"
      :key="item.rota"
      :href="`#/${item.rota}`"
      :class="{ ativa: rota.nome === item.rota }"
      :aria-current="rota.nome === item.rota ? 'page' : undefined"
    >
      <Icone :nome="item.icone" :tamanho="20" />
      <span class="nome" :class="{ 'tem-curto': item.curto }">{{ item.nome }}</span>
      <span v-if="item.curto" class="nome curto" aria-hidden="true">{{ item.curto }}</span>
    </a>
  </nav>
</template>

<style scoped>
/* Sem barra no topo (referência de espaço): a marca abre a lateral e o tema fica no pé dela. */
.moldura { display: flex; }

.menu {
  position: sticky;
  top: 0;
  flex: none;
  display: flex;
  flex-direction: column;
  width: var(--nav);
  height: 100vh;
  padding: var(--e4) var(--e3);
  overflow-y: auto;
  border-right: 1px solid var(--borda);
  background: var(--superficie);
}
.menu nav { display: grid; gap: 2px; }

.marca {
  display: flex;
  align-items: center;
  gap: var(--e3);
  margin: 0 0 var(--e4);
  padding: var(--e2);
  color: var(--texto);
  text-decoration: none;
}
.marca-icone {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border: 1px solid var(--borda);
  border-radius: var(--raio-p);
  background: var(--fundo);
}
.marca-texto { display: grid; line-height: 1.2; }
.marca-sobre { font-size: 10.5px; font-weight: 600; letter-spacing: 0.08em; text-transform: uppercase; color: var(--texto-fraco); }
.marca-nome { font-family: var(--fonte-titulo); font-size: 18px; font-weight: 500; }

.rodape {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
  padding: var(--e3) 0 0 var(--e2);
  color: var(--texto-fraco);
  font-size: 12px;
}

.conteudo {
  flex: 1;
  min-width: 0;
  min-height: 100vh;
}

.recado-enter-active,
.recado-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.recado-enter-from,
.recado-leave-to {
  opacity: 0;
  transform: translate(-50%, 8px);
}

.abas-celular { display: none; }

@media (max-width: 899px) {
  .menu { display: none; }

  .abas-celular {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 30;
    display: grid;
    grid-template-columns: repeat(9, 1fr);
    padding: var(--e1) var(--e2) calc(var(--e1) + env(safe-area-inset-bottom));
    background: var(--fundo);
    border-top: 1px solid var(--borda);
  }
  .abas-celular a {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: 6px 0;
    font-size: 10.5px;
    font-weight: 500;
    color: var(--texto-fraco);
    text-decoration: none;
    border-radius: var(--raio-p);
  }
  .abas-celular a.ativa { color: var(--texto); background: var(--superficie-2); }

  /* Nove abas numa linha: o nome longo dá lugar ao curto (o leitor de tela continua lendo o longo). */
  .abas-celular .nome.tem-curto {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }

  .recado { bottom: calc(80px + env(safe-area-inset-bottom)); }
}

/* Celular estreito: nove nomes só cabem com a letra um pouco menor. */
@media (max-width: 420px) {
  .abas-celular a { font-size: 9px; letter-spacing: -0.02em; }
}
</style>
