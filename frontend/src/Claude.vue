<script setup>
import { inject, onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { dataHora } from './formato.js'

// Claude (H11): as duas contas que dão a IA do Acervo, do Flashcards e do Streaming. Mostra o uso das
// janelas de 5 horas e da semana e troca a conta que os apps usam; a troca é sempre à mão.
const recado = inject('recado')

const contas = ref(null)
const erro = ref('')
const ocupado = ref(false)
const colando = ref('') // id da conta com o campo do token aberto
const token = ref('')

const QUASE_NO_FIM = 0.9

async function carregar() {
  ocupado.value = true
  try {
    // Medir chama o Claude de cada conta: leva uns segundos.
    contas.value = (await api('/api/claude?medir=1', { tempo: 90000 })).contas
    erro.value = ''
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: o painel só abre na rede de casa.' : e.message
  } finally {
    ocupado.value = false
  }
}

async function mudar(metodo, caminho, corpo, texto) {
  ocupado.value = true
  try {
    contas.value = (await api(caminho, { metodo, corpo, tempo: 90000 })).contas
    recado(texto)
    return true
  } catch (e) {
    recado(e.message, true)
    return false
  } finally {
    ocupado.value = false
  }
}

function usar(c) {
  mudar('POST', '/api/claude/ativa', { conta: c.id }, `Os apps passam a usar a conta ${c.nome} na próxima tarefa.`)
}

function abrirCampo(c) {
  colando.value = colando.value === c.id ? '' : c.id
  token.value = ''
}

async function guardar(c) {
  if (await mudar('PUT', `/api/claude/contas/${c.id}`, { token: token.value }, `Token da conta ${c.nome} guardado.`)) {
    colando.value = ''
    token.value = ''
  }
}

const pct = (u) => Math.round(u * 100)

function zera(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const hoje = new Date().toLocaleDateString('pt-BR', { timeZone: 'America/Sao_Paulo' })
  const dia = d.toLocaleDateString('pt-BR', { timeZone: 'America/Sao_Paulo' })
  const hora = d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit', timeZone: 'America/Sao_Paulo' })
  return dia === hoje ? `zera às ${hora}` : `zera ${dataHora(iso)}`
}

const data = (iso) => new Date(iso).toLocaleDateString('pt-BR', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'America/Sao_Paulo' })

onMounted(carregar)
</script>

<template>
  <div class="pagina">
    <div class="titulo">
      <h1>Claude</h1>
      <button class="botao icone" :disabled="ocupado" aria-label="Medir de novo" title="Medir de novo" @click="carregar"><Icone nome="atualizar" /></button>
    </div>
    <p class="suave explicacao">
      As contas que dão a IA do Acervo, do Flashcards e do Streaming. Os apps usam a conta ativa; a troca vale na próxima tarefa, sem reiniciar nada. O uso
      é da conta inteira: inclui o que você usa no Mac e no app do Claude.
    </p>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <p v-else-if="!contas" class="suave">Medindo as contas…</p>

    <section v-for="c in contas" :key="c.id" class="cartao conta" :class="{ ativa: c.ativa }">
      <div class="cabeca">
        <h2>{{ c.nome }} <span v-if="c.ativa" class="chip ok">Ativa</span></h2>
        <button v-if="c.configurada && !c.ativa" class="botao principal" :disabled="ocupado" @click="usar(c)">Usar esta conta</button>
      </div>

      <template v-if="c.configurada">
        <div v-if="c.erro" class="aviso erro">{{ c.erro }}</div>
        <template v-if="c.uso">
          <p v-if="c.uso.esgotada" class="aviso erro">Limite esgotado: o Claude recusa as tarefas desta conta até a janela zerar.</p>
          <div v-for="j in [{ nome: '5 horas', v: c.uso.cinco_horas }, { nome: 'Semana', v: c.uso.semana }]" :key="j.nome" class="janela" :class="{ fim: j.v.uso >= QUASE_NO_FIM }">
            <div class="topo">
              <span>{{ j.nome }}</span>
              <span class="valor">{{ pct(j.v.uso) }}% · {{ zera(j.v.zera_em) }}</span>
            </div>
            <div class="trilha" role="meter" :aria-label="`${c.nome}, ${j.nome}`" aria-valuemin="0" aria-valuemax="100" :aria-valuenow="pct(j.v.uso)">
              <div class="preenchido" :style="{ width: `${Math.min(1, j.v.uso) * 100}%` }" />
            </div>
          </div>
        </template>
        <p class="suave pequeno rodape">
          <template v-if="c.medido_em">Medido {{ dataHora(c.medido_em) }} · </template>
          Token gravado em {{ data(c.gravado_em) }}, vence em {{ data(c.vence_em) }}
          · <button class="link" @click="abrirCampo(c)">Trocar token</button>
        </p>
      </template>
      <p v-else class="suave pequeno">Sem token ainda.</p>

      <form v-if="!c.configurada || colando === c.id" class="colar" @submit.prevent="guardar(c)">
        <p class="suave pequeno">No terminal do Mac, logado na conta {{ c.nome }}, rode <code>claude setup-token</code> e cole aqui o token que ele mostrar.</p>
        <div class="linha-do-campo">
          <input v-model="token" class="campo" type="password" placeholder="sk-ant-oat01-…" autocomplete="off" spellcheck="false" :aria-label="`Token da conta ${c.nome}`" />
          <button class="botao" :disabled="ocupado || !token.trim()">Guardar</button>
        </div>
      </form>
    </section>
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

.conta {
  display: grid;
  gap: 10px;
  margin-bottom: 14px;
}

.conta.ativa {
  border-color: var(--marca);
}

.cabeca {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
}

.cabeca h2 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 1.05rem;
}

.topo {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 6px;
}

.valor {
  color: var(--texto-suave);
  font-size: 0.88rem;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.trilha {
  height: 8px;
  border-radius: 999px;
  background: var(--azul-fundo);
  overflow: hidden;
}

.preenchido {
  height: 100%;
  min-width: 4px;
  border-radius: 999px;
  background: var(--azul);
}

.fim .trilha {
  background: var(--vermelho-fundo);
}

.fim .preenchido {
  background: var(--vermelho);
}

.rodape {
  margin: 0;
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--link);
  font: inherit;
  cursor: pointer;
  text-decoration: underline;
}

.colar p {
  margin: 0 0 8px;
}

.linha-do-campo {
  display: flex;
  gap: 8px;
}

.linha-do-campo .campo {
  flex: 1;
  min-width: 0;
}
</style>
