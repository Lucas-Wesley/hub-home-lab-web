<script setup>
import { computed } from 'vue'
import { QUASE_CHEIO, bytes, porcento } from './formato.js'

// Uma linha com nome, quanto está em uso e uma barra. A barra fica vermelha quando passa de 85%.
const props = defineProps({
  nome: { type: String, required: true },
  usado: { type: Number, required: true },
  total: { type: Number, required: true },
  detalhe: { type: String, default: '' },
})

const fracao = computed(() => (props.total ? Math.min(1, props.usado / props.total) : 0))
const cheio = computed(() => fracao.value >= QUASE_CHEIO)
</script>

<template>
  <div class="medidor" :class="{ cheio }">
    <div class="topo">
      <span class="nome">{{ nome }} <span v-if="cheio" class="chip erro">Quase cheio</span></span>
      <span class="valor">{{ bytes(usado) }} de {{ bytes(total) }}</span>
    </div>
    <div
      class="trilha"
      role="meter"
      :aria-label="nome"
      aria-valuemin="0"
      aria-valuemax="100"
      :aria-valuenow="porcento(usado, total)"
      :aria-valuetext="`${porcento(usado, total)}% em uso`"
      :title="`${porcento(usado, total)}% em uso`"
    >
      <div class="preenchido" :style="{ width: `${fracao * 100}%` }" />
    </div>
    <p v-if="detalhe" class="detalhe">{{ detalhe }}</p>
  </div>
</template>

<style scoped>
.medidor {
  padding: 12px 0;
}

.topo {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
  margin-bottom: 8px;
}

.nome {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.valor {
  color: var(--texto-suave);
  font-size: 0.88rem;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
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

.cheio .trilha {
  background: var(--vermelho-fundo);
}

.cheio .preenchido {
  background: var(--vermelho);
}

.detalhe {
  margin: 6px 0 0;
  color: var(--texto-suave);
  font-size: 0.82rem;
}
</style>
