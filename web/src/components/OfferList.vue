<script setup>
/** Las Ofertas, agrupadas por Tipo de Carta y por Precio dentro de cada una. */
import { computed, ref, watch } from 'vue'
import { formatAmount, formatClp } from '../api.js'
import {
  BY_EDITION, BY_PRICE, groupByCardType, pickable as canPick, spreadUnits,
} from '../search.js'

const emit = defineEmits(['look', 'confirm', 'cheap'])

// La más barata Merece un Comentario. Solo esa: Celebrar cada Oferta Sería no
// Celebrar ninguna. Pasar dos veces por la misma no Repite la Frase, así que
// Recorrer la Lista con el Mouse no Deja a Muchi hablando solo.
let cheered = ''
function cheerCheap(offer) {
  if (!offer.best || cheered === offer.offer_id) return
  cheered = offer.offer_id
  emit('cheap', offer)
}

// Cuántas Copias se Compran en cada Oferta. Cero es lo normal: de casi toda
// Oferta no se Compra nada. Las que el Reparto Recomienda nacen con su
// Cantidad puesta, y de ahí en adelante Manda quien Compra.
const units = defineModel('units', { type: Object, default: () => ({}) })

// Una Agotada no se Compra. Se Sigue Mostrando —su Precio Dice algo del
// Mercado— pero en gris y sin Selector: Ofrecerla sería Ofrecer una Compra
// que la Tienda ya Dijo que no Puede hacer.
const pickable = (offer) => canPick(offer)
const bought = (offer) => units.value[offer.offer_id] || 0

// Cuántas Copias de esta Carta Faltan por Elegir, sin Contar esta Oferta.
function missingFor(group, offer) {
  const total = askedFor(offer)
  const others = group.rows
    .filter((row) => row.offer_id !== offer.offer_id)
    .reduce((count, row) => count + bought(row), 0)
  return Math.max(1, total - others)
}

// Tocar la Oferta Pregunta si Está, y nada más. Comprar es una Decisión con
// Cantidad: para eso están el Más y el Menos, que no se Aprietan sin querer.
function touchOffer(offer, event) {
  if (!offer.offer_id || !pickable(offer)) return
  // El Nombre Abre la Carta y el Enlace Lleva a la Tienda: cada Control
  // Sigue Haciendo lo suyo, y solo el Resto de la Ficha Pregunta.
  if (event.target.closest('a, input, label, button, summary, details, .mu-mirable')) return
  emit('confirm', offer)
}

// Una Copia más, o una menos. Nunca bajo cero: un Carrito negativo no Existe.
function stepUnits(offer, step) {
  countUnits(offer, String(Math.max(0, bought(offer) + step)))
}

// Marcar una Oferta es Pedirle lo que Falta, no una Copia suelta: quien
// Tilda la única Tienda de una Lista de cuatro Quiere las cuatro.
function toggleOffer(group, offer, taken) {
  countUnits(offer, taken ? String(missingFor(group, offer)) : '0')
}

function countUnits(offer, written) {
  const value = Math.floor(Number(written))
  const clean = { ...units.value }
  if (written === '' || Number.isNaN(value) || value <= 0) delete clean[offer.offer_id]
  else clean[offer.offer_id] = Math.min(value, 999)
  units.value = clean
}

// Lo que identifica la Impresion en venta. Sin Edicion, Scryfall elige ella.
const printingOf = (offer) => ({
  name: offer.card_name,
  language: offer.language || '',
  edition: offer.edition || '',
  foil: Boolean(offer.finish && offer.finish.toLowerCase().includes('foil')),
  // La Foto llega como Campo propio de la Oferta. Buscarla solo en `metadata`
  // la Perdia entera: ahi solo cae lo que la Tienda manda de mas.
  image: offer.image || offer.metadata?.image || '',
  url: offer.metadata?.url || '',
})

const props = defineProps({
  items: { type: Array, default: () => [] },
  offers: { type: Array, default: () => [] },
  summary: { type: Object, default: null },
  notices: { type: Array, default: () => [] },
  placeholder: { type: String, default: '' },
  advertiseGroups: { type: Boolean, default: false },
})

// Cuántas Copias Pide la Lista de cada Carta. El Badge lo Muestra al lado de
// lo contado: "1 / 4" Dice de una vez que esta Tienda no Alcanza sola.
const asked = computed(() => Object.fromEntries(
  props.items.map((item) => [item.name.toLowerCase(), item.quantity])))
const askedFor = (offer) =>
  asked.value[(offer.card_type || offer.card_name || '').toLowerCase()] || 0

