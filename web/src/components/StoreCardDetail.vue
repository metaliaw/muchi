<script setup>
/** Presenta una Oferta cuyo Catálogo vive fuera del Sitio de la Tienda. */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import * as api from '../api.js'
import { declaredStock } from '../search.js'

const props = defineProps({
  offer: { type: Object, required: true },
  game: { type: String, required: true },
})
const emit = defineEmits(['close'])

const card = ref({ ...props.offer.metadata })
const busy = ref(true)
const failed = ref('')

const HIDDEN_FIELDS = new Set([
  'functional_id', 'id', 'image', 'image_url', 'name', 'printed_name',
  'scryfall_uri', 'set_id', 'url',
])
const FIELD_NAMES = {
  artist: 'Artista', artist_ids: 'Identificadores de artista', booster: 'Disponible en sobres',
  border_color: 'Color del borde', card_back_id: 'Identificador del reverso',
  collector_number: 'Número de colección', color_identity: 'Identidad de color',
  color_indicator: 'Indicador de color', colors: 'Colores', condition: 'Estado',
  cmc: 'Valor de maná', converted_mana_cost: 'Valor de maná', digital: 'Carta digital',
  edition: 'Edición', finish: 'Acabado', finishes: 'Acabados disponibles',
  flavor_name: 'Nombre alternativo',
  flavor_text: 'Texto de ambientación', foil: 'Disponible foil', frame: 'Marco',
  frame_effects: 'Efectos del marco', full_art: 'Arte extendido', games: 'Formatos disponibles',
  game: 'Juego', hand_modifier: 'Modificador de mano', keywords: 'Palabras clave', lang: 'Idioma',
  language: 'Idioma', layout: 'Diseño', legalities: 'Legalidad por formato',
  life_modifier: 'Modificador de vida', loyalty: 'Lealtad', mana_cost: 'Coste de maná',
  mana_value: 'Valor de maná', oracle_text: 'Texto de la carta', oversized: 'Tamaño especial',
  penny_rank: 'Clasificación Penny Dreadful', pokemon_type: 'Tipo Pokémon', power: 'Fuerza',
  printed_text: 'Texto impreso',
  printed_type_line: 'Tipo impreso', produced_mana: 'Maná que produce', promo: 'Promocional',
  promo_types: 'Tipos de promoción', rarity: 'Rareza', released_at: 'Fecha de lanzamiento',
  reprint: 'Reimpresión', reserved: 'Lista reservada', security_stamp: 'Sello de seguridad',
  set: 'Código de edición', set_code: 'Código de edición', set_name: 'Edición',
  story_spotlight: 'Momento de la historia', textless: 'Sin texto', toughness: 'Resistencia',
  title: 'Título publicado', type: 'Tipo', type_line: 'Tipo', variant: 'Variante',
  variation: 'Variación', watermark: 'Marca de agua',
}

function nameField(key) {
  return FIELD_NAMES[key] || `Atributo adicional (${key.replaceAll('_', ' ')})`
}

const image = computed(() =>
  props.offer.image || card.value.image || card.value.image_url || '')
const name = computed(() =>
  card.value.printed_name || card.value.name || props.offer.card_name)
const quantity = computed(() => declaredStock(props.offer))
const stock = computed(() => {
  if (quantity.value !== null)
    return `${quantity.value} ${quantity.value === 1 ? 'unidad' : 'unidades'}`
  return props.offer.stock_label || 'Cantidad no informada'
})
const traits = computed(() => {
  const merged = {
    edition: props.offer.edition,
    language: props.offer.language,
    finish: props.offer.finish,
    condition: props.offer.metadata?.condition,
    ...card.value,
  }
  return Object.entries(merged)
    .filter(([key, value]) => !HIDDEN_FIELDS.has(key) && value !== '' && value != null)
    .map(([key, value]) => ({
      key,
      label: nameField(key),
      value: Array.isArray(value) ? value.join(', ') : String(value),
      wide: ['flavor_text', 'oracle_text'].includes(key),
    }))
})

function closeWithKey(event) {
  if (event.key === 'Escape') emit('close')
}

onMounted(async () => {
  document.addEventListener('keydown', closeWithKey)
  try {
    const found = await api.readCardMetadata({
      game: props.game,
      name: props.offer.card_name,
      language: props.offer.language || '',
      edition: props.offer.edition || '',
      foil: Boolean(props.offer.finish?.toLowerCase().includes('foil')),
    })
    card.value = { ...props.offer.metadata, ...found }
  } catch (error) {
    failed.value = error.message
  } finally {
    busy.value = false
  }
})
onUnmounted(() => document.removeEventListener('keydown', closeWithKey))
</script>

