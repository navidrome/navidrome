import { useEffect, useRef } from 'react'
import { useHistory, useLocation } from 'react-router-dom'

// Keyed on the history entry, so back and forward each restore their own offset. Bounded because
// history entries are not: without a cap this grows for the life of the session.
const positions = new Map()
const maxEntries = 50

export const useScrollRestoration = (ready = true) => {
  const { key = 'initial' } = useLocation()
  const history = useHistory()
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
