<script setup>
import { computed, inject, onMounted, ref, watch } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { LETRAS, diasComChoque, doDia, nomeDoBloco, nomeDoMes, semanasDoMes, somarDias, somarMeses } from './rotina.js'

// O mês: as semanas planejadas com os eventos e tarefas de cada dia, o "!" dos choques e as semanas
// ainda sem plano (claras). Tocar num dia abre a semana nele.
const props = defineProps({
  cfg: { type: Object, required: true },
  porId: { type: Object, required: true },
  versao: { type: Number, default: 0 }, // muda quando algo foi adicionado fora daqui
})
const emit = defineEmits(['abrir', 'novo'])
const recado = inject('recado')

const mes = ref(props.cfg.hoje.slice(0, 7))
const semanas = ref({}) // inicio → semana planejada
const carregando = ref(false)

const lista = computed(() => semanasDoMes(mes.value))

async function carregar() {
  carregando.value = true
  try {
    const r = await api(`/api/rotina/semanas?de=${lista.value[0]}&ate=${lista.value.at(-1)}`)
    semanas.value = Object.fromEntries(r.map((s) => [s.inicio, s]))
  } catch (e) {
    recado(e.message, true)
  } finally {
    carregando.value = false
  }
}

watch([mes, () => props.versao], carregar)
onMounted(carregar)

// Cada dia da grade: data, se é do mês, os eventos/tarefas (o que não veio do modelo) e o "!".
const linhas = computed(() =>
  lista.value.map((inicio) => {
    const s = semanas.value[inicio]
    const choque = s ? diasComChoque(s.blocos) : new Set()
    return {
      inicio,
      planejada: !!s,
      dias: LETRAS.map((_, d) => {
        const data = somarDias(inicio, d)
        const extras = s ? doDia(s.blocos, d).filter((b) => b.extra || b.titulo) : []
        return { d, data, numero: Number(data.slice(8)), doMes: data.slice(0, 7) === mes.value, hoje: data === props.cfg.hoje, extras, choque: choque.has(d), nota: s?.notas?.[d] ?? '' }
      }),
    }
  }),
)

// Planeja pelo modelo as semanas do mês (a atual e as seguintes) que ainda não têm plano.
const semPlano = computed(() => lista.value.filter((i) => !semanas.value[i] && i >= props.cfg.semana_atual))
async function planejarTodas() {
  carregando.value = true
  try {
    for (const inicio of semPlano.value) await api(`/api/rotina/semanas/${inicio}`, { metodo: 'POST' })
    recado(`${semPlano.value.length === 1 ? 'Uma semana planejada' : `${semPlano.value.length} semanas planejadas`} pelo modelo.`)
  } catch (e) {
    recado(e.message, true)
  }
  await carregar()
}
</script>

<template>
  <section class="mes" aria-label="Mês">
    <div class="navegar">
      <button class="botao icone" aria-label="Mês anterior" @click="mes = somarMeses(mes, -1)"><Icone nome="voltar" /></button>
      <strong>{{ nomeDoMes(mes) }}</strong>
      <button class="botao icone" aria-label="Próximo mês" @click="mes = somarMeses(mes, 1)"><Icone nome="avancar" /></button>
    </div>

    <div class="calendario" :class="{ carregando }">
      <span v-for="(l, i) in LETRAS" :key="`c${i}`" class="cabeca">{{ l }}</span>
      <template v-for="linha in linhas" :key="linha.inicio">
        <button
          v-for="dia in linha.dias"
          :key="dia.data"
          class="celula"
          :class="{ fora: !dia.doMes, hoje: dia.hoje, 'sem-plano': !linha.planejada }"
          :aria-label="`Abrir ${dia.data}`"
          @click="emit('abrir', { inicio: linha.inicio, dia: dia.d })"
        >
          <span class="numero">{{ dia.numero }}</span>
          <span v-if="dia.choque" class="alerta" title="Tem coisas no mesmo horário">!</span>
          <span v-if="dia.nota" class="nota">{{ dia.nota }}</span>
          <span v-for="b in dia.extras.slice(0, 3)" :key="b.id" class="evento" :class="`cor-${porId[b.atividade]?.cor ?? 'evento'}`">
            <span v-if="b.inicio" class="hora">{{ b.inicio }}</span> {{ nomeDoBloco(b, porId) }}
          </span>
          <span v-if="dia.extras.length > 3" class="mais">+{{ dia.extras.length - 3 }}</span>
        </button>
      </template>
    </div>

    <p class="suave legenda">As semanas claras ainda não têm plano: nascem do modelo quando você planeja ou adiciona algo nelas.</p>
    <div class="linha-acoes">
      <button class="botao" @click="emit('novo', mes === cfg.hoje.slice(0, 7) ? cfg.hoje : `${mes}-01`)">+ Evento com data</button>
      <button v-if="semPlano.length" class="botao" :disabled="carregando" @click="planejarTodas">
        Planejar {{ semPlano.length === 1 ? 'a semana sem plano' : `as ${semPlano.length} semanas sem plano` }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.navegar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  font-size: 1.0625rem;
}

.calendario {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  border-top: 1px solid var(--linha);
  border-left: 1px solid var(--linha);
  transition: opacity 0.15s ease;
}

.calendario.carregando {
  opacity: 0.6;
}

.cabeca {
  padding: 6px 0;
  border-right: 1px solid var(--linha);
  border-bottom: 1px solid var(--linha);
  color: var(--texto-suave);
  font-size: 0.75rem;
  font-weight: 600;
  text-align: center;
}

.celula {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 2px;
  min-width: 0;
  min-height: 84px;
  padding: 4px 3px;
  border: 0;
  border-right: 1px solid var(--linha);
  border-bottom: 1px solid var(--linha);
  background: var(--fundo);
  text-align: left;
  cursor: pointer;
}

.celula:hover {
  background: var(--superficie);
}

.celula.sem-plano {
  background: repeating-linear-gradient(135deg, var(--superficie) 0 6px, var(--fundo) 6px 12px);
}

.celula.fora .numero {
  color: var(--texto-fraco);
}

.numero {
  font-size: 0.82rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.celula.hoje .numero {
  align-self: flex-start;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--marca);
  color: var(--marca-texto);
}

.alerta {
  position: absolute;
  top: 4px;
  right: 4px;
  display: grid;
  place-items: center;
  width: 16px;
  height: 16px;
  border-radius: 999px;
  background: var(--vermelho);
  color: var(--fundo);
  font-size: 0.68rem;
  font-weight: 800;
}

.nota {
  overflow: hidden;
  color: var(--vermelho);
  font-size: 0.64rem;
  font-weight: 600;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.evento {
  overflow: hidden;
  padding: 1px 3px;
  border-left: 2px solid var(--cor);
  border-radius: 3px;
  background: color-mix(in srgb, var(--cor) 16%, var(--fundo));
  font-size: 0.64rem;
  line-height: 1.25;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.evento .hora {
  color: var(--texto-suave);
  font-variant-numeric: tabular-nums;
}

.mais {
  color: var(--texto-suave);
  font-size: 0.64rem;
}

.legenda {
  margin: 10px 0 12px;
  font-size: 0.82rem;
}

@media (max-width: 520px) {
  .celula {
    min-height: 68px;
  }

  .evento .hora {
    display: none;
  }
}
</style>
