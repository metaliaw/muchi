<script setup>
defineProps({
  donationUrl: { type: String, default: '' },
})
</script>

<template>
  <aside class="mu-panel mu-apoyo">
    <div aria-hidden="true" class="mu-apoyo__huella">🐾</div>
    <div>
      <h2>Ayuda a Muchi</h2>
      <p>Tu aporte mantiene las búsquedas disponibles y ayuda a sumar nuevas tiendas.</p>
    </div>
    <a
      v-if="donationUrl"
      class="mu-boton mu-apoyo__boton"
      :href="donationUrl"
      target="_blank"
      rel="noopener noreferrer"
    >
      Hacer una donación
    </a>
    <p v-else class="mu-caption mu-apoyo__pendiente">Las donaciones estarán disponibles pronto.</p>
  </aside>
</template>

<style scoped>
.mu-apoyo {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px 12px;
  align-items: start;
  box-shadow: none;
}
.mu-apoyo__huella {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border-radius: 50%;
  background: var(--mu-rosa-lav);
  font-size: 1.1rem;
}
h2 { margin: 1px 0 4px; font-size: 1rem; }
p { margin: 0; }
.mu-apoyo__boton,
.mu-apoyo__pendiente { grid-column: 1 / -1; }
.mu-apoyo__boton { margin-top: 4px; text-align: center; }
</style>
