<script setup>
/** El Carrito en CLP: reparte la Lista entre Tiendas cuidando los Envíos. */
import { ref, watch } from 'vue'
import { formatClp, readCart } from '../api.js'

const props = defineProps({ searchId: { type: String, required: true } })

const shipping = ref(4000)
const plan = ref(null)
const error = ref('')
const open = ref(false)
const loading = ref(false)

async function refresh() {
  if (!open.value) return
  loading.value = true
  error.value = ''
  try {
    plan.value = await readCart(props.searchId, Math.max(0, Number(shipping.value) || 0))
  } catch (failure) {
    error.value = failure.message
  } finally {
    loading.value = false
  }
}

watch([open, shipping, () => props.searchId], refresh)
</script>

<template>
  <section class="mu-panel">
    <button class="mu-ghost" @click="open = !open">
      {{ open ? 'Cerrar Carrito' : 'Carrito en CLP' }}
    </button>

    <div v-if="open" class="mu-carro">
      <label>Envío por Tienda
        <input type="number" v-model="shipping" min="0" step="500" />
      </label>
      <p v-if="error" class="mu-aviso error">{{ error }}</p>
      <p v-else-if="loading" class="mu-caption">Calculando…</p>

      <template v-if="plan && !loading">
        <p class="mu-caption">
          Usa Ofertas sin alertas de Precio ni Stock agotado.
          <strong>Muchi Dólar: 1 USD = {{ formatClp(plan.muchi_dolar) }}</strong>, el Cambio
          de Muchi con Costos de Compra incluidos, no el del Mercado.
        </p>
        <p v-if="plan.converted_offers" class="mu-caption">
          {{ plan.converted_offers }} Ofertas en USD entraron convertidas.
        </p>
        <p class="mu-total">Total con Envíos: {{ formatClp(plan.total) }}</p>

        <div v-for="store in plan.stores" :key="store.store" class="mu-tienda">
          <h3>{{ store.store }}</h3>
          <p class="mu-caption">
            {{ store.cards }} Cartas · {{ formatClp(store.subtotal) }}
            · Envío {{ formatClp(plan.shipping_per_store) }}
          </p>
          <p v-for="line in store.lines" :key="`${line.card_name}-${line.url}`" class="mu-linea">
            <span>{{ line.quantity }}× {{ line.card_name }}</span>
            <a :href="line.url" target="_blank" rel="noopener noreferrer">
              {{ formatClp(line.subtotal) }}
            </a>
          </p>
        </div>

        <p v-if="plan.missing.length" class="mu-caption">
          Sin Oferta apta: {{ plan.missing.join(', ') }}
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
