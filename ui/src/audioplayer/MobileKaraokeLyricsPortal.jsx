import React, { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'

export const MOBILE_KARAOKE_LYRICS_TRANSITION_MS = 260
export const MOBILE_KARAOKE_LYRICS_HOST_SELECTOR =
  '.react-jinke-music-player-mobile-cover'
export const MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS = 'nd-mobile-lyrics-active'
export const MOBILE_KARAOKE_LYRICS_ENTERED_CLASS = 'nd-mobile-lyrics-entered'
export const MOBILE_KARAOKE_LYRICS_LAYER_CLASS = 'nd-mobile-lyrics-layer'

const resolveMobileLyricsHost = () => {
  if (typeof document === 'undefined') return null
  return document.querySelector(MOBILE_KARAOKE_LYRICS_HOST_SELECTOR)
}

const MobileKaraokeLyricsPortal = ({ active, children }) => {
  const [rendered, setRendered] = useState(active)
  const [entered, setEntered] = useState(active)
  const [host, setHost] = useState(() =>
    active ? resolveMobileLyricsHost() : null,
  )

  useEffect(() => {
    if (active) {
      setRendered(true)
      const frameId = window.requestAnimationFrame(() => setEntered(true))
      return () => window.cancelAnimationFrame(frameId)
    }

    setEntered(false)
    const timerId = window.setTimeout(
      () => setRendered(false),
      MOBILE_KARAOKE_LYRICS_TRANSITION_MS,
    )
    return () => window.clearTimeout(timerId)
  }, [active])

  useEffect(() => {
    if (typeof document === 'undefined') {
      setHost(null)
      return undefined
    }

    if (!rendered) {
      setHost(null)
      return undefined
    }

    const syncHost = () => setHost(resolveMobileLyricsHost())
    syncHost()

    const observer = new MutationObserver(syncHost)
    observer.observe(document.body, { childList: true, subtree: true })

    return () => observer.disconnect()
  }, [rendered])

  useEffect(() => {
    if (!host) return undefined

    host.classList.toggle(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS, rendered)
    host.classList.toggle(MOBILE_KARAOKE_LYRICS_ENTERED_CLASS, entered)
    return () => {
      host.classList.remove(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
      host.classList.remove(MOBILE_KARAOKE_LYRICS_ENTERED_CLASS)
    }
  }, [entered, host, rendered])

  if (!rendered || !host) return null
  return createPortal(
    <div
      className={MOBILE_KARAOKE_LYRICS_LAYER_CLASS}
      data-entered={entered ? 'true' : 'false'}
      aria-hidden={!entered}
    >
      {children}
    </div>,
    host,
  )
}

export default MobileKaraokeLyricsPortal
