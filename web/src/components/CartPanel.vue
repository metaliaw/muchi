<script setup>
/** El Carrito en CLP: reparte la Lista entre Tiendas cuidando los Envíos. */
import { computed, ref, watch } from 'vue'
import { formatClp, readCartWithUnits } from '../api.js'

const props = defineProps({
  searchId: { type: String, required: true },
  // El Carrito vuelve a la Carta pedida: el Modo le dice si hubo Derivados.
  match: { type: String, default: 'exact' },
  // Cuántas Copias se Compran en cada Oferta. Vacío Devuelve la Recomendación.
  units: { type: Object, default: () => ({}) },
  // MUCHI Abierto se Queda con el Borde de abajo en Móvil. El Botón del
  // Carrito Espera a que MUCHI se Guarde en vez de Pisarle la Barra.
  docked: { type: Boolean, default: false },
  // La Dirección la sirve el Servidor, igual que en el Aviso de la Comunidad.
  repositoryUrl: { type: String, default: '' },
  // Con la Compra lista el Aviso Sobra. Lo Dice el Servidor: el Día que Llegue,
  // Retirarlo es una Línea en el Entorno y no una Versión nueva del Front.
  ready: { type: Boolean, default: false },
})

// El Envío ya no se Pregunta: MUCHI no Sabe cuánto Cobra cada Tienda, y un
// Número inventado en el Total Hacía Dudar de todo el resto. Este Valor no
// Aparece en ninguna Cifra mostrada; solo Sirve para que el Reparto Prefiera
// Juntar Cartas en pocas Tiendas, que es lo que alguien Haría igual.
const SHIPPING_GUESS = 4000
const feedbackUrl = computed(() => `${props.repositoryUrl}/issues/new`)
const plan = ref(null)
const error = ref('')
// Quién Está abierto lo Sabe la Página entera: el Cajón y el Muelle se
// Reparten la misma Esquina, y abiertos los dos uno Tapa al otro.
const open = defineModel('open', { type: Boolean, default: false })
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
  <!-- El Carrito Vive en un Cajón a la Derecha. Cerrado es un Botón redondo,
       igual que MUCHI: la Lista se Lee entera y el Total Espera al Lado. -->
  <aside class="mu-carrito" :class="{ abierto: open, ocupado: docked }">
    <button class="mu-carrito__tirador" type="button"
            :aria-expanded="open" @click="open = !open"
            :aria-label="open ? 'Cerrar el carrito' : 'Abrir el carrito'">
      <span class="mu-carrito__icono" aria-hidden="true">🛒</span>
      <span class="mu-carrito__dicho">Carrito en CLP</span>
      <span class="mu-carrito__flecha" aria-hidden="true">✕</span>
    </button>

    <div class="mu-carrito__cuerpo">
      <div v-if="open" class="mu-carro">
        <!-- El Carrito Reparte y Suma, pero todavía no Compra: Decirlo acá
             Evita que alguien Espere un Botón de Pagar que no Existe. Y quien
             se Topa con algo raro Tiene dónde Contarlo sin salir a buscarlo. -->
        <p v-if="!ready" class="mu-aviso mu-obra">
          <span aria-hidden="true">⚠️</span>
          Todavía estamos trabajando en la compra: por ahora el carrito reparte
          tu lista entre tiendas y te deja los enlaces para comprar en cada una.
          <a v-if="repositoryUrl" :href="feedbackUrl"
             target="_blank" rel="noopener noreferrer">Cuéntanos qué te pasó</a>
          <span v-else>Cuéntanos cualquier cosa que veas rara.</span>
        </p>
        <p v-if="error" class="mu-aviso error">{{ error }}</p>
        <p v-else-if="loading" class="mu-caption">Calculando…</p>

        <template v-if="plan && !loading">
          <!-- El MUCHI Dólar Salía acá, al lado del Envío y del Total, y ahí
               Parecía parte del Despacho: quien Leía "1 USD = $1.000" junto a
               un Envío en Pesos Concluía otra cosa. El Cambio sigue Aplicándose
               igual; lo que se Fue es la Explicación en el Lugar equivocado. -->
          <p class="mu-caption">Usa ofertas sin alertas de precio ni stock agotado.</p>
          <!-- El Total es el de las Cartas. Sumarle un Envío que MUCHI Inventó
               Sería Dar por cierto un Número que ninguna Tienda Dijo. -->
          <p class="mu-total">Total de las cartas: {{ formatClp(plan.cards_cost) }}</p>

          <div v-for="store in plan.stores" :key="store.store" class="mu-tienda">
            <h3>{{ store.store }}</h3>
            <p class="mu-caption">
              {{ store.cards }} cartas · {{ formatClp(store.subtotal) }} · Envío aparte
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
            Sin oferta apta: {{ plan.missing.join(', ') }}
          </p>
          <!-- Entregar menos Copias de las pedidas sin Decirlo es Mentir el Total. -->
          <p v-for="row in plan.short" :key="row.card_name" class="mu-aviso">
            {{ row.card_name }}: faltan {{ row.units }}
            {{ row.units === 1 ? 'copia' : 'copias' }} — las tiendas contadas no
            tienen tantas.
          </p>
        </template>
      </div>
    </div>
  </aside>
