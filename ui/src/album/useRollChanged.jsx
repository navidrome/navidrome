import { useRef } from 'react'

// useRollChanged reports that the albums on screen belong to a different roll than the one being
// loaded. Only a seed change is a re-roll; a search keystroke refetches the same roll.
export const useRollChanged = (seed, loading) => {
  // Starts empty, not at the current seed: a re-roll redirects and remounts this component, so an
  // initial value of `seed` would look already-settled while the stale roll is still on screen.
  const shown = useRef(null)
  const wasLoading = useRef(loading)

  if (!loading && (shown.current === null || wasLoading.current)) {
    shown.current = seed
  }
  wasLoading.current = loading
  return shown.current !== seed
}
