<script setup>
/** Muchi Presenta Búsquedas y Resultados persistidos por la API. */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import * as api from './api.js'
import { useSearch } from './search.js'
import MuchiPanel from './components/MuchiPanel.vue'
import CardLookup from './components/CardLookup.vue'
import CardArt from './components/CardArt.vue'
import CommunityPanel from './components/CommunityPanel.vue'
import SiteLinks from './components/SiteLinks.vue'
import SearchForm from './components/SearchForm.vue'
import SearchProgress from './components/SearchProgress.vue'
import OfferList from './components/OfferList.vue'
import CartPanel from './components/CartPanel.vue'
import SourcesPanel from './components/SourcesPanel.vue'
import AdSpot from './components/AdSpot.vue'
import GoogleAdsense from './components/GoogleAdsense.vue'
import SponsorSpot from './components/SponsorSpot.vue'

const THEME_KEY = 'muchi_tema'

// El Ritmo lo manda el Servidor; este es el mismo de config/api.defaults.yaml,
// para los milisegundos que van entre que arranca la Página y llega la Config.
const config = ref({ poll_seconds: 3, adsense_client: 'ca-pub-6368656861543000' })
const book = ref(null)
const games = ref([])
const game = ref('')
const dark = ref(localStorage.getItem(THEME_KEY) === 'oscuro')
const message = ref(null)

// La Búsqueda vive en su propio Módulo: sus Refs, su Limpieza y la Mezcla de
// las Páginas se declaran una sola vez ahí.
const {
  id: searchId, state, items, offers, summary, notices, cursor, hasMore,
  checked, unavailable, start: startSearch, applyState, applyResults,
} = useSearch()
// El Muelle nace Cerrado: quien Busca quiere ver Ofertas, y Muchi y la Carta
// Esperan a un Toque. Solo Existe en Móvil; en Escritorio el Lateral los Muestra.
const dockOpen = ref(false)
const error = ref('')
const pending = ref(null)
const lookupText = ref('')
// El Modo viaja en la URL junto a la Búsqueda: recargar no debe reagrupar las
// mismas Ofertas de otra manera ni mandar Derivados al Carrito.
const match = ref(new URLSearchParams(location.search).get('match') === 'includes'
  ? 'includes' : 'exact')
// La Carta que se mira: un Nombre, y el Idioma en que se escribió.
const watched = ref(null)
const history = ref(JSON.parse(localStorage.getItem('muchi_historial') || '[]'))

let timer = null
let refreshing = false

const busy = computed(() => Boolean(
  state.value && (!state.value.done || hasMore.value) && !unavailable.value
))
// Fuera de Producción el Algoritmo Decide igual, pero AdSpot Dibuja
// un Placeholder en vez del Anuncio: así se Prueba la Elección sin Google.
const googleReady = computed(() =>
  config.value.environment !== 'production'
  || Boolean(config.value.adsense_client && config.value.adsense_slot)
)
const sponsorReady = computed(() => Boolean(config.value.sponsor_name && config.value.sponsor_url))

function hashSearch(id) {
  return [...id].reduce((value, letter) => ((value * 31) + letter.charCodeAt(0)) >>> 0, 0)
}

const showSponsor = computed(() => sponsorReady.value && hashSearch(searchId.value) % 4 === 0)
const advertiseSections = computed(() => match.value === 'includes' || game.value === 'pokemon')

const placeholder = computed(() => {
  if (unavailable.value) return 'No hay Ofertas recibidas para mostrar.'
  if (state.value?.done) return 'No hay Ofertas para mostrar.'
  if (state.value?.status === 'queued')
    return 'Búsqueda en Cola. Esperando que el Servicio la procese.'
  if (state.value)
    return 'Consultando Ofertas. Una Carta puede tardar varios minutos; los Resultados aparecen cuando la API termina de consultarla.'
  return ''
})

function say(text, mood = 'happy') {
  message.value = { text, state: mood }
}

function applyTheme() {
  document.documentElement.dataset.tema = dark.value ? 'oscuro' : 'claro'
  localStorage.setItem(THEME_KEY, dark.value ? 'oscuro' : 'claro')
}
watch(dark, applyTheme)
watch(game, () => {
  watched.value = null
})

