<script setup>
const REPOSITORY_URL = 'https://github.com/metaliaw/muchi'
const NEW_ISSUE_URL = `${REPOSITORY_URL}/issues/new`
</script>

<template>
  <aside class="mu-panel mu-comunidad">
    <p class="mu-comunidad__eyebrow">Código abierto</p>
    <h2>Aprende con Muchi</h2>
    <p>
      Revisa cómo está hecha la parte visible, conoce sus criterios y ayúdanos
      a mejorarla.
    </p>
    <nav aria-label="Participar en Muchi" class="mu-comunidad__enlaces">
      <a :href="REPOSITORY_URL" target="_blank" rel="noopener noreferrer">
        Ver el código
      </a>
      <a :href="NEW_ISSUE_URL" target="_blank" rel="noopener noreferrer">
        Enviar un comentario
      </a>
    </nav>
  </aside>
</template>

<style scoped>
.mu-comunidad {
  box-shadow: none;
  background:
    linear-gradient(145deg, color-mix(in srgb, var(--mu-peri) 9%, transparent), transparent 55%),
    var(--mu-blanco);
}
.mu-comunidad__eyebrow {
  margin: 0 0 5px;
  color: var(--mu-peri);
  font-size: .72rem;
  font-weight: 800;
  letter-spacing: .09em;
  text-transform: uppercase;
}
h2 { margin: 0 0 6px; font-size: 1rem; }
p { margin: 0; }
.mu-comunidad__enlaces {
  display: flex;
  gap: 8px 14px;
  margin-top: 12px;
  flex-wrap: wrap;
}
.mu-comunidad__enlaces a { font-weight: 700; }
</style>
