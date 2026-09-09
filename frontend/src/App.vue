<script setup>
/** Muchi Presenta Búsquedas y Resultados persistidos por la API. */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import * as api from './api.js'
import MuchiPanel from './components/MuchiPanel.vue'
import CardLookup from './components/CardLookup.vue'
import CardArt from './components/CardArt.vue'
import SiteFooter from './components/SiteFooter.vue'
import SearchForm from './components/SearchForm.vue'
import SearchProgress from './components/SearchProgress.vue'
import OfferList from './components/OfferList.vue'
import CartPanel from './components/CartPanel.vue'
import SourcesPanel from './components/SourcesPanel.vue'
import GoogleAd from './components/GoogleAd.vue'
import GoogleAdsense from './components/GoogleAdsense.vue'
import SponsorSpot from './components/SponsorSpot.vue'

const THEME_KEY = 'muchi_tema'

const config = ref({ poll_seconds: 5, adsense_client: 'ca-pub-6368656861543000' })
const book = ref(null)
const dark = ref(localStorage.getItem(THEME_KEY) === 'oscuro')
const message = ref(null)

const searchId = ref(new URLSearchParams(location.search).get('search') || '')
const state = ref(null)
const offers = ref([])
const summary = ref(null)
const notices = ref([])
const checked = ref('')
const unavailable = ref('')
const error = ref('')
const pending = ref(null)
const lookupText = ref('')
// La Carta que se mira: un Nombre, y el Idioma en que se escribió.
const watched = ref(null)
const history = ref(JSON.parse(localStorage.getItem('muchi_historial') || '[]'))

let timer = null

const busy = computed(() => Boolean(state.value && !state.value.done && !unavailable.value))
const googleReady = computed(() => Boolean(config.value.adsense_client && config.value.adsense_slot))
const sponsorReady = computed(() => Boolean(config.value.sponsor_name && config.value.sponsor_url))

function hashSearch(id) {
  return [...id].reduce((value, letter) => ((value * 31) + letter.charCodeAt(0)) >>> 0, 0)
}

const showSponsor = computed(() => sponsorReady.value && hashSearch(searchId.value) % 4 === 0)

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

function remember(id, label) {
  const rest = history.value.filter((entry) => entry.id !== id)
  history.value = [{ id, label: label || id }, ...rest].slice(0, 20)
  localStorage.setItem('muchi_historial', JSON.stringify(history.value))
}

function selectSearch(id) {
  searchId.value = id
  state.value = null
  offers.value = []
  summary.value = null
  notices.value = []
  unavailable.value = ''
  checked.value = ''
  const url = new URL(location.href)
  url.searchParams.set('search', id)
  history.value.length && window.history.replaceState({}, '', url)
  refresh()
}

async function submit(text) {
  pending.value = { text, key: api.newKey() }
  await send()
}

function loadCard(canonicalName) {
  lookupText.value = canonicalName
}

function lookAtCard(name, language = '') {
  watched.value = { name, language }
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
    const reply = await api.createSearch(pending.value.text, pending.value.key)
    remember(reply.state.id, reply.label)
    pending.value = null
    selectSearch(reply.state.id)
    say('¡Miau! Ya salí a buscar', 'happy')
  } catch (failure) {
    error.value = failure.message
    // Un Rechazo no se reintenta: el Envío pendiente se descarta.
    if (!failure.retriable) pending.value = null
    say(failure.message, 'angry')
  }
}

async function refresh() {
  if (!searchId.value || unavailable.value) return
  try {
    const reply = await api.readSearch(searchId.value)
    state.value = reply.state
    offers.value = reply.offers
    summary.value = reply.summary
    notices.value = reply.notices
    checked.value = new Date().toISOString().slice(11, 19) + ' UTC'
    remember(reply.state.id)
    if (reply.state.done) stopPolling()
  } catch (failure) {
    if (failure.retriable) {
      // Los Resultados recibidos se conservan y se reintenta la Consulta.
      error.value = failure.message
    } else {
      unavailable.value = failure.message
      stopPolling()
    }
  }
}

async function cancel() {
  try {
    const reply = await api.cancelSearch(searchId.value, api.newKey())
    state.value = reply.state
    say('Ya paré de buscar', 'idle')
  } catch (failure) {
    error.value = failure.message
  }
}

function startPolling() {
  stopPolling()
  const seconds = config.value.poll_seconds || 5
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
    <h1>🐱 Muchi</h1>
    <p>Busca Cartas y cotiza tu Lista</p>
  </header>

  <main class="mu-grilla">
    <div class="mu-lateral">
      <MuchiPanel :book="book" v-model:dark="dark" :message="message" />
      <CardArt :card="watched" />
    </div>

    <div class="mu-columna">
      <SearchForm
        v-model:text="lookupText"
        :busy="busy" :pending="Boolean(pending)" :error="error"
        @search="submit" @retry="send" @resume="selectSearch"
      >
        <template #lookup>
          <CardLookup
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
          v-if="showSponsor || !googleReady"
          :key="`promo-${searchId}`"
          :searching="!state.done"
          :sponsor-name="showSponsor ? config.sponsor_name : ''"
          :sponsor-text="showSponsor ? config.sponsor_text : ''"
          :sponsor-url="showSponsor ? config.sponsor_url : ''"
        />
        <GoogleAd
          v-else
          :key="`google-${searchId}`"
          :client="config.adsense_client"
          :slot="config.adsense_slot"
        />
      </SearchProgress>

      <OfferList
        v-if="state" :offers="offers" :summary="summary"
        :notices="notices" :placeholder="placeholder"
        @look="lookAtCard"
      />

      <CartPanel v-if="offers.length" :search-id="searchId" />
      <SourcesPanel />
    </div>
  </main>

  <SiteFooter
    :donation-url="config.donation_url"
    :repository-url="config.repository_url"
    :socials="config.socials || []"
  />
</template>

<style scoped>
.mu-hero {
  background: linear-gradient(135deg, var(--mu-acento), var(--mu-peri));
  color: #fff; padding: 26px 24px; border-radius: 0 0 28px 28px;
  box-shadow: var(--mu-sombra);
}
.mu-hero h1 { margin: 0; font-size: 2.4rem; font-weight: 800; letter-spacing: -1px; }
.mu-hero p { margin: 4px 0 0; opacity: .9; }
.mu-grilla {
  display: grid; grid-template-columns: 260px 1fr; gap: 20px;
  max-width: 1100px; margin: 22px auto; padding: 0 16px; align-items: start;
}
.mu-columna { display: flex; flex-direction: column; gap: 16px; }
.mu-lateral { display: flex; flex-direction: column; gap: 16px; }
.mu-historia { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
summary { cursor: pointer; font-weight: 600; }
@media (max-width: 800px) { .mu-grilla { grid-template-columns: 1fr; } }
</style>
