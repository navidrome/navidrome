import { useRef } from 'react'

// Reports that the albums on screen belong to a different roll than the one loading: only a seed
// change is a re-roll. `shown` is owned above the grid, which a refresh remounts under the new seed.
export const useRollChanged = (shown, seed, loading) => {
  const wasLoading = useRef(loading)

  if (!loading && (shown.current === null || wasLoading.current)) {
    shown.current = seed
  }
  wasLoading.current = loading
  return shown.current !== seed
}
