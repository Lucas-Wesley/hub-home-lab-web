<script setup>
import { computed } from 'vue'
import Icone from './Icone.vue'
import Sugestoes from './Sugestoes.vue'

// Ajustes em níveis (design system): a primeira tela lista as seções; cada uma abre numa página.
const props = defineProps({
  tela: { type: String, default: '' },
  secao: { type: String, default: '' },
  tema: { type: String, default: 'auto' },
})
const emit = defineEmits(['tema'])
const TEMAS = [
  ['auto', 'Do aparelho'],
  ['claro', 'Claro'],
  ['escuro', 'Escuro'],
]

const SECOES = [
  { id: 'sugestoes', nome: 'Sugestões de melhoria', descricao: 'Anote o que pode melhorar no Hub; o Claude faz e responde aqui.' },
  { id: 'numeros', nome: 'De onde vêm os números', descricao: 'Como o Hub mede o disco, a memória e o Docker.' },
]
const atual = computed(() => SECOES.find((s) => s.id === props.secao))
</script>

<template>
  <div class="pagina">
    <template v-if="!atual">
      <div class="topo-pagina"><h1>Ajustes</h1></div>
      <div class="linha-tema">
        <span>
          <span class="nome">Tema</span>
          <span class="descricao">Vale só neste aparelho.</span>
        </span>
        <div class="abas" role="radiogroup" aria-label="Tema">
          <button v-for="[id, nome] in TEMAS" :key="id" role="radio" :class="{ ativa: tema === id }" :aria-checked="tema === id" @click="emit('tema', id)">
            {{ nome }}
          </button>
        </div>
      </div>
      <nav class="lista-de-ajustes" aria-label="Seções dos ajustes">
        <a v-for="s in SECOES" :key="s.id" :href="`#/ajustes/${s.id}`">
          <span>
            <span class="nome">{{ s.nome }}</span>
            <span class="descricao">{{ s.descricao }}</span>
          </span>
          <Icone nome="avancar" class="seta" />
        </a>
      </nav>
    </template>

    <template v-else>
      <a href="#/ajustes" class="voltar-ajustes"><Icone nome="voltar" /> Ajustes</a>
      <div class="topo-pagina"><h1>{{ atual.nome }}</h1></div>

      <template v-if="atual.id === 'sugestoes'">
        <p class="lead">Anote o que pode melhorar no Hub. Depois é só pedir ao Claude para ver as sugestões: ele faz as mudanças e responde aqui.</p>
        <Sugestoes :tela="tela" />
      </template>

      <p v-else-if="atual.id === 'numeros'" class="leitura">
        O disco físico, os armazenamentos e as VMs vêm da API do Proxmox, com um token só de leitura. O disco, a memória e as pastas de cada app são
        medidos dentro da VM apps, onde o Hub roda. O uso do Docker vem de um proxy que só deixa o painel ler o disco e limpar o cache: ele não cria
        nem entra em contêineres.
      </p>
    </template>
  </div>
</template>

<style scoped>
.linha-tema {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--e2) var(--e4);
  max-width: var(--leitura);
  padding: var(--e3) 0;
  border-top: 1px solid var(--borda);
}
.linha-tema .nome { display: block; font-size: 15px; font-weight: 600; }
.linha-tema .descricao { display: block; margin-top: 3px; color: var(--texto-suave); font-size: 14px; }
</style>
