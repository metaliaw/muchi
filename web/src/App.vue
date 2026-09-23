<script setup>
/** Muchi Presenta Búsquedas y Resultados persistidos por la API. */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import * as api from './api.js'
import { declaredStock, topOf, useSearch } from './search.js'
import {
  confirmOffers, countReachable, readFreshChecks, rememberChecks, sayAge,
} from './stock.js'
import MuchiPanel from './components/MuchiPanel.vue'
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
import StoreCardDetail from './components/StoreCardDetail.vue'

const THEME_KEY = 'muchi_tema'
// Quien Busca Cartas de un Juego Vuelve al mismo: el Selector Recuerda el
// último, y no lo Devuelve a Magic en cada Visita.
const GAME_KEY = 'muchi_juego'
const KIND_KEY = 'muchi_catalogo'
// Lo último que alguien Buscó Vuelve escrito en el Campo: repetir la misma
// Lista al día siguiente no Debería Costar tipearla de nuevo.
const TEXT_KEY = 'muchi_busqueda'
// Recargar no Debería Mandar a las Tiendas otra vez. La última Búsqueda mirada
// se Retoma por su Identificador, con el Modo y el Catálogo con que salió: el
// Servidor ya la Tiene hecha y solo la Devuelve.
const LAST_KEY = 'muchi_ultima'

// El Ritmo lo manda el Servidor; este es el mismo de config/api.defaults.yaml,
// para los milisegundos que van entre que arranca la Página y llega la Config.
const config = ref({ poll_seconds: 3, adsense_client: 'ca-pub-6368656861543000',
                     stock_fresh_seconds: 600, browser_check_limit: 5 })
const book = ref(null)
const games = ref([])
const game = ref('')
const dark = ref(localStorage.getItem(THEME_KEY) === 'oscuro')
const message = ref(null)

// La Búsqueda vive en su propio Módulo: sus Refs, su Limpieza y la Mezcla de
// las Páginas se declaran una sola vez ahí.
const {
  id: searchId, state, items, offers, summary, notices, cursor, hasMore,
  checked, unavailable, stocked, start: startSearch, applyState, applyResults,
  applyStock,
} = useSearch()
// El Muelle nace Cerrado: quien Busca quiere ver Ofertas, y Muchi y la Carta
// Esperan a un Toque. Solo Existe en Móvil; en Escritorio el Lateral los Muestra.
const dockOpen = ref(false)
// El Carrito Tiene su propio Cajón a la Derecha. Los dos Comparten la Esquina
// de abajo en Móvil, así que Abrir uno Guarda al otro.
const cartOpen = ref(false)
watch(dockOpen, (shown) => { if (shown) cartOpen.value = false })
watch(cartOpen, (shown) => { if (shown) dockOpen.value = false })
const error = ref('')
const pending = ref(null)
// El Campo nace con lo último que se Buscó. Se Lee acá, antes de que el
// Formulario Exista: con Texto propio, el Ejemplo no lo Pisa.
const lookupText = ref(localStorage.getItem(TEXT_KEY) || '')
// El Modo viaja en la URL junto a la Búsqueda: recargar no debe reagrupar las
// mismas Ofertas de otra manera ni mandar Derivados al Carrito.
const match = ref(new URLSearchParams(location.search).get('match') === 'includes'
  ? 'includes' : 'exact')
// El Catálogo viaja igual que el Modo: una Búsqueda de Cajas Retomada por su
// Enlace no debe Volver como una de Cartas.
const kind = ref(new URLSearchParams(location.search).get('kind') === 'sealed'
  ? 'sealed' : 'single')
// La Carta que se mira: un Nombre, y el Idioma en que se escribió.
const watched = ref(null)
const detailed = ref(null)
const history = ref(JSON.parse(localStorage.getItem('muchi_historial') || '[]'))
// Cuántas Copias Tiene cada Oferta, contadas a mano por quien está mirando la
// Tienda. Viven acá porque las Escribe la Lista y las Usa el Carrito.
const units = ref({})
// El Reparto lo Hace la Lista, que es la única que Sabe qué Edición se está
// Mirando y con qué Criterio. Acá solo se Guarda lo Elegido.

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

// Una Fuente caída Deja un Hueco, no un Vacío. Los Avisos de Nivel `warning`
// son justo eso: una Consulta que no se Completó o una Tienda que no Contestó.
const incomplete = computed(() =>
  notices.value.some((notice) => notice.level === 'warning'))

