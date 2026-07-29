import React, { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import useEnterExitTransition from './useEnterExitTransition'
import useRestoreFocusOnExit from './useRestoreFocusOnExit'

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

const MobileKaraokeLyricsPortal = ({ active, children, returnFocusRef }) => {
  const { rendered, entered } = useEnterExitTransition(
    active,
    MOBILE_KARAOKE_LYRICS_TRANSITION_MS,
  )
  const [host, setHost] = useState(() =>
    active ? resolveMobileLyricsHost() : null,
  )
  const layerRef = useRef(null)

  useRestoreFocusOnExit({
    surfaceRef: layerRef,
    entered,
    returnFocusRef,
    surfaceKey: host,
  })

  useEffect(() => {
    if (typeof document === 'undefined') {
      setHost(null)
      return undefined
    }

    if (!rendered) {
      setHost(null)
      return undefined
    }

    setHost(resolveMobileLyricsHost())

    const observer = new MutationObserver(() => {
      const nextHost = resolveMobileLyricsHost()
      setHost((current) => (current === nextHost ? current : nextHost))
    })
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
      ref={layerRef}
      data-entered={entered ? 'true' : 'false'}
      aria-hidden={!entered}
      inert={entered ? undefined : ''}
      style={{ pointerEvents: entered ? 'auto' : 'none' }}
    >
      {children}
    </div>,
    host,
  )
}

export default MobileKaraokeLyricsPortal
