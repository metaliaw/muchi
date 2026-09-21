<script setup>
/** El Carrito en CLP: reparte la Lista entre Tiendas cuidando los Envíos. */
import { ref, watch } from 'vue'
import { formatClp, readCartWithUnits } from '../api.js'

const props = defineProps({
  searchId: { type: String, required: true },
  // El Carrito vuelve a la Carta pedida: el Modo le dice si hubo Derivados.
  match: { type: String, default: 'exact' },
  // Cuántas Copias se Compran en cada Oferta. Vacío Devuelve la Recomendación.
  units: { type: Object, default: () => ({}) },
})

// El Envío ya no se Pregunta: Muchi no Sabe cuánto Cobra cada Tienda, y un
// Número inventado en el Total Hacía Dudar de todo el resto. Este Valor no
// Aparece en ninguna Cifra mostrada; solo Sirve para que el Reparto Prefiera
// Juntar Cartas en pocas Tiendas, que es lo que alguien Haría igual.
const SHIPPING_GUESS = 4000
const plan = ref(null)
const error = ref('')
const open = ref(false)
const loading = ref(false)

async function refresh() {
  if (!open.value) return
  loading.value = true
  error.value = ''
  try {
    plan.value = await readCartWithUnits(
      props.searchId, SHIPPING_GUESS,
      Object.entries(props.units).map(([offer_id, count]) => ({ offer_id, units: count })),
      props.match)
  } catch (failure) {
    error.value = failure.message
  } finally {
    loading.value = false
  }
}

watch([open, () => props.searchId, () => props.match, () => props.units], refresh)
</script>

<template>
  <section class="mu-panel">
    <button class="mu-ghost" @click="open = !open">
      {{ open ? 'Cerrar Carrito' : 'Carrito en CLP' }}
    </button>

    <div v-if="open" class="mu-carro">
      <p v-if="error" class="mu-aviso error">{{ error }}</p>
      <p v-else-if="loading" class="mu-caption">Calculando…</p>

      <template v-if="plan && !loading">
        <!-- El Muchi Dólar Salía acá, al lado del Envío y del Total, y ahí
             Parecía parte del Despacho: quien Leía "1 USD = $1.000" junto a
             un Envío en Pesos Concluía otra cosa. El Cambio sigue Aplicándose
             igual; lo que se Fue es la Explicación en el Lugar equivocado. -->
        <p class="mu-caption">Usa Ofertas sin alertas de Precio ni Stock agotado.</p>
        <!-- El Total es el de las Cartas. Sumarle un Envío que Muchi Inventó
             Sería Dar por cierto un Número que ninguna Tienda Dijo. -->
        <p class="mu-total">Total de las Cartas: {{ formatClp(plan.cards_cost) }}</p>

        <div v-for="store in plan.stores" :key="store.store" class="mu-tienda">
          <h3>{{ store.store }}</h3>
          <p class="mu-caption">
            {{ store.cards }} Cartas · {{ formatClp(store.subtotal) }} · Envío aparte
          </p>
          <p v-for="line in store.lines" :key="`${line.card_name}-${line.url}`" class="mu-linea">
            <span>
              {{ line.quantity }}× {{ line.card_name }}
              <!-- Dos Líneas con el mismo Nombre y distinta Edición no son la
                   misma Compra: sin esto, el Carrito Parecía repetirse. -->
              <span v-if="line.edition || line.finish || line.condition" class="mu-caption">
                {{ [line.edition?.toUpperCase(), line.finish, line.condition]
                     .filter(Boolean).join(' · ') }}
              </span>
            </span>
            <a :href="line.url" target="_blank" rel="noopener noreferrer">
              {{ formatClp(line.subtotal) }}
            </a>
          </p>
        </div>

        <p v-if="plan.missing.length" class="mu-caption">
          Sin Oferta apta: {{ plan.missing.join(', ') }}
        </p>
        <!-- Entregar menos Copias de las pedidas sin Decirlo es Mentir el Total. -->
        <p v-for="row in plan.short" :key="row.card_name" class="mu-aviso">
          {{ row.card_name }}: Faltan {{ row.units }}
          {{ row.units === 1 ? 'Copia' : 'Copias' }} — las Tiendas contadas no
          Tienen tantas.
        </p>
      </template>
    </div>
  </section>
</template>

<style scoped>
.mu-carro { margin-top: 14px; display: flex; flex-direction: column; gap: 8px; }
label { display: flex; flex-direction: column; gap: 6px; max-width: 240px; font-size: .9rem; }
.mu-total { font-size: 1.3rem; font-weight: 800; color: var(--mu-acento); }
.mu-tienda { border-top: 1px solid var(--mu-rosa-cl); padding-top: 10px; }
.mu-tienda h3 { margin: 0; }
.mu-linea { display: flex; justify-content: space-between; gap: 12px; margin: 4px 0; }
</style>
