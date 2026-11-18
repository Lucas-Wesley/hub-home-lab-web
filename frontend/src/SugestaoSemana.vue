<script setup>
import { computed, inject, ref } from 'vue'
import GradeSemana from './GradeSemana.vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { meta, placar } from './rotina.js'

// Sugestão da IA: com as atividades e metas da tela (mesmo antes de salvar), o Claude monta um rascunho
// da semana para dar uma noção geral. Dá para pedir de novo com um comentário e, se ficar bom, usar
// como modelo.
const props = defineProps({
  atividades: { type: Array, required: true }, // já com id (as novas ganham um)
  ligada: { type: Boolean, default: true },
})
const emit = defineEmits(['usar'])
const recado = inject('recado')

const blocos = ref(null)
const explicacao = ref('')
const comentario = ref('')
const pensando = ref(false)

const porId = computed(() => Object.fromEntries(props.atividades.map((a) => [a.id, a])))
const metas = computed(() => (blocos.value ? placar(props.atividades, blocos.value) : []))

async function pedir(refazer) {
  pensando.value = true
  try {
    const corpo = { atividades: props.atividades }
    if (refazer) Object.assign(corpo, { comentario: comentario.value.trim(), anterior: blocos.value })
    const r = await api('/api/rotina/sugerir', { metodo: 'POST', corpo, tempo: 250000 })
    blocos.value = r.blocos
    explicacao.value = r.explicacao
    comentario.value = ''
  } catch (e) {
    recado(e.message, true)
  } finally {
    pensando.value = false
  }
}
</script>

<template>
  <section class="sugestao">
    <h2 class="rotulo">Sugestão da IA</h2>
    <p class="descricao-da-pagina">
      O Claude monta um rascunho de semana com as atividades e metas acima, mesmo sem salvar. Os compromissos (trabalho, futebol, namorada) ficam onde estão no modelo.
    </p>
    <p v-if="!ligada" class="aviso">A IA não está ligada neste Hub.</p>
    <template v-else>
      <button v-if="!blocos" class="botao" :disabled="pensando" @click="pedir(false)">
        <Icone nome="faisca" /> {{ pensando ? 'Montando a semana… (até 2 min)' : 'Ver a semana sugerida' }}
      </button>

      <template v-if="blocos">
        <p class="aviso explicacao">{{ explicacao }}</p>
        <GradeSemana :blocos="blocos" :por-id="porId" />
        <div class="metas">
          <span v-for="m in metas" :key="m.atividade.id" class="chip" :class="{ erro: m.estado === 'falta', ok: m.estado === 'ok' }">
            {{ m.atividade.nome }} {{ m.planejados }}/{{ meta(m.atividade) }}
          </span>
        </div>
        <label class="comentario">
          <span class="rotulo-campo">Pedir de novo, com um comentário</span>
          <textarea
            v-model="comentario"
            class="campo"
            rows="2"
            maxlength="2000"
            placeholder="Ex.: ficou bom, mas prioriza mais programação e tira a leitura da terça"
          />
        </label>
        <div class="linha-acoes">
          <button class="botao" :disabled="pensando || !comentario.trim()" @click="pedir(true)">
            <Icone nome="faisca" /> {{ pensando ? 'Refazendo…' : 'Refazer com o comentário' }}
          </button>
          <button class="botao" :disabled="pensando" @click="pedir(false)">Começar de novo</button>
          <button class="botao" :disabled="pensando" @click="emit('usar', blocos)">Usar como modelo</button>
        </div>
      </template>
    </template>
  </section>
</template>

<style scoped>
.sugestao .botao svg {
  width: 17px;
  height: 17px;
}

.explicacao {
  margin: 0;
}

.metas {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin: 12px 0;
}

.comentario {
  display: grid;
  gap: 5px;
  margin-bottom: 10px;
}

.rotulo-campo {
  color: var(--texto-suave);
  font-size: 0.81rem;
  font-weight: 600;
}

textarea.campo {
  resize: vertical;
}
</style>
