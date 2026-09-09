let scriptLoad = null

/** Carga AdSense una sola vez aunque varios Componentes lo soliciten. */
export function loadAds(client) {
  if (scriptLoad) return scriptLoad

  scriptLoad = new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.async = true
    script.crossOrigin = 'anonymous'
    script.dataset.muchiAdsense = 'true'
    script.src = `https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${encodeURIComponent(client)}`
    script.addEventListener('load', resolve, { once: true })
    script.addEventListener('error', reject, { once: true })
    document.head.appendChild(script)
  })
  return scriptLoad
}
