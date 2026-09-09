<script setup>
import { nextTick, onMounted, ref } from 'vue'
import { loadAds } from '../adsense.js'

const props = defineProps({
  client: { type: String, required: true },
  slot: { type: String, required: true },
})

const failed = ref(false)

onMounted(async () => {
  try {
    await nextTick()
    await loadAds(props.client)
    window.adsbygoogle = window.adsbygoogle || []
    window.adsbygoogle.push({})
  } catch {
    failed.value = true
  }
})
</script>

<template>
  <aside class="mu-google" aria-label="Publicidad">
    <span class="mu-google__marca">Publicidad</span>
    <p v-if="failed" class="mu-caption">Muchi sigue buscando las mejores ofertas para ti.</p>
    <ins
      v-else
      class="adsbygoogle"
      :data-ad-client="client"
      :data-ad-slot="slot"
      data-ad-format="auto"
      data-full-width-responsive="true"
    ></ins>
  </aside>
</template>

<style scoped>
.mu-google {
  width: 100%;
  min-width: 0;
  padding: 10px 12px;
  border: 1px dashed var(--mu-tinta-sw);
  border-radius: 14px;
  background: color-mix(in srgb, var(--mu-blanco) 55%, transparent);
}
.mu-google__marca {
  display: block;
  margin-bottom: 4px;
  color: var(--mu-tinta-sw);
  font-size: .72rem;
}
.adsbygoogle { display: block; width: 100%; }
p { margin: 4px 0; }
</style>
