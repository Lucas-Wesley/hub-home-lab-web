<script setup>
defineProps({ nome: { type: String, required: true } })
</script>

<template>
  <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path v-if="nome === 'fechar'" d="M6 6l12 12M18 6L6 18" />
    <path v-else-if="nome === 'voltar'" d="M15 5l-7 7 7 7" />
    <path v-else-if="nome === 'avancar'" d="M9 5l7 7-7 7" />
    <template v-else-if="nome === 'hub'">
      <rect x="4" y="4" width="16" height="6" rx="1.5" />
      <rect x="4" y="14" width="16" height="6" rx="1.5" />
      <path d="M7.5 7h.01M7.5 17h.01" />
    </template>
    <template v-else-if="nome === 'disco'">
      <ellipse cx="12" cy="6.5" rx="7.5" ry="2.5" />
      <path d="M4.5 6.5v11c0 1.4 3.4 2.5 7.5 2.5s7.5-1.1 7.5-2.5v-11M4.5 12c0 1.4 3.4 2.5 7.5 2.5s7.5-1.1 7.5-2.5" />
    </template>
    <template v-else-if="nome === 'servidor'">
      <rect x="4" y="5" width="16" height="11" rx="1.5" />
      <path d="M9 20h6M12 16v4" />
    </template>
    <template v-else-if="nome === 'limpeza'">
      <path d="M14.5 4.5l-5 9" />
      <path d="M7 13.5h7.5l1.5 6.5H5.5z" />
      <path d="M9.5 20l.5-3M12.5 20l-.2-3" />
    </template>
    <template v-else-if="nome === 'ajustes'">
      <path d="M4 7.5h9M17 7.5h3M4 16.5h3M11 16.5h9" />
      <circle cx="15" cy="7.5" r="2" />
      <circle cx="9" cy="16.5" r="2" />
    </template>
    <template v-else-if="nome === 'claude'">
      <path d="M12 3.5v4M12 16.5v4M3.5 12h4M16.5 12h4M6 6l2.8 2.8M15.2 15.2 18 18M6 18l2.8-2.8M15.2 8.8 18 6" />
    </template>
    <template v-else-if="nome === 'rede'">
      <circle cx="12" cy="12" r="8" />
      <path d="M4 12h16M12 4c2.2 2.2 3.3 4.9 3.3 8s-1.1 5.8-3.3 8M12 4c-2.2 2.2-3.3 4.9-3.3 8s1.1 5.8 3.3 8" />
    </template>
    <template v-else-if="nome === 'atualizar'">
      <path d="M19.5 12a7.5 7.5 0 11-2.2-5.3" />
      <path d="M19.5 4.5v4h-4" />
    </template>
  </svg>
</template>