// La Búsqueda nombra su Juego; el Selector lo sigue. Vacío no dice nada.
function playGame(named) {
  if (named) game.value = named
}

function remember(id, label) {
  const rest = history.value.filter((entry) => entry.id !== id)
  history.value = [{ id, label: label || id }, ...rest].slice(0, 20)
  localStorage.setItem('muchi_historial', JSON.stringify(history.value))
}

function selectSearch(id, initialState = null, initialItems = []) {
  startSearch(id, initialState, initialItems)
  const url = new URL(location.href)
  url.searchParams.set('search', id)
  url.searchParams.set('match', match.value)
  history.value.length && window.history.replaceState({}, '', url)
  refresh()
}

async function submit(text) {
  pending.value = { text, game: game.value, key: api.newKey(), match: match.value }
  await send()
}

function loadCard(canonicalName) {
  lookupText.value = canonicalName
}

// Del Buscador llega un Nombre; de una Oferta, la Impresion entera.
function lookAtCard(card) {
  watched.value = card
}

// A quien mira las Estadísticas, Muchi lo saluda como se merece. La Frase
// sale al azar del Catálogo, igual que las de la Luz y las de las Caricias.
function sayNerd() {
  const rows = book.value?.nerd
  if (!rows?.length) return
  const said = rows[Math.floor(Math.random() * rows.length)]
  say(said.text, said.state)
}

// Muchi no completa el Campo: dice lo que vio y quien escribe decide.
function suggestNames(names) {
  const [first, ...rest] = names
  say(rest.length ? `¿Buscabas «${first}»? También veo ${rest.map((n) => `«${n}»`).join(' y ')}.`
                  : `¿Buscabas «${first}»?`, 'talk')
}

async function send() {
  if (!pending.value) return
  error.value = ''
  try {
    const reply = await api.createSearch(
      pending.value.text, pending.value.game, pending.value.key, pending.value.match
    )
    remember(reply.state.id, reply.label)
    pending.value = null
    selectSearch(reply.state.id, reply.state, reply.items || [])
    say('¡Miau! Ya salí a buscar', 'happy')
  } catch (failure) {
    error.value = failure.message
    // Un Rechazo no se reintenta: el Envío pendiente se descarta.
    if (!failure.retriable) pending.value = null
    say(failure.message, 'angry')
  }
}

async function refresh() {
  if (!searchId.value || unavailable.value || refreshing) return
  refreshing = true
  try {
    const reply = await api.readSearch(searchId.value, cursor.value, match.value)
    playGame(applyState(reply.state))
    playGame(applyResults(reply))
    checked.value = new Date().toISOString().slice(11, 19) + ' UTC'
    remember(reply.state.id)
    if (reply.state.done && !reply.has_more) {
      stopPolling()
    }
  } catch (failure) {
    if (failure.retriable) {
      // Los Resultados recibidos se conservan y se reintenta la Consulta.
      error.value = failure.message
    } else {
      unavailable.value = failure.message
      stopPolling()
    }
  } finally {
    refreshing = false
  }
}

// Mirar una Carta ya no Abre el Muelle: la Carta se Muestra sola arriba de la
// Lista, y Muchi encima de ella sería un Gato tapando lo que le pediste ver.
// Se Abre solo con un Toque, o cuando algo Salió mal y hay que Contarlo. Lo
// demás que Muchi Dice Cabe en la Barra, sin Robarle la Pantalla a nadie.
watch(message, (said) => { if (said?.state === 'angry') dockOpen.value = true })

async function cancel() {
  try {
    const reply = await api.cancelSearch(searchId.value, api.newKey())
    playGame(applyState(reply.state))
    say('Ya paré de buscar', 'idle')
  } catch (failure) {
    error.value = failure.message
  }
}

function startPolling() {
  stopPolling()
  const seconds = config.value.poll_seconds || 3
  timer = setInterval(refresh, seconds * 1000)
}
function stopPolling() {
  if (timer) clearInterval(timer)
  timer = null
}

watch(busy, (value) => (value ? startPolling() : stopPolling()))

