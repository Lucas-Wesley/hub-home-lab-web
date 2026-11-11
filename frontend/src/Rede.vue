<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { dataHora, porcento } from './formato.js'

// Rede (H10): o AdGuard Home, o DNS da casa. Quando algo não abre, pausar o bloqueio diz na hora se a
// culpa é dele; aí é achar o domínio em "Bloqueados agora" e liberar.
const recado = inject('recado')

const resposta = ref(null) // { ligado, painel, rede, erro }
const erro = ref('')
const ocupado = ref(false)
const dominio = ref('')

const PAUSAS = [
  { minutos: 10, nome: '10 min' },
  { minutos: 60, nome: '1 hora' },
  { minutos: 0, nome: 'Até religar' },
]

const rede = computed(() => resposta.value?.rede)

async function carregar() {
  try {
    resposta.value = await api('/api/rede')
    erro.value = ''
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: o painel só abre na rede de casa.' : e.message
  }
}

// mudar faz a ação e mostra a rede como ficou (toda ação devolve o estado novo).
async function mudar(metodo, caminho, corpo, texto) {
  ocupado.value = true
  try {
    resposta.value = await api(caminho, { metodo, corpo })
    recado(texto)
    return true
  } catch (e) {
    recado(e.message, true)
    return false
  } finally {
    ocupado.value = false
  }
}

function pausar(p) {
  const texto = p.minutos ? `Bloqueio pausado por ${p.nome}. Tudo abre normalmente até lá.` : 'Bloqueio pausado até você religar.'
  mudar('POST', '/api/rede/protecao', { ligada: false, minutos: p.minutos }, texto)
}

function religar() {
  mudar('POST', '/api/rede/protecao', { ligada: true }, 'Bloqueio religado.')
}

function liberar(d) {
  return mudar('POST', '/api/rede/liberados', { dominio: d }, `${d} liberado. Pode levar até um minuto para valer em cada aparelho.`)
}

async function liberarDigitado() {
  if (await liberar(dominio.value)) dominio.value = ''
}

function bloquearDeNovo(d) {
  if (!confirm(`Bloquear ${d} de novo?`)) return
  mudar('DELETE', `/api/rede/liberados/${encodeURIComponent(d)}`, undefined, `${d} volta a ser bloqueado.`)
}

const hora = (iso) => new Date(iso).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit', timeZone: 'America/Sao_Paulo' })

onMounted(carregar)
</script>

