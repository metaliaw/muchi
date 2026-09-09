<script setup>
/** Las Ofertas, ordenadas por Precio dentro de cada Moneda por el BFF. */
import { formatAmount, formatClp } from '../api.js'

const emit = defineEmits(['look'])

defineProps({
  offers: { type: Array, default: () => [] },
  summary: { type: Object, default: null },
  notices: { type: Array, default: () => [] },
  placeholder: { type: String, default: '' },
})
</script>

<template>
  <section>
    <p v-for="notice in notices" :key="notice.text"
       :class="notice.level === 'warning' ? 'mu-aviso' : 'mu-caption'">{{ notice.text }}</p>

    <div v-if="offers.length" class="mu-fichas">
      <div class="mu-panel mu-ficha">
        <span class="mu-caption">Menor Observado</span>
        <strong>{{ formatClp(summary?.lowest_clp) }}</strong>
      </div>
      <div class="mu-panel mu-ficha">
        <span class="mu-caption">Ofertas</span><strong>{{ summary?.offers }}</strong>
      </div>
      <div class="mu-panel mu-ficha">
        <span class="mu-caption">Tiendas</span><strong>{{ summary?.stores }}</strong>
      </div>
    </div>

    <article v-for="(offer, index) in offers" :key="`${offer.url}-${index}`"
             class="mu-panel mu-oferta" :class="{ mejor: offer.best }">
      <div class="mu-oferta-cab">
        <h3 class="mu-mirable" tabindex="0" role="button"
            :title="`Mira ${offer.card_name}`"
            @click="emit('look', offer.card_name)"
            @keydown.enter="emit('look', offer.card_name)">{{ offer.card_name }}</h3>
        <span class="mu-precio">{{ formatAmount(offer.amount, offer.currency) }}</span>
      </div>
      <div>
        <span v-for="pill in offer.pills" :key="pill.text" class="mu-pill" :class="pill.kind">
          {{ pill.text }}
        </span>
        <span v-if="offer.best" class="mu-pill mejor">🐾 el mas barato</span>
      </div>
      <p v-if="offer.note" class="mu-caption">{{ offer.note }}</p>
      <a :href="offer.url" target="_blank" rel="noopener noreferrer">{{ offer.action }} →</a>
    </article>

    <p v-if="!offers.length && placeholder" class="mu-aviso">{{ placeholder }}</p>
  </section>
</template>

<style scoped>
.mu-fichas { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; margin: 14px 0; }
.mu-ficha { display: flex; flex-direction: column; gap: 4px; }
.mu-ficha strong { font-size: 1.4rem; }
.mu-oferta { margin-bottom: 12px; }
.mu-oferta.mejor { border-color: var(--mu-peri); }
.mu-oferta-cab { display: flex; justify-content: space-between; gap: 12px; align-items: baseline; flex-wrap: wrap; }
h3 { margin: 0; font-size: 1.05rem; }
/* El Cursor avisa que el Nombre hace algo antes de que nadie lo pulse. */
.mu-mirable { cursor: pointer; }
.mu-mirable:hover, .mu-mirable:focus-visible { text-decoration: underline dotted; }
.mu-precio { font-weight: 800; color: var(--mu-acento); }
</style>