// Con qué Criterio Baja la Cantidad pedida sobre las Ofertas.
const criterion = ref(BY_PRICE)

const edition = ref('')
const editions = computed(() => [...new Set(props.offers
  .map((offer) => offer.edition)
  .filter(Boolean))].sort())
const editionOffers = computed(() => edition.value
  ? props.offers.filter((offer) => offer.edition === edition.value)
  : props.offers)
const variantOf = (offer) => {
  const name = offer.card_name.toLowerCase()
  if (name.includes('master ball pattern')) return 'master-ball'
  if (name.includes('poke ball pattern')) return 'poke-ball'
  return 'normal'
}
const variantNames = { normal: 'Normal', 'poke-ball': 'Poké Ball', 'master-ball': 'Master Ball' }
const variant = ref('')
const variants = computed(() => [...new Set(editionOffers.value.map(variantOf))])
const visibleOffers = computed(() => variant.value
  ? editionOffers.value.filter((offer) => variantOf(offer) === variant.value)
  : editionOffers.value)
watch(editions, (values) => {
  if (edition.value && !values.includes(edition.value)) edition.value = ''
})
watch(variants, (values) => {
  if (variant.value && !values.includes(variant.value)) variant.value = ''
})

// Un solo Tipo no es una Agrupación: mostrar un Encabezado para él sería
// repetir el Nombre que ya está en cada Oferta.
const groups = computed(() => groupByCardType(visibleOffers.value))
const grouped = computed(() => groups.value.length > 1)

// La Cantidad pedida se Reparte sola sobre lo que se Ve, de la barata a la
// cara. Lo último repartido se Guarda: mientras los Selectores Sigan igual a
// eso, nadie los Tocó y se Pueden Rehacer. Tocado uno, no se Pisa más —salvo
// que quien Compra Cambie el Criterio o el Filtro, que es Pedirlo de nuevo.
const spread = ref({})
function spreadNow(force = false) {
  if (!force && JSON.stringify(units.value) !== JSON.stringify(spread.value)) return
  spread.value = spreadUnits(groups.value, (group) => askedFor(group.rows[0]),
                             criterion.value)
  units.value = { ...spread.value }
}
watch([edition, variant, criterion], () => spreadNow(true))
watch(() => groups.value, () => spreadNow(), { immediate: true })
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

    <div v-if="offers.length" class="mu-filtros">
      <label class="mu-filtro">
        <span>Repartir por</span>
        <select v-model="criterion">
          <option :value="BY_PRICE">La más barata</option>
          <!-- Sin dos Ediciones no hay nada que Elegir, y el Criterio Sobra. -->
          <option v-if="editions.length > 1" :value="BY_EDITION">Una sola Edición</option>
        </select>
      </label>
    </div>

    <div v-if="editions.length > 1 || variants.length > 1" class="mu-filtros">
      <label v-if="editions.length > 1" class="mu-filtro">
        <span>Edición</span>
        <select v-model="edition">
          <option value="">Todas</option>
          <option v-for="value in editions" :key="value" :value="value">{{ value }}</option>
        </select>
      </label>
      <label v-if="variants.length > 1" class="mu-filtro">
        <span>Variante</span>
        <select v-model="variant">
          <option value="">Todas</option>
          <option v-for="value in variants" :key="value" :value="value">{{ variantNames[value] }}</option>
        </select>
      </label>
    </div>

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
    <h2 v-if="grouped || advertiseGroups" class="mu-grupo">{{ group.name }}
      <span class="mu-caption">{{ group.rows.length }} Ofertas</span>
    </h2>
    <slot v-if="advertiseGroups" name="advertisement" :group="group" />
    <article v-for="(offer, index) in group.rows" :key="`${offer.url}-${index}`"
             class="mu-panel mu-oferta"
             :class="{ mejor: offer.best, elegida: bought(offer) > 0,
                       agotada: offer.offer_id && !pickable(offer),
                       tomable: offer.offer_id && pickable(offer) }"
             @click="touchOffer(offer, $event)"
             @mouseenter="cheerCheap(offer)">
      <!-- Tildar es Decir «de acá me Llevo». El Reparto ya Tildó lo que
           Recomienda; una Agotada ni siquiera Lleva Casilla. -->
      <input v-if="offer.offer_id && pickable(offer)" type="checkbox" class="mu-elige"
             :checked="bought(offer) > 0"
             :aria-label="`Compra ${offer.card_name} en ${offer.store}`"
             @change="toggleOffer(group, offer, $event.target.checked)" />
      <span v-else-if="offer.offer_id" class="mu-elige mu-elige--fuera"
            aria-hidden="true" title="Agotada: no se puede comprar"></span>

      <!-- Cuántas Copias Salen de acá. Cero es lo normal, y el Total al lado
           Evita Contar de memoria cuántas Faltan. -->
      <div v-if="offer.offer_id && pickable(offer)" class="mu-copias"
           :class="{ vacia: !bought(offer) }">
        <button type="button" class="mu-copias__paso" :disabled="!bought(offer)"
                :aria-label="`Una Copia menos en ${offer.store}`"
                @click="stepUnits(offer, -1)">−</button>
        <label class="mu-copias__cuenta">
          <input type="number" min="0" max="999" step="1"
                 :value="bought(offer)"
                 :aria-label="`Copias de ${offer.card_name} en ${offer.store}`"
                 @input="countUnits(offer, $event.target.value)" />
          <span v-if="askedFor(offer)" class="mu-copias__total">/ {{ askedFor(offer) }}</span>
        </label>
        <button type="button" class="mu-copias__paso"
                :aria-label="`Una Copia más en ${offer.store}`"
                @click="stepUnits(offer, 1)">+</button>
        <span class="mu-copias__rotulo">Copias</span>
      </div>
      <span v-else-if="offer.offer_id" class="mu-copias mu-copias--fuera">
        <span class="mu-copias__rotulo">Agotada</span>
      </span>

      <div class="mu-oferta-cuerpo">
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
      <!-- Un solo Lugar no se Pliega: abrir un Detalle vacio no Muestra nada. -->
      <p v-if="offer.locations?.length === 1" class="mu-retiros">
        <span aria-hidden="true">📍</span> Retiro: {{ offer.locations[0] }}
      </p>
      <details v-else-if="offer.locations?.length" class="mu-retiros">
        <summary><span aria-hidden="true">📍</span> Retiro: {{ offer.locations[0] }}
          <span class="mu-retiros__mas">
            · +{{ offer.locations.length - 1 }}
            {{ offer.locations.length === 2 ? 'retiro' : 'retiros' }}
          </span>
        </summary>
        <ul>
          <li v-for="location in offer.locations" :key="location">{{ location }}</li>
        </ul>
      </details>
      <p v-if="offer.note" class="mu-caption">{{ offer.note }}</p>
      <a :href="offer.url" target="_blank" rel="noopener noreferrer">{{ offer.action }} →</a>
      </div>
    </article>
    </template>

    <p v-if="!offers.length && placeholder" class="mu-aviso">{{ placeholder }}</p>
  </section>