<template>
  <div class="pagina">
    <div class="titulo">
      <h1>Rede</h1>
      <button class="botao icone" :disabled="ocupado" aria-label="Atualizar" title="Atualizar" @click="carregar"><Icone nome="atualizar" /></button>
    </div>
    <p class="suave explicacao">
      O AdGuard Home é o DNS da casa: bloqueia anúncios e rastreadores em todos os aparelhos. Se um site ou app parar de abrir, pause o bloqueio. Se
      voltar a abrir, a culpa era dele: ache o domínio em "Bloqueados agora" e libere. Pausado, o DNS continua respondendo, só sem bloquear.
    </p>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <div v-else-if="resposta && !resposta.ligado" class="aviso erro">O AdGuard não está ligado ao Hub (falta a senha do usuário hub no hub.env).</div>
    <div v-else-if="resposta?.erro" class="aviso erro">{{ resposta.erro }}</div>

    <template v-if="rede">
      <section class="estado" :class="{ pausado: !rede.protecao }">
        <div>
          <p class="situacao">
            <span class="ponto" aria-hidden="true"></span>
            <template v-if="rede.protecao">Bloqueando</template>
            <template v-else-if="rede.pausado_ate">Pausado até {{ hora(rede.pausado_ate) }}</template>
            <template v-else>Pausado até você religar</template>
          </p>
          <p class="suave numeros">
            {{ rede.consultas.toLocaleString('pt-BR') }} consultas nas últimas 24 h ·
            {{ rede.bloqueadas.toLocaleString('pt-BR') }} bloqueadas ({{ porcento(rede.bloqueadas, rede.consultas) }}%)
          </p>
        </div>
        <button v-if="!rede.protecao" class="botao principal" :disabled="ocupado" @click="religar">Religar o bloqueio</button>
        <div v-else class="pausar">
          <span class="suave pequeno">Pausar</span>
          <div class="pilulas">
            <button v-for="p in PAUSAS" :key="p.minutos" class="pilula" :disabled="ocupado" @click="pausar(p)">{{ p.nome }}</button>
          </div>
        </div>
      </section>

      <p class="rotulo">Liberar um domínio</p>
      <form class="liberar" @submit.prevent="liberarDigitado">
        <input v-model="dominio" class="campo" placeholder="exemplo.com ou o link inteiro" autocapitalize="off" autocomplete="off" spellcheck="false" aria-label="Domínio" />
        <button class="botao" :disabled="ocupado || !dominio.trim()">Liberar</button>
      </form>

      <p class="rotulo">Bloqueados agora</p>
      <ul v-if="rede.recentes.length" class="lista">
        <li v-for="b in rede.recentes" :key="b.dominio" class="linha">
          <span class="esquerda">
            <span class="dominio">{{ b.dominio }}</span>
            <span class="suave pequeno">{{ b.aparelho }} · {{ dataHora(b.quando) }}</span>
          </span>
          <button class="botao" :disabled="ocupado" @click="liberar(b.dominio)">Liberar</button>
        </li>
      </ul>
      <p v-else class="suave pequeno">Nada bloqueado ainda.</p>

      <p class="rotulo">Liberados</p>
      <ul v-if="rede.liberados.length" class="lista">
        <li v-for="d in rede.liberados" :key="d" class="linha">
          <span class="dominio">{{ d }}</span>
          <button class="botao" :disabled="ocupado" @click="bloquearDeNovo(d)">Bloquear de novo</button>
        </li>
      </ul>
      <p v-else class="suave pequeno">Nenhum domínio liberado.</p>

      <template v-if="rede.mais_bloqueados.length">
        <p class="rotulo">Mais bloqueados nas últimas 24 h</p>
        <ul class="lista">
          <li v-for="c in rede.mais_bloqueados" :key="c.dominio" class="linha">
            <span class="dominio">{{ c.dominio }}</span>
            <span class="valor">{{ c.vezes.toLocaleString('pt-BR') }}×</span>
          </li>
        </ul>
      </template>

      <p class="suave pequeno rodape-da-rede">
        Listas, aparelhos e o registro completo ficam no <a :href="resposta.painel" target="_blank" rel="noopener">painel do AdGuard</a>.
      </p>
    </template>
  </div>
</template>

<style scoped>
.titulo {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.explicacao {
  margin: 0 0 14px;
  font-size: 0.88rem;
  line-height: 1.5;
}

.pequeno {
  font-size: 0.82rem;
}

.estado {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px 20px;
  padding: 16px;
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  background: var(--marca-fundo);
}

.estado.pausado {
  background: var(--vermelho-fundo);
}

.situacao {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 4px;
  font-size: 1.1rem;
  font-weight: 600;
}

.ponto {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--marca);
}

.pausado .ponto {
  background: var(--vermelho);
}

.numeros {
  margin: 0;
  font-size: 0.88rem;
  font-variant-numeric: tabular-nums;
}

.pausar {
  display: grid;
  gap: 6px;
}

.pilulas {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.liberar {
  display: flex;
  gap: 8px;
}

.liberar .campo {
  flex: 1;
  min-width: 0;
}

.linha {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
}

.esquerda {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.dominio {
  overflow-wrap: anywhere;
}

.valor {
  color: var(--texto-suave);
  font-size: 0.9rem;
  font-variant-numeric: tabular-nums;
}

.rodape-da-rede {
  margin-top: 20px;
}
</style>
