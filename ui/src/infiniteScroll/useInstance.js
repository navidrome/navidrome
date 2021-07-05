import { useRef } from 'react'

// Instance value persists through re-renders
// and doesn't trigger a re-render
export const useInstance = (initialValue = null) => {
  const mountedRef = useRef(initialValue)

  const update = (value) => (mountedRef.current = value)

  return [mountedRef.current, update]
}
