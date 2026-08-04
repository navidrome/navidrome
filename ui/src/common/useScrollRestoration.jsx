import { useEffect, useLayoutEffect, useRef } from 'react'
import { useHistory, useLocation } from 'react-router-dom'

// Keyed on the route, not location.key: the app uses hash history, which never assigns one, so
// every page would otherwise share a slot and overwrite the offset we came back for.
const positions = new Map()
const MAX_ENTRIES = 50
const MAX_FRAMES = 60

export const useScrollRestoration = (ready = true) => {
  const { pathname, search } = useLocation()
  const history = useHistory()
  const key = pathname + search
  const handled = useRef(null)
  const latest = useRef(0)

  useEffect(() => {
    latest.current = window.scrollY
    const track = () => {
      // A document with nothing to scroll is a page being torn down, not a user scrolling: the
      // collapse snaps us to the top, and recording that would erase the offset we are leaving.
      if (document.documentElement.scrollHeight <= window.innerHeight) return
      latest.current = window.scrollY
    }
    window.addEventListener('scroll', track, { passive: true })
    return () => window.removeEventListener('scroll', track)
  }, [key])

  // Committed on the way out rather than on every scroll: leaving collapses this page's document
  // and snaps us to the top, which a live listener would record over the offset we are leaving.
  useLayoutEffect(
    () => () => {
      positions.delete(key)
      positions.set(key, latest.current)
      if (positions.size > MAX_ENTRIES) {
        positions.delete(positions.keys().next().value)
      }
    },
    [key],
  )

  // Layout effect, not passive: a passive one runs after paint, so the restored list would be
  // painted at the old offset first and visibly jump.
  useLayoutEffect(() => {
    if (!ready || handled.current === key) return
    handled.current = key
    const saved = positions.get(key)
    const top = history.action === 'POP' && saved !== undefined ? saved : 0
    window.scrollTo({ top })
    if (!top) return

    // A page can still be filling in once `ready` turns true, and until it is tall enough the
    // browser silently clamps us to the top. Keep asking until the offset sticks.
    let frame
    let frames = 0
    let landed = Math.round(window.scrollY)
    const retry = () => {
      const y = Math.round(window.scrollY)
      // Yielding on where we actually are, not on input events: a trackpad back-swipe keeps
      // firing wheel momentum through the restore, and that is the gesture asking to come back.
      if (y === top || y !== landed || ++frames > MAX_FRAMES) return
      window.scrollTo({ top })
      landed = Math.round(window.scrollY)
      frame = requestAnimationFrame(retry)
    }
    frame = requestAnimationFrame(retry)
    return () => cancelAnimationFrame(frame)
  }, [ready, key, history.action])
}
