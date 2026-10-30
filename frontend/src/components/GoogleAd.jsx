import { useEffect, useState } from 'react'
import { loadAds } from '@shared/adsense.js'
import Scope from './Scope.jsx'
import '../styles/GoogleAd.css'

export default function GoogleAd({ client, slot }) {
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    loadAds(client)
      .then(() => { (window.adsbygoogle = window.adsbygoogle || []).push({}) })
      .catch(() => setFailed(true))
  }, [client])
  return (
    <Scope name="GoogleAd">
      <aside className="mu-google" aria-label="Publicidad">
        <span className="mu-google__marca">Publicidad</span>
        {failed
          ? <p className="mu-caption">MUCHI sigue buscando las mejores ofertas para ti.</p>
          : <ins className="adsbygoogle" data-ad-client={client} data-ad-slot={slot}
                 data-ad-format="auto" data-full-width-responsive="true"></ins>}
      </aside>
    </Scope>
  )
}
