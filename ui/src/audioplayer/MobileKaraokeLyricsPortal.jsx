import React, { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'

export const MOBILE_KARAOKE_LYRICS_HOST_SELECTOR =
  '.react-jinke-music-player-mobile-cover'
export const MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS = 'nd-mobile-lyrics-active'

const resolveMobileLyricsHost = () => {
  if (typeof document === 'undefined') {
    return null
  }
  return document.querySelector(MOBILE_KARAOKE_LYRICS_HOST_SELECTOR)
}

const MobileKaraokeLyricsPortal = ({ active, children }) => {
  const [host, setHost] = useState(() =>
    active ? resolveMobileLyricsHost() : null,
  )

  useEffect(() => {
    if (typeof document === 'undefined') {
      setHost(null)
      return undefined
    }

    if (!active) {
      setHost(null)
      return undefined
    }

    const syncHost = () => {
      setHost(resolveMobileLyricsHost())
    }

    syncHost()

    const observer = new MutationObserver(syncHost)
    observer.observe(document.body, {
      childList: true,
      subtree: true,
    })

    return () => observer.disconnect()
  }, [active])

  useEffect(() => {
    if (!host) {
      return undefined
    }

    host.classList.toggle(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS, active)

    return () => {
      host.classList.remove(MOBILE_KARAOKE_LYRICS_ACTIVE_CLASS)
    }
  }, [active, host])

  if (!active || !host) {
    return null
  }

  return createPortal(children, host)
}

export default MobileKaraokeLyricsPortal
