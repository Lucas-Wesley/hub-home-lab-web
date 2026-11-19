<script setup>
import { computed, inject, onMounted, ref } from 'vue'
import { api } from './api.js'
import { FREQUENCIAS, bytes, dataHora } from './formato.js'

// Limpeza do Docker da VM: agora, pelo botão, ou pela agenda (a cada N dias, numa hora).
const recado = inject('recado')

const estado = ref(null) // { agenda, historico, docker_ligado, em_andamento }
const docker = ref(null)
const agenda = ref(null) // cópia editável
const erro = ref('')
const limpando = ref(false)
const salvando = ref(false)

const HORAS = Array.from({ length: 24 }, (_, h) => h)

async function carregar() {
  try {
    const [l, a] = await Promise.all([api('/api/limpeza'), api('/api/armazenamento', { tempo: 60000 })])
    estado.value = l
    docker.value = a.docker
    agenda.value = { ...l.agenda }
    limpando.value = l.em_andamento
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: o painel só abre na rede de casa.' : e.message
  }
}

const mudou = computed(() => {
  if (!agenda.value || !estado.value) return false
  const a = agenda.value
  const b = estado.value.agenda
  return ['ativa', 'a_cada_dias', 'hora', 'cache_de_build', 'imagens_soltas'].some((k) => a[k] !== b[k])
})

async function limparAgora() {
  limpando.value = true
  try {
    const l = await api('/api/limpeza', {
      metodo: 'POST',
      corpo: { cache_de_build: true, imagens_soltas: true },
      tempo: 16 * 60 * 1000,
    })
    if (l.erro) recado(l.erro, true)
    else recado(`Liberados ${bytes(l.cache_liberado + l.imagens_liberado)}.`)
  } catch (e) {
    recado(e.message, true)
  } finally {
    limpando.value = false
    await carregar()
  }
}

async function salvar() {
  salvando.value = true
  try {
    const salva = await api('/api/limpeza/agenda', { metodo: 'PUT', corpo: agenda.value })
    estado.value.agenda = salva
    agenda.value = { ...salva }
    recado(salva.ativa ? `Agenda salva. Próxima limpeza: ${dataHora(salva.proxima_em)}.` : 'Limpeza automática desligada.')
  } catch (e) {
    recado(e.message, true)
  } finally {
    salvando.value = false
  }
}

const hora2 = (h) => `${String(h).padStart(2, '0')}:00`

onMounted(carregar)
</script>

<template>
  <div class="pagina">
    <div class="topo-pagina"><h1>Limpeza</h1></div>
    <p class="suave explicacao">
      Cada deploy constrói o app de novo na VM, e o Docker guarda um cache desse build. Com o tempo, esse cache vira a maior parte do disco. A limpeza
      apaga o cache e as imagens soltas, que sobram de builds antigos. Os dados dos apps e a versão anterior de cada um, usada para voltar se um deploy
      falhar, não são tocados. Depois de limpar, o próximo deploy de cada app demora um pouco mais.
    </p>

    <div v-if="erro" class="aviso">{{ erro }}</div>
    <div v-else-if="estado && !estado.docker_ligado" class="aviso">O Docker não está ligado ao painel; a limpeza não roda.</div>

    <template v-if="estado">
      <div class="quadros">
      <section class="quadro">
      <div class="quadro-topo"><h2 class="rotulo-secao">Agora</h2></div>
      <div class="agora">
        <p v-if="docker">
          Para liberar: <strong>{{ bytes(docker.cache_liberavel) }}</strong> de cache de build e <strong>{{ bytes(docker.imagens_soltas) }}</strong> de imagens soltas.
        </p>
        <button class="botao primario" :disabled="limpando || !estado.docker_ligado" @click="limparAgora">{{ limpando ? 'Limpando…' : 'Limpar agora' }}</button>
      </div>

      </section>

      <section class="quadro">
      <div class="quadro-topo"><h2 class="rotulo-secao">Agenda</h2></div>
      <form v-if="agenda" class="agenda" @submit.prevent="salvar">
        <label class="marcar">
          <input v-model="agenda.ativa" type="checkbox" />
          Limpar automaticamente
        </label>

        <fieldset :disabled="!agenda.ativa">
          <legend class="suave pequeno">Com que frequência</legend>
          <div class="abas" role="radiogroup" aria-label="Frequência">
            <button
              v-for="f in FREQUENCIAS"
              :key="f.dias"
              type="button"
              role="radio"
              :class="{ ativa: agenda.a_cada_dias === f.dias }"
              :aria-checked="agenda.a_cada_dias === f.dias"
              @click="agenda.a_cada_dias = f.dias"
            >
              {{ f.nome }}
            </button>
          </div>

          <label class="hora">
            <span class="suave pequeno">Às</span>
            <select v-model.number="agenda.hora" class="campo" aria-label="Hora da limpeza">
              <option v-for="h in HORAS" :key="h" :value="h">{{ hora2(h) }}</option>
            </select>
          </label>

          <div class="oque">
            <label class="marcar"><input v-model="agenda.cache_de_build" type="checkbox" /> Cache de build</label>
            <label class="marcar"><input v-model="agenda.imagens_soltas" type="checkbox" /> Imagens soltas</label>
          </div>
        </fieldset>

        <div class="linha-acoes">
          <button class="botao" :disabled="!mudou || salvando">{{ salvando ? 'Salvando…' : 'Salvar agenda' }}</button>
          <span v-if="estado.agenda.ativa && !mudou" class="suave pequeno">Próxima limpeza: {{ dataHora(estado.agenda.proxima_em) }}</span>
        </div>
      </form>
      </section>
      </div>

      <h2 class="secao">Histórico</h2>
      <ul v-if="estado.historico.length" class="lista">
        <li v-for="l in estado.historico" :key="l.id" class="linha">
          <span class="esquerda">
            <span>{{ dataHora(l.iniciada_em) }}</span>
            <span class="etiqueta">{{ l.origem === 'agenda' ? 'Agenda' : 'Manual' }}</span>
            <span v-if="l.erro" class="etiqueta vermelho" :title="l.erro">Erro</span>
            <span v-else-if="!l.terminada_em" class="etiqueta azul">Limpando</span>
          </span>
          <span class="valor">{{ bytes(l.cache_liberado + l.imagens_liberado) }} liberados</span>
          <p v-if="l.erro" class="erro-texto">{{ l.erro }}</p>
        </li>
      </ul>
      <p v-else class="suave pequeno">Nenhuma limpeza ainda.</p>
    </template>
  </div>
</template>

<style scoped>
.explicacao {
  margin: 0 0 14px;
  font-size: 14px;
  line-height: 1.5;
}

.agora {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.agora p {
  margin: 0;
  line-height: 1.5;
}

.agenda {
  display: grid;
  gap: 14px;
}

fieldset {
  display: grid;
  gap: 12px;
  margin: 0;
  padding: 0;
  border: 0;
}

fieldset:disabled {
  opacity: 0.5;
}

legend {
  margin-bottom: 6px;
  padding: 0;
}

.marcar {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.marcar input {
  width: 16px;
  height: 16px;
  accent-color: var(--azul);
}

.hora {
  display: flex;
  align-items: center;
  gap: 10px;
}

.hora .campo {
  width: auto;
}

.oque {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 20px;
}

.linha {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 6px 12px;
  padding: 12px 0;
}

.esquerda {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 8px;
}

.valor {
  color: var(--texto-suave);
  font-size: 13.5px;
  font-variant-numeric: tabular-nums;
}

.erro-texto {
  flex-basis: 100%;
  margin: 0;
  color: var(--vermelho);
  font-size: 13px;
}
</style>