const placeholder = computed(() => {
  if (unavailable.value) return 'No hay ofertas recibidas para mostrar.'
  // Decir «No hay» Cuando alguien no Contestó es Afirmar lo que no se Sabe.
  if (state.value?.done && incomplete.value)
    return 'Ninguna oferta llegó, y algunas fuentes no contestaron. Lo que falta puede existir igual: reintenta en un rato.'
  if (state.value?.done) return 'No hay ofertas para mostrar.'
  if (state.value?.status === 'queued')
    return 'Búsqueda en cola. Esperando que el servicio la procese.'
  if (state.value)
    return 'Consultando ofertas. Una carta puede tardar varios minutos; los resultados aparecen cuando la API termina de consultarla.'
  return ''
})

// Confirmar el Stock es un Paso aparte, y lo Pide quien Compra. Cuesta una
// Visita a cada Tienda: las que el Navegador Alcanza las Pide él mismo, y por
// el Resto Pregunta el Servicio. Sin nada que Alcanzar no se Ofrece el Botón,
// porque sería Prometer una Certeza que Traería el Servicio entero a cuestas.
const confirming = ref(false)
// Hace cuánto se Confirmó lo que se está mostrando. Cero es «no se Confirmó»:
// la Certeza vieja se Descartó sola y el Botón Volvió a su lugar.
const confirmedAge = ref(0)
const buying = computed(() =>
  offers.value.filter((offer) => units.value[offer.offer_id] > 0))
const reachable = computed(() => countReachable(
  buying.value.length ? buying.value : offers.value, config.value.browser_check_limit))
const confirmable = computed(() =>
  Boolean(state.value?.done) && reachable.value > 0 && !stocked.value)

// Tocar o Elegir una Oferta Pregunta si Está, a esa Tienda y a nadie más: un
// Toque no Debe Desatar una Ronda del Servicio. Lo que el Navegador Alcanza lo
// Pide él; por el Resto —una Tienda leída de Listas, un Catálogo sin JSON—
// Preguntamos nosotros, pero solo por esa Oferta.
async function confirmOne(offer) {
  if (!offer?.offer_id) return
  const [check] = await confirmOffers([offer], fetch, 1)
  if (check) rememberChecks(searchId.value, [check])
  let answer
  try {
    // Un No del Navegador Cierra el Asunto: no se Compra lo que ya no Está, y
    // no hace falta que nadie lo Vuelva a mirar. Un Sí no Cierra nada, porque
    // el Catálogo que el Navegador Lee Dice si Queda y no Cuántas: el Número
    // está en la Página, y esa la Leemos nosotros.
    answer = check && !check.available
      ? await api.confirmStock(searchId.value, [check], match.value, false)
      : await api.confirmStock(searchId.value, [], match.value, false,
                               [offer.offer_id])
  } catch {
    say(`No pude preguntarle a ${offer.store}`, 'idle')
    return
  }
  applyStock(answer)
  const said = (answer.offers || []).find((row) => row.offer_id === offer.offer_id)
  // El Navegador Contesta sí o no; el Servicio Contesta una Fila, y una Fila
  // que no Vuelve es una Tienda que no Dijo nada.
  const available = said ? said.stock_status !== 'unavailable'
    : check ? check.available : null
  if (available === null) {
    say(`${offer.store} no dice si queda`, 'idle')
    return
  }
  if (!available) {
    // Contar una Copia de lo que ya no Está Metería al Carrito una Compra que
    // la Tienda acaba de Negar.
    const { [offer.offer_id]: gone, ...rest } = units.value
    units.value = rest
    say(`En ${offer.store} ya no queda`, 'idle')
    return
  }
  // Quedan menos de las Elegidas: se Baja al Tope de esa Tienda. Dejar tres
  // Copias donde Hay una Prometería una Compra que no se Puede hacer, y una
  // Tienda que Dijo que Queda sin Decir cuántas Vale una sola.
  const counted = declaredStock(said || offer)
  const left = topOf(said || offer)
  if (left && units.value[offer.offer_id] > left) {
    units.value = { ...units.value, [offer.offer_id]: left }
    // Decir un Número que la Tienda no Dijo sería Inventarlo: sin Cuenta se
    // Dice por qué Bajó, no cuántas Quedan.
    say(counted === null ? `${offer.store} no dice cuántas quedan; te dejo una`
                         : `En ${offer.store} quedan ${counted}`, 'idle')
    return
  }
  say(`En ${offer.store} sí queda`, 'happy')
}

async function confirmStock() {
  confirming.value = true
  say('Estoy preguntando en las tiendas', 'talk')
  try {
    // Se Comprueba lo que se va a Comprar. Sin nada Elegido todavía, se
    // Comprueban las más baratas, que es lo que alguien Compraría.
    const buying = offers.value.filter((offer) => units.value[offer.offer_id] > 0)
    const found = await confirmOffers(buying.length ? buying : offers.value, fetch,
                                      config.value.browser_check_limit)
    applyStock(await api.confirmStock(searchId.value, found, match.value))
    rememberChecks(searchId.value, found)
    confirmedAge.value = 0
    const gone = found.filter((check) => !check.available).length
    say(gone ? `Pregunté, y ${gone} de las baratas ya no están`
             : 'Pregunté, y las baratas siguen en pie', gone ? 'idle' : 'happy')
  } catch (failure) {
    error.value = failure.message
    say(failure.message, 'angry')
  } finally {
    confirming.value = false
  }
}

