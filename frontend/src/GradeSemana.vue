<script setup>
import { computed } from 'vue'
import { DIAS, LETRAS, conflitos, diasComChoque, doDia, minutos, nomeDoBloco } from './rotina.js'

// A semana inteira numa grade de horas: uma coluna por dia, os blocos na altura do horário. Serve para
// a semana, o modelo e a sugestão da IA. As tarefas sem horário não entram na grade.
const props = defineProps({
  blocos: { type: Array, required: true },
  porId: { type: Object, required: true },
  numeros: { type: Array, default: null }, // dia do mês de cada coluna (semana); null no modelo
  notas: { type: Object, default: () => ({}) },
  hoje: { type: Number, default: -1 },
})
const emit = defineEmits(['bloco', 'dia'])

const HORA = 30 // px por hora
const comHorario = computed(() => props.blocos.filter((b) => b.inicio))
const idsEmChoque = computed(() => conflitos(comHorario.value))
const diasEmChoque = computed(() => diasComChoque(comHorario.value))
const primeiraHora = computed(() => Math.floor(comHorario.value.reduce((m, b) => Math.min(m, minutos(b.inicio)), 6 * 60) / 60))
const horas = computed(() => Array.from({ length: 24 - primeiraHora.value }, (_, i) => primeiraHora.value + i))

function posicao(b) {
  const topo = ((minutos(b.inicio) - primeiraHora.value * 60) / 60) * HORA
  const altura = ((minutos(b.fim) - minutos(b.inicio)) / 60) * HORA
  return { top: `${topo}px`, height: `${Math.max(altura - 2, 4)}px` }
}

const corDe = (b) => props.porId[b.atividade]?.cor ?? (b.titulo ? 'evento' : 'cinza')
</script>

<template>
  <div class="grade-semana">
    <div class="grade" :style="{ '--hora': `${HORA}px`, '--horas': horas.length }">
      <div class="coluna-horas">
        <div class="topo-coluna" />
        <div class="corpo-coluna">
          <span v-for="h in horas" :key="h" class="hora">{{ String(h).padStart(2, '0') }}</span>
        </div>
      </div>
      <div v-for="(letra, d) in LETRAS" :key="d" class="coluna" :class="{ hoje: hoje === d }">
        <button class="topo-coluna" :aria-label="DIAS[d]" @click="emit('dia', d)">
          {{ letra }}<template v-if="numeros"> {{ numeros[d] }}</template>
          <span v-if="diasEmChoque.has(d)" class="alerta" title="Tem coisas no mesmo horário">!</span>
          <span v-else-if="notas[d]" class="ponto" :title="notas[d]" />
        </button>
        <div class="corpo-coluna">
          <button
            v-for="b in doDia(comHorario, d)"
            :key="b.id"
            class="pedaco"
            :class="[`cor-${corDe(b)}`, { feito: b.feito, conflito: idsEmChoque.has(b.id) }]"
            :style="posicao(b)"
            :title="`${nomeDoBloco(b, porId)} · ${b.inicio}–${b.fim}`"
            @click="emit('bloco', b)"
          >
            {{ nomeDoBloco(b, porId) }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.grade-semana {
  margin-top: 14px;
  overflow-x: auto;
}

.grade {
  display: grid;
  grid-template-columns: 30px repeat(7, minmax(40px, 1fr));
  gap: 0 3px;
}

.topo-coluna {
  position: relative;
  height: 28px;
  border: 0;
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.78rem;
  font-weight: 600;
  cursor: pointer;
}

.coluna.hoje .topo-coluna {
  color: var(--marca);
}

.ponto {
  position: absolute;
  top: 4px;
  right: 2px;
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--texto-fraco);
}

.alerta {
  position: absolute;
  top: 1px;
  right: 0;
  display: grid;
  place-items: center;
  width: 14px;
  height: 14px;
  border-radius: 999px;
  background: var(--vermelho);
  color: var(--fundo);
  font-size: 0.62rem;
  font-weight: 800;
}

.corpo-coluna {
  position: relative;
  height: calc(var(--hora) * var(--horas));
  background: repeating-linear-gradient(to bottom, var(--linha) 0 1px, transparent 1px var(--hora));
}

.coluna-horas .corpo-coluna {
  display: grid;
  grid-auto-rows: var(--hora);
  background: none;
}

.hora {
  color: var(--texto-fraco);
  font-size: 0.66rem;
  font-variant-numeric: tabular-nums;
  transform: translateY(-5px);
}

.pedaco {
  position: absolute;
  right: 0;
  left: 0;
  overflow: hidden;
  padding: 2px 4px;
  border: 0;
  border-left: 3px solid var(--cor);
  border-radius: 4px;
  background: color-mix(in srgb, var(--cor) 18%, var(--fundo));
  color: var(--texto);
  font-size: 0.66rem;
  line-height: 1.2;
  text-align: left;
  -webkit-hyphens: auto;
  hyphens: auto;
  cursor: pointer;
}

.pedaco.feito {
  opacity: 0.5;
}

.pedaco.conflito {
  outline: 1.5px solid var(--vermelho);
  z-index: 1;
}
</style>