<template>
  <div class="mu-detalle-fondo" role="presentation" @click.self="emit('close')">
    <article class="mu-detalle" role="dialog" aria-modal="true" :aria-labelledby="`card-${offer.offer_id}`">
      <button class="mu-detalle__cerrar mu-ghost" type="button" aria-label="Cerrar detalle"
              @click="emit('close')">×</button>

      <div class="mu-detalle__imagen">
        <img v-if="image" :src="image" :alt="name" />
        <div v-else class="mu-detalle__sin-imagen">Imagen no disponible</div>
      </div>

      <div class="mu-detalle__contenido">
        <p class="mu-detalle__muchi"><span aria-hidden="true">🐱</span> Esta tienda está dentro de Muchi</p>
        <p class="mu-detalle__tienda">{{ offer.store }}</p>
        <h2 :id="`card-${offer.offer_id}`">{{ name }}</h2>

        <div class="mu-detalle__compra">
          <div><span>Precio</span><strong>{{ api.formatAmount(offer.amount, offer.currency) }}</strong></div>
          <div><span>Cantidad</span><strong>{{ stock }}</strong></div>
        </div>

        <p v-if="busy" class="mu-caption">Buscando las características de la carta…</p>
        <p v-else-if="failed" class="mu-aviso">{{ failed }} Se muestran los datos publicados por la tienda.</p>

        <dl v-if="traits.length" class="mu-detalle__atributos">
          <div v-for="trait in traits" :key="trait.key" :class="{ ancho: trait.wide }">
            <dt>{{ trait.label }}</dt>
            <dd>{{ trait.value }}</dd>
          </div>
        </dl>

        <div class="mu-detalle__acciones">
          <a v-if="offer.url" class="mu-boton" :href="offer.url" target="_blank"
             rel="noopener noreferrer">Ver catálogo en Moxfield</a>
          <button class="mu-ghost" type="button" @click="emit('close')">Volver a las ofertas</button>
        </div>
      </div>
    </article>
  </div>
</template>

<style scoped>
.mu-detalle-fondo {
  position: fixed; inset: 0; z-index: 50; padding: 24px;
  display: grid; place-items: center; overflow-y: auto;
  background: rgba(45, 34, 43, .72); backdrop-filter: blur(8px);
}
.mu-detalle {
  position: relative; width: min(980px, 100%); max-height: calc(100vh - 48px);
  display: grid; grid-template-columns: minmax(250px, 38%) 1fr; overflow: auto;
  background: var(--mu-blanco); border: 2px solid var(--mu-rosa-cl);
  border-radius: 28px; box-shadow: var(--mu-sombra);
}
.mu-detalle__cerrar {
  position: absolute; top: 14px; right: 14px; z-index: 1;
  width: 42px; height: 42px; padding: 0; font-size: 1.7rem; line-height: 1;
}
.mu-detalle__imagen { padding: 32px; background: var(--mu-niebla); display: grid; place-items: center; }
.mu-detalle__imagen img { width: 100%; max-height: 70vh; object-fit: contain; border-radius: 4.8%; box-shadow: var(--mu-sombra); }
.mu-detalle__sin-imagen { color: var(--mu-tinta-sw); text-align: center; }
.mu-detalle__contenido { padding: 36px 42px 40px; }
.mu-detalle__muchi { width: fit-content; margin: 0 52px 28px 0; padding: 7px 12px; border-radius: 999px; background: var(--mu-rosa-lav); font-weight: 700; }
.mu-detalle__tienda { margin: 0; color: var(--mu-acento); font-size: 1.25rem; font-weight: 800; }
h2 { margin: 2px 0 24px; color: var(--mu-tinta); font-size: clamp(2rem, 5vw, 3.6rem); line-height: .98; letter-spacing: -.045em; }
.mu-detalle__compra { display: grid; grid-template-columns: 1fr 1fr; gap: 1px; margin-bottom: 26px; overflow: hidden; border: 1px solid var(--mu-rosa-cl); border-radius: 16px; background: var(--mu-rosa-cl); }
.mu-detalle__compra div { padding: 14px 16px; background: var(--mu-papel); }
.mu-detalle__compra span, .mu-detalle__compra strong { display: block; }
.mu-detalle__compra span { color: var(--mu-tinta-sw); font-size: .82rem; }
.mu-detalle__compra strong { margin-top: 2px; color: var(--mu-acento); font-size: 1.25rem; }
.mu-detalle__atributos { display: grid; grid-template-columns: 1fr 1fr; gap: 0 22px; margin: 0; }
.mu-detalle__atributos div { padding: 10px 0; border-bottom: 1px solid var(--mu-rosa-cl); }
.mu-detalle__atributos .ancho { grid-column: 1 / -1; }
dt { color: var(--mu-tinta-sw); font-size: .78rem; }
dd { margin: 2px 0 0; white-space: pre-line; }
.mu-detalle__acciones { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 28px; }
@media (max-width: 700px) {
  .mu-detalle-fondo { padding: 0; place-items: stretch; }
  .mu-detalle { max-height: none; min-height: 100%; grid-template-columns: 1fr; border: 0; border-radius: 0; }
  .mu-detalle__imagen { padding: 24px 64px; }
  .mu-detalle__imagen img { max-height: 46vh; }
  .mu-detalle__contenido { padding: 26px 20px 36px; }
  .mu-detalle__atributos { grid-template-columns: 1fr; }
  .mu-detalle__atributos .ancho { grid-column: auto; }
}
</style>
