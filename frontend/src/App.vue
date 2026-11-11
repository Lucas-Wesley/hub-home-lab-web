<script setup>
import { computed, onBeforeUnmount, onMounted, provide, ref } from 'vue'
import Ajustes from './Ajustes.vue'
import Armazenamento from './Armazenamento.vue'
import Icone from './Icone.vue'
import Limpeza from './Limpeza.vue'
import Rede from './Rede.vue'
import Servidor from './Servidor.vue'
import { api } from './api.js'

// Rotas no hash, como nos outros apps: o "voltar" do celular volta para a tela anterior.
const hash = ref(location.hash)
const recado = ref(null) // { texto, erro } — aviso rápido
const versao = ref('')

const rota = computed(() => {
  const h = decodeURIComponent(hash.value.replace(/^#\/?/, ''))
  const [nome, ...resto] = h.split('/')
  return { nome: nome || 'armazenamento', param: resto.join('/') }
})

const MENU = [
  { rota: 'armazenamento', nome: 'Armazenamento', icone: 'disco' },
  { rota: 'servidor', nome: 'Servidor', icone: 'servidor' },
  { rota: 'limpeza', nome: 'Limpeza', icone: 'limpeza' },
  { rota: 'rede', nome: 'Rede', icone: 'rede' },
  { rota: 'ajustes', nome: 'Ajustes', icone: 'ajustes' },
]

function ir(destino) {
  location.hash = `#/${destino}`
}

// A tela de onde se foi para Ajustes: vai junto com a sugestão, para dar contexto.
const telaAnterior = ref('')

function aoMudarHash() {
  if (location.hash.startsWith('#/ajustes') && !hash.value.startsWith('#/ajustes')) telaAnterior.value = hash.value || '#/armazenamento'
  hash.value = location.hash
}

let tempoDoRecado
function mostrarRecado(texto, erro = false) {
  recado.value = { texto, erro }
  clearTimeout(tempoDoRecado)
  tempoDoRecado = setTimeout(() => (recado.value = null), 4000)
}
provide('recado', mostrarRecado)

onMounted(async () => {
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
  <div class="app">
    <aside class="menu" aria-label="Menu">
      <div class="marca"><span class="marca-icone"><Icone nome="hub" /></span>Hub</div>
      <nav>
        <button
          v-for="item in MENU"
          :key="item.rota"
          class="item"
          :class="{ ativo: rota.nome === item.rota }"
          :aria-current="rota.nome === item.rota ? 'page' : undefined"
          @click="ir(item.rota)"
        >
          <Icone :nome="item.icone" />
          <span class="nome">{{ item.nome }}</span>
        </button>
      </nav>
      <div class="rodape">{{ versao ? `versão ${versao.slice(0, 7)}` : 'sem conexão' }}</div>
    </aside>

    <main class="conteudo">
      <Servidor v-if="rota.nome === 'servidor'" />
      <Limpeza v-else-if="rota.nome === 'limpeza'" />
      <Rede v-else-if="rota.nome === 'rede'" />
      <Ajustes v-else-if="rota.nome === 'ajustes'" :tela="telaAnterior" :secao="rota.param" />
      <Armazenamento v-else />
    </main>

    <Transition name="recado">
      <div v-if="recado" class="recado vidro" :class="{ erro: recado.erro }" role="status">{{ recado.texto }}</div>
    </Transition>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
}

.menu {
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 20;
  display: flex;
  flex-direction: column;
  width: 216px;
  padding: max(28px, env(safe-area-inset-top)) 12px 16px;
  border-right: 1px solid var(--linha);
  background: var(--lateral);
}

/* A cor do app aparece aqui, no item ativo e no botão principal. */
.marca {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 8px 24px;
  font-size: 1.02rem;
  font-weight: 700;
  letter-spacing: -0.015em;
}

.marca-icone {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--marca);
  color: var(--marca-texto);
}

.marca-icone svg {
  width: 17px;
  height: 17px;
  stroke-width: 1.9;
}

nav {
  display: grid;
  gap: 1px;
}

.item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 34px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--raio-p);
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.9rem;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.item:hover {
  background: var(--linha);
  color: var(--texto);
}

.item.ativo {
  background: var(--item-ativo);
  box-shadow: 0 0 0 1px var(--linha), 0 1px 2px rgb(23 25 30 / 5%);
  color: var(--texto);
  font-weight: 600;
}

.item.ativo svg {
  color: var(--marca);
}


.rodape {
  margin-top: auto;
  padding: 0 10px;
  color: var(--texto-fraco);
  font-size: 0.72rem;
}

.conteudo {
  margin-left: 216px;
  min-height: 100vh;
}


.recado {
  position: fixed;
  left: 50%;
  bottom: calc(24px + env(safe-area-inset-bottom));
  z-index: 40;
  max-width: calc(100% - 32px);
  padding: 10px 16px;
  border: 1px solid var(--vidro-borda);
  border-radius: 999px;
  box-shadow: var(--sombra);
  font-size: 0.88rem;
  transform: translateX(-50%);
}

.recado.erro {
  color: var(--vermelho);
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

/* Celular: o menu vira uma barra de abas de vidro no pé da tela. */
@media (max-width: 820px) {
  .menu {
    inset: auto 0 0 0;
    width: auto;
    padding: 6px 8px max(6px, env(safe-area-inset-bottom));
    border-right: 0;
    border-top: 1px solid var(--linha);
    background: var(--vidro);
    -webkit-backdrop-filter: var(--blur);
    backdrop-filter: var(--blur);
  }

  .marca,
  .rodape {
    display: none;
  }

  nav {
    grid-template-columns: repeat(4, 1fr);
  }

  .item {
    position: relative;
    flex-direction: column;
    justify-content: center;
    gap: 2px;
    min-height: 48px;
    padding: 4px 0 2px;
    font-size: 0.66rem;
    font-weight: 500;
  }

  .item.ativo,
  .item:hover {
    background: transparent;
    box-shadow: none;
    font-weight: 600;
  }

  .item.ativo {
    color: var(--marca);
  }


  .conteudo {
    margin-left: 0;
  }

  .recado {
    bottom: calc(80px + env(safe-area-inset-bottom));
  }

}
</style>