onMounted(async () => {
  applyTheme()
  try {
    const supported = await api.readSupportedGames()
    games.value = supported.games || []
    game.value = games.value[0]?.reference_key || ''
    ;[config.value, book.value] = await Promise.all([api.readConfig(), api.readMuchi()])
  } catch (failure) {
    error.value = failure.message
  }
  if (searchId.value) refresh()
})
onUnmounted(stopPolling)
</script>

<template>
  <GoogleAdsense :client="config.adsense_client" />

  <header class="mu-hero">
    <div class="mu-hero__nombre">
      <h1><a href="/">🐱 Muchi</a></h1>
      <p>Busca Cartas y cotiza tu Lista</p>
    </div>
    <SiteLinks
      :donation-url="config.donation_url"
      :socials="config.socials || []"
    />
  </header>

  <main class="mu-grilla">
    <!-- El Aviso se Lee una vez y se Queda quieto. Muchi y la Carta Acompañan
         el Recorrido: Flotan juntos en Escritorio, y en Móvil Muchi Espera en
         la Esquina mientras la Carta se Pega arriba de la Lista. -->
    <div class="mu-abierto">
      <CommunityPanel :repository-url="config.repository_url" />
    </div>

    <div class="mu-flotante">
      <div class="mu-muelle" :class="{ abierto: dockOpen, dijo: Boolean(message) }">
        <button class="mu-muelle__tirador" type="button"
                :aria-expanded="dockOpen" @click="dockOpen = !dockOpen"
                :aria-label="dockOpen ? 'Guardar a Muchi' : 'Llamar a Muchi'">
          <span class="mu-muelle__gato" aria-hidden="true">🐱</span>
          <span class="mu-muelle__dicho">{{ message?.text || 'Muchi' }}</span>
          <span class="mu-muelle__flecha" aria-hidden="true">{{ dockOpen ? '▼' : '▲' }}</span>
        </button>
        <div class="mu-muelle__cuerpo">
          <MuchiPanel :book="book" v-model:dark="dark" :message="message" />
        </div>
      </div>

      <div class="mu-tarjeta" :class="{ vacia: !watched }">
        <CardArt :card="watched" :game="game" />
      </div>
    </div>

    <div class="mu-columna">
      <SearchForm
        v-model:text="lookupText"
        v-model:game="game"
        v-model:match="match"
        :games="games"
        :busy="busy" :pending="Boolean(pending)" :error="error"
        :limits="config.limits"
        @search="submit" @retry="send" @resume="selectSearch"
      >
        <template #lookup>
          <CardLookup
            v-if="game"
            :game="game"
            @found="loadCard"
            @failed="(text) => say(text, 'angry')"
            @suggest="suggestNames"
            @look="lookAtCard"
          />
        </template>
      </SearchForm>

      <section v-if="history.length" class="mu-panel">
        <details>
          <summary>Búsquedas Recientes</summary>
          <p class="mu-caption">Guarda el Enlace para retomarlas después.</p>
          <p v-for="entry in history" :key="entry.id" class="mu-historia">
            <button class="mu-ghost" @click="selectSearch(entry.id)">Abrir</button>
            <span>{{ entry.label }}</span>
            <a :href="`?search=${encodeURIComponent(entry.id)}`">Enlace</a>
          </p>
        </details>
      </section>

      <SearchProgress
        v-if="state" :state="state" :checked="checked"
        :poll-seconds="config.poll_seconds" :unavailable="unavailable"
        @cancel="cancel"
      >
        <SponsorSpot
          v-if="(showSponsor || !googleReady) && (!advertiseSections || !offers.length)"
          :key="`promo-${searchId}`"
          :searching="!state.done"
          :sponsor-name="showSponsor ? config.sponsor_name : ''"
          :sponsor-text="showSponsor ? config.sponsor_text : ''"
          :sponsor-url="showSponsor ? config.sponsor_url : ''"
        />
        <AdSpot
          v-else-if="!advertiseSections || !offers.length"
          :key="`google-${searchId}`"
          :environment="config.environment"
          :client="config.adsense_client"
          :slot="config.adsense_slot"
        />
      </SearchProgress>

      <OfferList
        v-if="state" :items="items" :offers="offers" :summary="summary"
        :notices="notices" :placeholder="placeholder"
        :advertise-groups="advertiseSections"
        @look="lookAtCard"
      >
        <template #advertisement="{ group }">
          <SponsorSpot
            v-if="showSponsor || !googleReady"
            :key="`promo-${searchId}-${group.card}`"
            :searching="!state.done"
            :sponsor-name="showSponsor ? config.sponsor_name : ''"
            :sponsor-text="showSponsor ? config.sponsor_text : ''"
            :sponsor-url="showSponsor ? config.sponsor_url : ''"
          />
          <AdSpot
            v-else
            :key="`google-${searchId}-${group.card}`"
            :environment="config.environment"
            :client="config.adsense_client"
            :slot="config.adsense_slot"
          />
        </template>
      </OfferList>

      <CartPanel v-if="offers.length" :search-id="searchId" :match="match" />
      <SourcesPanel @nerd="sayNerd" />
    </div>
  </main>