</template>

<style scoped>
/* Cerrado: un Botón redondo pegado al Borde derecho, sobre la Lista. */
.mu-carrito {
  position: fixed; z-index: 30;
  right: 16px; bottom: calc(16px + env(safe-area-inset-bottom, 0px));
  display: flex; flex-direction: column;
  background: var(--mu-papel);
  border: 2px solid var(--mu-rosa-cl);
  border-radius: 999px;
  box-shadow: var(--mu-sombra);
}
.mu-carrito__tirador {
  display: flex; align-items: center; justify-content: center;
  width: 54px; height: 54px; padding: 0;
  border: 0; border-radius: 999px; background: none;
  color: var(--mu-tinta); font: inherit; font-size: 1.5rem; font-weight: 700;
  cursor: pointer;
}
.mu-carrito:not(.abierto) .mu-carrito__dicho,
.mu-carrito:not(.abierto) .mu-carrito__flecha { display: none; }
.mu-carrito__cuerpo { display: none; }
/* Abierto: una Barra lateral de Alto completo. El Total y el Reparto se Leen
   sin Perder de vista las Ofertas que los Arman. */
.mu-carrito.abierto {
  top: 0; right: 0; bottom: 0; width: min(380px, 92vw);
  padding: 8px 14px 14px;
  border-width: 0 0 0 2px; border-radius: 18px 0 0 18px;
}
.mu-carrito.abierto .mu-carrito__tirador {
  width: 100%; height: auto; padding: 8px 4px;
  justify-content: space-between; gap: 12px;
  border-radius: 12px; font-size: 1rem; text-align: left;
}
.mu-carrito.abierto .mu-carrito__dicho {
  flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.mu-carrito.abierto .mu-carrito__cuerpo { display: block; flex: 1; overflow-y: auto; }
.mu-obra { margin: 0; font-size: .9rem; }
.mu-carro { margin-top: 14px; display: flex; flex-direction: column; gap: 8px; }
label { display: flex; flex-direction: column; gap: 6px; max-width: 240px; font-size: .9rem; }
.mu-total { font-size: 1.3rem; font-weight: 800; color: var(--mu-acento); }
.mu-tienda { border-top: 1px solid var(--mu-rosa-cl); padding-top: 10px; }
.mu-tienda h3 { margin: 0; }
.mu-linea { display: flex; justify-content: space-between; gap: 12px; margin: 4px 0; }

/* En Móvil el Botón se Pone al lado del de MUCHI, en la misma Esquina, y el
   Cajón Toma la Pantalla entera: 380px sobre un Teléfono no Dejan Lista
   detrás que Mirar. */
@media (max-width: 800px) {
  .mu-carrito:not(.abierto) {
    right: calc(12px + 54px + 10px);
    bottom: calc(12px + env(safe-area-inset-bottom, 0px));
  }
  /* MUCHI abierto se Queda con el Borde entero: el Botón Espera su Turno. */
  .mu-carrito.ocupado:not(.abierto) { display: none; }
  .mu-carrito.abierto {
    width: 100%; border-radius: 0; border-left-width: 0;
    padding-bottom: calc(14px + env(safe-area-inset-bottom, 0px));
  }
}
</style>