</template>

<style scoped>
/* Las Copias Abren la Oferta por la Izquierda: es lo primero que se Mira
   cuando se está Armando una Compra, y ahí el Número Cabe grande. */
/* Flex y no Grid: una Oferta sin Identificador no Lleva Ficha, y una Columna
   vacía le Comería el Ancho al Cuerpo. */
.mu-oferta { display: flex; align-items: flex-start; gap: 14px; }
.mu-elige {
  flex: none; width: 20px; height: 20px; margin: 26px 0 0; accent-color: var(--mu-acento);
  cursor: pointer;
}
/* El Hueco de una Agotada Guarda la Columna: sin él, su Ficha se Corre y las
   Ofertas Dejan de Alinearse entre sí. */
.mu-elige--fuera { display: block; cursor: default; }
/* Una Agotada se Sigue Viendo, apagada: su Precio Dice algo del Mercado,
   pero no es una Compra posible y no Debería Competir por la Atención. */
/* Se Puede Tocar para Sumar, y la Mano lo Dice antes que cualquier Cartel. */
.mu-oferta.tomable { cursor: pointer; }
.mu-oferta.tomable:hover { border-color: var(--mu-rosa); }
.mu-oferta.agotada { opacity: .55; }
.mu-oferta.agotada .mu-precio { color: var(--mu-tinta-sw); }
.mu-copias--fuera {
  flex: none; min-width: 4.6rem; padding: 10px 12px; border-radius: 16px;
  display: grid; align-content: center; justify-content: center;
  background: var(--mu-cond-bg); border: 1px dashed var(--mu-tinta-sw);
}
.mu-oferta-cuerpo { flex: 1; min-width: 0; }
.mu-copias {
  flex: none;
  display: grid; grid-template-columns: auto auto auto; justify-content: center;
  align-items: center; align-content: center; gap: 2px 6px;
  /* Un Ancho fijo Alinea todas las Fichas: con uno y con doce Dígitos, los
     Nombres de las Ofertas Empiezan en la misma Columna. */
  min-width: 4.6rem; padding: 8px 10px; border-radius: 16px;
  background: var(--mu-papel); border: 1px solid var(--mu-rosa-cl);
  cursor: pointer;
}
.mu-copias input {
  width: 3ch; padding: 0; border: 0; background: none; text-align: right;
  font-size: 1.5rem; font-weight: 800; color: var(--mu-acento);
  font-family: inherit;
}
/* Las Flechitas Roban el Ancho que el Número Necesita para Verse. */
.mu-copias input::-webkit-outer-spin-button,
.mu-copias input::-webkit-inner-spin-button { -webkit-appearance: none; margin: 0; }
.mu-copias input[type=number] { -moz-appearance: textfield; appearance: textfield; }
.mu-copias input:focus { outline: none; }
/* Cero es la Mayoría: la Ficha se Apaga para que Resalten las que sí Compran. */
.mu-copias.vacia { background: none; border-color: var(--mu-niebla); }
.mu-copias.vacia input { color: var(--mu-tinta-sw); }
.mu-copias:focus-within { border-color: var(--mu-acento); }
.mu-copias__cuenta { display: flex; align-items: baseline; gap: 4px; cursor: pointer; }
.mu-copias__total { font-size: 1.05rem; font-weight: 700; color: var(--mu-tinta-sw); }
/* El Más y el Menos Son el Camino corto; el Campo Sigue ahí para el Número
   que no se Alcanza a Golpes. */