</template>

<style scoped>
.mu-hero {
  display: flex; align-items: center; justify-content: space-between;
  gap: 16px; flex-wrap: wrap;
  background: linear-gradient(135deg, var(--mu-acento), var(--mu-peri));
  color: #fff; padding: 26px 24px; border-radius: 0 0 28px 28px;
  box-shadow: var(--mu-sombra);
}
/* El Nombre Manda a la Izquierda; los Enlaces Acompañan a la Derecha y, cuando
   no Caben, Bajan a su propia Línea sin Perder ese Lado. La Regla Nombra al
   `nav` y no al último Hijo: sin Redes configuradas no hay `nav`, y el Nombre
   se Iba a la Derecha él solo. */
.mu-hero__nombre { min-width: 0; }
.mu-hero > nav { margin-left: auto; }
.mu-hero h1 { margin: 0; font-size: 2.4rem; font-weight: 800; letter-spacing: -1px; }
.mu-hero h1 a { color: inherit; text-decoration: none; }
.mu-hero p { margin: 4px 0 0; opacity: .9; }
.mu-grilla {
  display: grid; grid-template-columns: 260px 1fr; gap: 16px 20px;
  grid-template-areas: "abierto lista" "flotante lista";
  /* La Lista Cruza las dos Filas. Sin esto, el Alto extra de las Ofertas
     Engorda la Fila del Aviso y Muchi Arranca más abajo. */
  grid-template-rows: max-content 1fr;
  max-width: 1100px; margin: 22px auto; padding: 0 16px; align-items: start;
}
.mu-abierto {
  grid-area: abierto;
  /* Un Anuncio que todavía Aterriza aquí no puede Abrir la Fila: Muchi
     Empieza debajo del Aviso, y cada Pixel de más lo Empuja fuera. */
  max-height: 240px;
  overflow: hidden;
}
/* Muchi y la Carta Acompañan el Recorrido de la Lista. Flotan juntos, en un
   solo Bloque: dos Pegados por separado se Taparían uno al otro al Bajar. */
.mu-flotante {
  grid-area: flotante; position: sticky; top: 16px;
  display: flex; flex-direction: column; gap: 16px;
  /* El Bloque nunca Pasa del Alto de la Ventana. Si algo le Crece adentro
     —Auto Ads Mete Anuncios donde Encuentra un Hueco— Muchi Quedaría empujado
     fuera de la Pantalla, Flotando debajo del Borde. */
  max-height: calc(100vh - 32px);
  overflow-y: auto;
}
/* Lo que Muchi no Puso ahí va al Final del Bloque, nunca delante de Muchi ni
   de la Carta. No se Esconde: se Ordena. */
.mu-flotante > :not(.mu-muelle):not(.mu-tarjeta) { order: 9; }
/* Un Anuncio suelto en la Rejilla no Abre una Fila encima de Muchi. El Slot
   de las Ofertas Vive dentro de AdSpot, no como Hijo de la Grilla. */
