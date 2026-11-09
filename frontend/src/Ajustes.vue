<script setup>
import Sugestoes from './Sugestoes.vue'

defineProps({ tela: { type: String, default: '' } })
</script>

<template>
  <div class="pagina">
    <h1>Ajustes</h1>

    <p class="rotulo">Sugestões de melhoria</p>
    <p class="suave explicacao">Anote o que pode melhorar no Hub. Depois é só pedir ao Claude para ver as sugestões: ele faz as mudanças e responde aqui.</p>
    <Sugestoes :tela="tela" />

    <p class="rotulo">De onde vêm os números</p>
    <p class="suave explicacao">
      O disco físico, os armazenamentos e as VMs vêm da API do Proxmox, com um token só de leitura. O disco, a memória e as pastas de cada app são
      medidos dentro da VM apps, onde o Hub roda. O uso do Docker vem de um proxy que só deixa o painel ler o disco e limpar o cache: ele não cria
      nem entra em contêineres.
    </p>
  </div>
</template>

<style scoped>
.rotulo:first-of-type {
  margin-top: 8px;
}

.explicacao {
  margin: 0 0 14px;
  font-size: 0.88rem;
  line-height: 1.5;
}
</style>