.mu-copias__paso {
  width: 26px; height: 26px; padding: 0; border-radius: 50%;
  border: 1px solid var(--mu-rosa-cl); background: var(--mu-blanco);
  color: var(--mu-acento); font-size: 1.1rem; font-weight: 800; line-height: 1;
  cursor: pointer;
}
.mu-copias__paso:hover:not(:disabled) { background: var(--mu-rosa-cl); }
.mu-copias__paso:disabled { opacity: .35; cursor: default; }
.mu-copias__rotulo {
  grid-column: 1 / -1; text-align: center;
  font-size: .72rem; letter-spacing: .04em; text-transform: uppercase;
  color: var(--mu-tinta-sw);
}
.mu-lista { margin-bottom: 14px; }
.mu-lista h2 { margin-top: 0; }
.mu-lista-fila { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.mu-filtros { display: flex; gap: 14px; flex-wrap: wrap; margin: 14px 0; }
.mu-filtro { display: flex; align-items: center; gap: 8px; width: fit-content; font-weight: 700; }
.mu-filtro select { min-width: 120px; padding: 7px 30px 7px 9px; border: 1px solid var(--mu-linea); border-radius: 4px; background: var(--mu-papel); color: var(--mu-tinta); font: inherit; }
.mu-fichas { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; margin: 14px 0; }
.mu-ficha { display: flex; flex-direction: column; gap: 4px; }
.mu-ficha strong { font-size: 1.4rem; }
.mu-oferta { margin-bottom: 12px; }
.mu-oferta.mejor { border-color: var(--mu-peri); }
/* La Elegida se Nota sin Gritar: un Borde, no un Fondo entero. Va después de
   `mejor` porque Elegir otra es Contradecir la Recomendación, y lo que Manda
   en la Pantalla es la Decisión de quien Compra. */
.mu-oferta.elegida { border-color: var(--mu-acento); box-shadow: var(--mu-sombra-sw); }
.mu-oferta-cab { display: flex; justify-content: space-between; gap: 12px; align-items: baseline; flex-wrap: wrap; }
h3 { margin: 0; font-size: 1.05rem; }
/* El Encabezado separa una Carta de la siguiente sin robarle Peso al Precio. */
.mu-grupo { display: flex; justify-content: space-between; align-items: baseline;
            gap: 12px; flex-wrap: wrap; margin: 18px 0 8px; font-size: 1.1rem; }
/* El Cursor avisa que el Nombre hace algo antes de que nadie lo pulse. */
.mu-mirable { cursor: pointer; }
.mu-mirable:hover, .mu-mirable:focus-visible { text-decoration: underline dotted; }
.mu-precio { font-weight: 800; color: var(--mu-acento); }
.mu-retiros { width: fit-content; margin: 8px 0 4px; color: var(--mu-tinta); font-size: .88rem; }
.mu-retiros summary { cursor: pointer; }
.mu-retiros summary::marker { color: var(--mu-peri); }
.mu-retiros__mas { color: var(--mu-tinta-sw); }
.mu-retiros ul { margin: 4px 0 0; padding-left: 20px; }
</style>