function say(text, mood = 'happy') {
  message.value = { text, state: mood }
}

function applyTheme() {
  document.documentElement.dataset.tema = dark.value ? 'oscuro' : 'claro'
  localStorage.setItem(THEME_KEY, dark.value ? 'oscuro' : 'claro')
}
watch(dark, applyTheme)
watch(game, (named) => {
  watched.value = null
  if (named) localStorage.setItem(GAME_KEY, named)
})
// La Carta mirada es de un Catálogo: pasar a Cajas Deja en el Panel una Carta
// que ya nadie Busca.
watch(kind, (named) => {
  watched.value = null
  localStorage.setItem(KIND_KEY, named)
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
  localStorage.setItem(LAST_KEY, JSON.stringify(
    { id, match: match.value, kind: kind.value }))
  const url = new URL(location.href)
  url.searchParams.set('search', id)
  url.searchParams.set('match', match.value)
  url.searchParams.set('kind', kind.value)
  history.value.length && window.history.replaceState({}, '', url)
  refresh()
}

async function submit(text) {
  // Se Guarda lo Escrito, no lo Enviado: en Modo ancho sale una Línea sola, y
  // quien Vuelve Quiere su Lista entera de vuelta.
  localStorage.setItem(TEXT_KEY, lookupText.value)
  pending.value = { text, game: game.value, key: api.newKey(), match: match.value,
                    kind: kind.value }
  await send()
}

// De una Oferta llega la Impresión entera, que es lo que el Panel Mira.
function lookAtCard(card) {
  watched.value = card
}

// Un Grupo del Catálogo, dicho al azar. Un Grupo vacío no dice nada: el
// Catálogo llega un Instante después del primer Pintado.
function sayFrom(group) {
  const rows = book.value?.[group]
  if (!rows?.length) return
  const said = rows[Math.floor(Math.random() * rows.length)]
  say(said.text, said.state)
}

// La Oferta más barata le Saca un Comentario a Muchi. En Móvil no hay Hover y
// no Pasa nada: el Comentario Adorna, no Informa.
function sayCheap() {
  sayFrom('bargain')
}

// A quien mira las Estadísticas, Muchi lo saluda como se merece.
function sayNerd() {
  sayFrom('nerd')
}

// Cerrar el Aviso del Código Abierto Despeja esta Visita, no las que Vengan:
// cada Recarga lo Trae de vuelta. Es un Aviso, no una Preferencia.
const libreOpen = ref(true)

function closeLibre() {
  libreOpen.value = false
}

// Quien Toca el Aviso del Código Abierto escucha a Muchi hablar de su Licencia.
function sayLibre() {
  sayFrom('libre')
}

// Muchi no completa el Campo: dice lo que vio y quien escribe decide.
async function send() {
  if (!pending.value) return
  error.value = ''
  try {
    const reply = await api.createSearch(
      pending.value.text, pending.value.game, pending.value.key, pending.value.match,
      pending.value.kind
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

// Recargar no Vuelve a preguntarle a nadie: lo Confirmado hace poco se Rearma
// desde la Memoria y el Servidor Corona con eso. Pasado el Tope, no Vuelve
// nada y el Botón Reaparece — que es exactamente lo que Debería pasar.
async function restoreStock() {
  const { checks, age } = readFreshChecks(searchId.value, config.value.stock_fresh_seconds)
  if (!checks.length) return
  try {
    applyStock(await api.confirmStock(searchId.value, checks, match.value))
    confirmedAge.value = age
  } catch {
    // Sin Corona rearmada se Sigue con la que trajo la Búsqueda. No es un
    // Error que Contarle a nadie: nadie Pidió esto.
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

// El Juego Guardado vale mientras el Servidor lo siga Sirviendo. Uno que se
// Retiró Volvería como un Selector en blanco, y ninguna Búsqueda saldría.
function rememberedGame() {
  const named = localStorage.getItem(GAME_KEY)
  return games.value.some((row) => row.reference_key === named) ? named : ''
}

// Una Búsqueda Guardada Caduca en el Servidor antes que en el Navegador. Si ya
// no Está, se Olvida callado: nadie Abrió un Enlace roto, solo Volvió a Casa.
async function resumeLast() {
  const saved = JSON.parse(localStorage.getItem(LAST_KEY) || 'null')
  if (!saved?.id) return
  match.value = saved.match === 'includes' ? 'includes' : 'exact'
  kind.value = saved.kind === 'sealed' ? 'sealed' : 'single'
  startSearch(saved.id)
  await refresh()
  if (unavailable.value) {
    localStorage.removeItem(LAST_KEY)
    startSearch('')
    unavailable.value = ''
    return
  }
  await restoreStock()
  const url = new URL(location.href)
  url.searchParams.set('search', saved.id)
  url.searchParams.set('match', match.value)
  url.searchParams.set('kind', kind.value)
  window.history.replaceState({}, '', url)
}

watch(busy, (value) => (value ? startPolling() : stopPolling()))

onMounted(async () => {
  applyTheme()
  try {
    const supported = await api.readSupportedGames()
    games.value = supported.games || []
    game.value = rememberedGame() || games.value[0]?.reference_key || ''
    // La URL Manda sobre lo Recordado: quien Abre un Enlace de Cajas Ve Cajas.
    if (!new URLSearchParams(location.search).has('kind')) {
      kind.value = localStorage.getItem(KIND_KEY) === 'sealed' ? 'sealed' : 'single'
    }
    ;[config.value, book.value] = await Promise.all([api.readConfig(), api.readMuchi()])
  } catch (failure) {
    error.value = failure.message
  }
  // La URL Manda: con Búsqueda escrita, esa se Mira. Sin ella, Vuelve la última.
  if (searchId.value) {
    await refresh()
    await restoreStock()
  } else {
    await resumeLast()
  }
})
onUnmounted(stopPolling)
</script>

<template>
  <GoogleAdsense :client="config.adsense_client" />

  <header class="mu-hero">
    <div class="mu-hero__nombre">
      <h1><a href="/">🐱 Muchi</a></h1>
      <p>Busca cartas y cotiza tu lista</p>
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
    <div v-if="libreOpen" class="mu-abierto">
      <CommunityPanel :repository-url="config.repository_url"
                      :api-repository-url="config.api_repository_url"
                      @libre="sayLibre" @close="closeLibre" />
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
        <CardArt :card="watched" :game="game" :kind="kind" />
      </div>
    </div>

    <div class="mu-columna">
      <SearchForm
        v-model:text="lookupText"
        v-model:game="game"
        v-model:match="match"
        v-model:kind="kind"
        :games="games"
        :busy="busy" :pending="Boolean(pending)" :error="error"
        :limits="config.limits"
        @search="submit" @retry="send" @resume="selectSearch"
      >
      </SearchForm>

      <section v-if="history.length" class="mu-panel">
        <details>
          <summary>Búsquedas recientes</summary>
          <p class="mu-caption">Guarda el enlace para retomarlas después.</p>
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

      <!-- El Stock se Confirma cuando la Búsqueda ya Cerró: preguntarlo a
           media Búsqueda Corona una Oferta que la siguiente Página abarata. -->
      <p v-if="stocked" class="mu-panel mu-confirmar">
        <span class="mu-caption">
          Stock confirmado {{ sayAge(confirmedAge) }} en las tiendas que lo dicen.
          Vale por un rato, no por el día.
        </span>
      </p>

      <p v-if="confirmable" class="mu-panel mu-confirmar">
        <button class="mu-ghost" :disabled="confirming" @click="confirmStock">
          {{ confirming ? 'Preguntando…' : 'Confirmar stock' }}
        </button>
        <span class="mu-caption">
          Tu navegador le pregunta directo a hasta {{ reachable }} tiendas y
          corta en cuanto una dice que sí. Lo que confirme vale para este
          minuto, no para mañana.
        </span>
      </p>

      <OfferList
        v-if="state" :items="items" :offers="offers" :summary="summary"
        :notices="notices" :placeholder="placeholder"
        :advertise-groups="advertiseSections"
        v-model:units="units"
        @confirm="confirmOne"
        @look="lookAtCard"
        @detail="detailed = $event"
        @cheap="sayCheap"
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

      <SourcesPanel @nerd="sayNerd" />
    </div>

    <!-- Comparar ya Sirve sin Comprar, así que el Carrito está siempre. Lo
         que el Servidor Decide es si todavía hace falta Avisar que la Compra
         no Está. -->
    <CartPanel v-if="offers.length"
               :search-id="searchId" :match="match"
               :units="units" :docked="dockOpen" v-model:open="cartOpen"
               :repository-url="config.repository_url"
               :ready="config.cart_ready" />
  </main>

  <StoreCardDetail v-if="detailed" :offer="detailed" :game="game"
                   @close="detailed = null" />

</template>

<style scoped>
.mu-confirmar { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.mu-confirmar > button { flex: none; }
.mu-confirmar > .mu-caption { flex: 1; min-width: 200px; }
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
