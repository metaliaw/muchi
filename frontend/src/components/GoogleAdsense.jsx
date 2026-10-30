import { useEffect } from 'react'
import { loadAds } from '@shared/adsense.js'

export default function GoogleAdsense({ client }) {
  useEffect(() => { loadAds(client).catch(() => {}) }, [client])
  return null
}
