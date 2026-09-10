let scriptLoad = null

/** Carga AdSense una sola vez aunque varios Componentes lo soliciten. */
export function loadAds(client) {
  if (scriptLoad) return scriptLoad

  // El Script del head ya dejó la Cola lista: no hay nada que Cargar.
  if (window.adsbygoogle) {
    scriptLoad = Promise.resolve()
    return scriptLoad
  }

  scriptLoad = new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.async = true
    script.crossOrigin = 'anonymous'
    script.src = `https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${encodeURIComponent(client)}`
    script.addEventListener('load', resolve, { once: true })
    script.addEventListener('error', reject, { once: true })
    document.head.appendChild(script)
  })
  return scriptLoad
}
