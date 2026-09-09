<script setup>
import { onMounted } from 'vue'
import { loadAds } from '../adsense.js'

const props = defineProps({
  client: { type: String, required: true },
})

onMounted(() => loadAds(props.client).catch(() => {}))
</script>

<template></template>
