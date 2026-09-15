<script setup>
/** Las Ofertas, agrupadas por Tipo de Carta y por Precio dentro de cada una. */
import { computed } from 'vue'
import { formatAmount, formatClp } from '../api.js'
import { groupByCardType } from '../search.js'

const emit = defineEmits(['look'])

// Lo que identifica la Impresion en venta. Sin Edicion, Scryfall elige ella.
const printingOf = (offer) => ({
  name: offer.card_name,
  language: offer.language || '',
  edition: offer.edition || '',
  foil: Boolean(offer.finish && offer.finish.toLowerCase().includes('foil')),
  image: offer.metadata?.image || '',
  url: offer.metadata?.url || '',
})

const props = defineProps({
  items: { type: Array, default: () => [] },
  offers: { type: Array, default: () => [] },
  summary: { type: Object, default: null },
  notices: { type: Array, default: () => [] },
  placeholder: { type: String, default: '' },
})

// Un solo Tipo no es una Agrupación: mostrar un Encabezado para él sería
// repetir el Nombre que ya está en cada Oferta.
const groups = computed(() => groupByCardType(props.offers))
const grouped = computed(() => groups.value.length > 1)
</script>

<template>
  <section>
    <div v-if="items.length" class="mu-panel mu-lista">
      <h2>Cartas de la Lista</h2>
      <p v-for="item in items" :key="item.position" class="mu-lista-fila">
        <span class="mu-mirable" tabindex="0" role="button"
              :title="`Mira ${item.name}`"
            @mouseenter="emit('look', { name: item.name })"
            @focus="emit('look', { name: item.name })"
              @click="emit('look', { name: item.name })"
              @keydown.enter="emit('look', { name: item.name })">
          <strong>{{ item.quantity }}×</strong> {{ item.name }}
        </span>
        <span class="mu-caption">{{ item.offers }} Ofertas</span>
      </p>
    </div>

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
      <div v-if="grouped" class="mu-panel mu-ficha">
        <span class="mu-caption">Cartas</span><strong>{{ groups.length }}</strong>
      </div>
      <div class="mu-panel mu-ficha">
        <span class="mu-caption">Tiendas</span><strong>{{ summary?.stores }}</strong>
      </div>
    </div>

    <template v-for="group in groups" :key="group.card">
    <h2 v-if="grouped" class="mu-grupo">{{ group.name }}
      <span class="mu-caption">{{ group.rows.length }} Ofertas</span>
    </h2>
    <article v-for="(offer, index) in group.rows" :key="`${offer.url}-${index}`"
             class="mu-panel mu-oferta" :class="{ mejor: offer.best }">
      <div class="mu-oferta-cab">
        <h3 class="mu-mirable" tabindex="0" role="button"
            :title="`Mira ${offer.card_name}`"
          @mouseenter="emit('look', printingOf(offer))"
          @focus="emit('look', printingOf(offer))"
            @click="emit('look', printingOf(offer))"
            @keydown.enter="emit('look', printingOf(offer))">{{ offer.card_name }}</h3>
        <span class="mu-precio">{{ formatAmount(offer.amount, offer.currency) }}</span>
      </div>
      <div>
        <span v-for="pill in offer.pills" :key="pill.text" class="mu-pill" :class="pill.kind">
          {{ pill.text }}
        </span>
        <span v-if="offer.best" class="mu-pill mejor">🐾 el mas barato</span>
      </div>
      <details v-if="offer.locations?.length" class="mu-retiros">
        <summary>Retiro: {{ offer.locations[0] }}
          <span v-if="offer.locations.length > 1">
            · +{{ offer.locations.length - 1 }}
            {{ offer.locations.length === 2 ? 'retiro' : 'retiros' }}
          </span>
        </summary>
        <ul v-if="offer.locations.length > 1">
          <li v-for="location in offer.locations" :key="location">{{ location }}</li>
        </ul>
      </details>
      <p v-if="offer.note" class="mu-caption">{{ offer.note }}</p>
      <a :href="offer.url" target="_blank" rel="noopener noreferrer">{{ offer.action }} →</a>
    </article>
    </template>

    <p v-if="!offers.length && placeholder" class="mu-aviso">{{ placeholder }}</p>
  </section>
</template>

<style scoped>
.mu-lista { margin-bottom: 14px; }
.mu-lista h2 { margin-top: 0; }
.mu-lista-fila { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.mu-fichas { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; margin: 14px 0; }
.mu-ficha { display: flex; flex-direction: column; gap: 4px; }
.mu-ficha strong { font-size: 1.4rem; }
.mu-oferta { margin-bottom: 12px; }
.mu-oferta.mejor { border-color: var(--mu-peri); }
.mu-oferta-cab { display: flex; justify-content: space-between; gap: 12px; align-items: baseline; flex-wrap: wrap; }
h3 { margin: 0; font-size: 1.05rem; }
/* El Encabezado separa una Carta de la siguiente sin robarle Peso al Precio. */
.mu-grupo { display: flex; justify-content: space-between; align-items: baseline;
            gap: 12px; flex-wrap: wrap; margin: 18px 0 8px; font-size: 1.1rem; }
/* El Cursor avisa que el Nombre hace algo antes de que nadie lo pulse. */
.mu-mirable { cursor: pointer; }
.mu-mirable:hover, .mu-mirable:focus-visible { text-decoration: underline dotted; }
.mu-precio { font-weight: 800; color: var(--mu-acento); }
.mu-retiros { width: fit-content; margin: 6px 0 4px; color: var(--mu-tinta-sw); font-size: .82rem; }
.mu-retiros summary { cursor: pointer; }
.mu-retiros summary::marker { color: var(--mu-peri); }
.mu-retiros ul { margin: 4px 0 0; padding-left: 20px; }
</style>
