import { useEffect, useRef } from 'react'
import { useHistory, useLocation } from 'react-router-dom'

// Keyed on the route, not location.key: the app uses hash history, which never assigns one, so
// every page would otherwise share a slot and overwrite the offset we came back for.
const positions = new Map()
const maxEntries = 50

export const useScrollRestoration = (ready = true) => {
  const { pathname, search } = useLocation()
  const history = useHistory()
  const key = pathname + search
  const handled = useRef(null)

  useEffect(() => {
    const save = () => {
      positions.delete(key)
      positions.set(key, window.scrollY)
      if (positions.size > maxEntries) {
        positions.delete(positions.keys().next().value)
      }
    }
    window.addEventListener('scroll', save, { passive: true })
    return () => window.removeEventListener('scroll', save)
  }, [key])

  useEffect(() => {
    if (!ready || handled.current === key) return
    handled.current = key
    const saved = positions.get(key)
    const top = history.action === 'POP' && saved !== undefined ? saved : 0
    window.scrollTo({ top })
  }, [ready, key, history.action])
}
