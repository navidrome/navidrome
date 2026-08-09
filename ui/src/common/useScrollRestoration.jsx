import { useEffect, useLayoutEffect, useRef } from 'react'
import { useHistory, useLocation } from 'react-router-dom'

// Keyed on the route because hash history assigns no location.key, so every page would
// otherwise share one slot and overwrite the offset we came back for.
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
      latest.current = window.scrollY
    }
    window.addEventListener('scroll', track, { passive: true })
    return () => window.removeEventListener('scroll', track)
  }, [key])

  // Layout cleanup, so the offset is banked while this page is torn down. A passive one runs
  // after the incoming page tops itself, and this page's listener records that 0 first.
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

  // Layout effect: a passive one runs after paint, so the list would be painted at the old
  // offset first and visibly jump.
  useLayoutEffect(() => {
    if (!ready || handled.current === key) return
    handled.current = key
    const saved = positions.get(key)
    const top = history.action === 'POP' && saved !== undefined ? saved : 0
    window.scrollTo({ top })
    if (!top || Math.round(window.scrollY) === top) return

    // The page can still be filling in, and until it is tall enough the browser clamps us to
    // the top. Keep asking until the offset sticks, yielding if anything else moves us.
    let frame
    let frames = 0
    let landed = Math.round(window.scrollY)
    const retry = () => {
      const y = Math.round(window.scrollY)
      if (y === top || y !== landed || ++frames > MAX_FRAMES) return
      window.scrollTo({ top })
      landed = Math.round(window.scrollY)
      frame = requestAnimationFrame(retry)
    }
    frame = requestAnimationFrame(retry)
    return () => cancelAnimationFrame(frame)
  }, [ready, key, history.action])
}
