<script setup>
import { inject, onBeforeUnmount, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { ouvir, temVoz } from './voz.js'

// Adicionar pela voz (ou escrevendo): "quinta das 19 às 22, encontro com amigos". O Claude entende a
// data e o horário, e o Hub só adiciona: o que já estava no horário fica, e o dia ganha um "!".
const emit = defineEmits(['adicionado'])
const recado = inject('recado')

const texto = ref('')
const estado = ref('parado') // parado | ouvindo | pensando
let parar = null

function microfone() {
  if (estado.value === 'ouvindo') return parar?.()
  if (estado.value !== 'parado') return
  texto.value = ''
  estado.value = 'ouvindo'
  parar = ouvir({
    aoMudar: (t) => (texto.value = t),
    aoTerminar: (t, erro) => {
      parar = null
      estado.value = 'parado'
      if (erro) recado(erro, true)
      // Falou e parou: já manda, sem mais um toque.
      else if (t.trim()) enviar()
    },
  })
}

async function enviar() {
  const t = texto.value.trim()
  if (!t || estado.value === 'pensando') return
  estado.value = 'pensando'
  try {
    const r = await api('/api/rotina/falar', { metodo: 'POST', corpo: { texto: t }, tempo: 130000 })
    texto.value = ''
    emit('adicionado', r)
  } catch (e) {
    recado(e.message, true)
  } finally {
    estado.value = 'parado'
  }
}

onBeforeUnmount(() => parar?.())
</script>

<template>
  <form class="falar" :class="estado" @submit.prevent="enviar">
    <input
      v-model="texto"
      class="campo"
      :placeholder="estado === 'ouvindo' ? 'Ouvindo…' : 'Adicionar: “quinta das 19 às 22, encontro com amigos”'"
      aria-label="Adicionar à rotina"
      :disabled="estado === 'pensando'"
      maxlength="1000"
    />
    <button
      v-if="temVoz"
      type="button"
      class="botao icone mic"
      :aria-label="estado === 'ouvindo' ? 'Parar e adicionar' : 'Falar'"
      :title="estado === 'ouvindo' ? 'Parar e adicionar' : 'Falar'"
      :disabled="estado === 'pensando'"
      @click="microfone"
    >
      <Icone :nome="estado === 'ouvindo' ? 'parar' : 'microfone'" />
    </button>
    <button class="botao principal enviar" :disabled="!texto.trim() || estado !== 'parado'" aria-label="Adicionar">
      <span v-if="estado === 'pensando'" class="girando" aria-hidden="true" />
      <Icone v-else nome="enviar" />
    </button>
  </form>
  <p v-if="estado === 'pensando'" class="suave pensando">O Claude está encaixando na agenda…</p>
</template>

<style scoped>
.falar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 16px;
}

.falar .campo {
  flex: 1;
  min-width: 0;
  border-radius: 999px;
  padding-left: 16px;
}

.mic {
  flex-shrink: 0;
  background: var(--superficie);
}

.ouvindo .mic {
  background: var(--vermelho-fundo);
  color: var(--vermelho);
  animation: pulso 1.6s ease-in-out infinite;
}

.enviar {
  flex-shrink: 0;
  justify-content: center;
  width: 38px;
  min-height: 38px;
  padding: 0;
  border-radius: 999px;
}

.enviar svg {
  width: 18px;
  height: 18px;
}

.pensando {
  margin: -8px 0 16px 16px;
  font-size: 0.82rem;
}

.girando {
  width: 16px;
  height: 16px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: giro 0.8s linear infinite;
}

@keyframes pulso {
  50% {
    box-shadow: 0 0 0 6px color-mix(in srgb, var(--vermelho) 15%, transparent);
  }
}

@keyframes giro {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .ouvindo .mic,
  .girando {
    animation: none;
  }
}
</style>