.mu-grilla > ins,
.mu-grilla > .google-auto-placed {
  grid-column: 1 / -1;
  max-height: 0;
  overflow: hidden;
  margin: 0;
  padding: 0;
}
/* En Escritorio el Muelle es un Panel más: sin Barra ni Tirador. */
.mu-muelle, .mu-muelle__cuerpo { display: contents; }
.mu-columna { grid-area: lista; display: flex; flex-direction: column; gap: 16px; }
.mu-historia { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
summary { cursor: pointer; font-weight: 600; }
/* En Escritorio el Muelle es el Lateral de siempre: sin Barra ni Tirador. */
.mu-muelle__tirador { display: none; }
/* En Móvil la Grilla es una sola Columna: el Lateral Suelta el Flote
   para no Tapar el Contenido al Bajar, y Muchi y la Carta Bajan al Muelle.
   Dos Columnas de verdad dejarían cada Oferta en 200px: el Precio y las
   Pastillas se Parten, y la Carta se Mira con lupa. */
@media (max-width: 800px) {
  /* La Carta Manda arriba y la Lista Corre debajo: la Pantalla se Parte en dos
     y quien Mira una Impresión sigue viendo los Precios. El Aviso Cierra la
     Página: se Lee una vez, y no antes que el Buscador. */
  .mu-grilla {
    grid-template-columns: 1fr;
    grid-template-areas: "carta" "lista" "abierto";
  }
  /* El Bloque flotante se Desarma: Muchi se Va a la Esquina por su cuenta y la
     Carta se Pega arriba de la Lista. */
  .mu-flotante { display: contents; }
  /* El Muelle Vuelve a ser un Bloque para poder Fijarse al pie. */
  .mu-muelle { display: flex; flex-direction: column; }
  .mu-muelle__cuerpo { display: none; flex-direction: column; gap: 16px; }
  .mu-tarjeta {
    position: sticky; top: 0; z-index: 10;
    margin: 0 -16px; padding: 0 16px 8px; background: var(--mu-papel);
  }
  /* Sin Carta mirada no hay nada que Fijar: la Lista se Queda con la Pantalla. */
  .mu-tarjeta.vacia { display: none; }
  .mu-muelle {
    /* `top` Vuelve a auto: Heredado del Lateral pegajoso, Estiraba el Muelle
       de Borde a Borde y Tapaba la Página entera. */
    position: fixed; top: auto; left: 0; right: 0; bottom: 0; z-index: 20; gap: 10px;
    padding: 8px 12px calc(8px + env(safe-area-inset-bottom, 0px));
    background: var(--mu-papel);
    border-top: 2px solid var(--mu-rosa-cl);
    border-radius: 18px 18px 0 0;
    box-shadow: 0 -6px 20px rgba(110, 90, 104, .22);
  }
  .mu-muelle__tirador {
    display: flex; justify-content: space-between; align-items: center; gap: 12px;
    width: 100%; padding: 8px 4px; border: 0; border-radius: 12px;
    background: none; color: var(--mu-tinta); font: inherit; font-weight: 700;
    cursor: pointer; text-align: left;
  }
  .mu-muelle__dicho { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  /* Cerrado no es una Barra: es un Botón redondo en la Esquina. Mientras hay
     una Carta a la Vista, Muchi Espera ahí en vez de Ocupar un Borde entero. */
  .mu-muelle:not(.abierto) {
    left: auto; right: 12px; width: auto;
    bottom: calc(12px + env(safe-area-inset-bottom, 0px));
    padding: 0; border-radius: 999px;
  }
  .mu-muelle:not(.abierto) .mu-muelle__dicho,
  .mu-muelle:not(.abierto) .mu-muelle__flecha { display: none; }
  .mu-muelle:not(.abierto) .mu-muelle__tirador {
    width: 54px; height: 54px; padding: 0;
    justify-content: center; font-size: 1.5rem;
  }
  /* Un Punto Avisa que Muchi Dijo algo, sin Abrirse encima de la Carta. */
  .mu-muelle:not(.abierto).dijo .mu-muelle__gato::after {
    content: ""; position: absolute; top: 10px; right: 10px;
    width: 10px; height: 10px; border-radius: 50%;
    background: var(--mu-acento); border: 2px solid var(--mu-papel);
  }
  /* Cerrado Ocupa una Línea; abierto Crece hasta donde la Lista sigue Asomando. */
  .mu-muelle.abierto .mu-muelle__cuerpo {
    display: flex; max-height: 62vh; overflow-y: auto; padding-bottom: 4px;
  }
}
</style>
