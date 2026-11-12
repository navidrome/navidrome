import { useLayoutEffect } from 'react'

const restoreFocusFromSurface = (surface, returnFocusRef) => {
  if (!surface || typeof document === 'undefined') return

  const activeElement = document.activeElement
  if (!activeElement || !surface.contains(activeElement)) return

  const returnTarget = returnFocusRef?.current
  const canReturnFocus =
    returnTarget?.isConnected &&
    !returnTarget.disabled &&
    returnTarget.getAttribute?.('aria-disabled') !== 'true' &&
    typeof returnTarget.focus === 'function'

  if (canReturnFocus) {
    returnTarget.focus()
    return
  }

  activeElement.blur?.()
}

const useRestoreFocusOnExit = ({
  surfaceRef,
  entered,
  returnFocusRef,
  surfaceKey,
}) => {
  useLayoutEffect(() => {
    const surface = surfaceRef.current
    if (!entered) restoreFocusFromSurface(surface, returnFocusRef)

    return () => restoreFocusFromSurface(surface, returnFocusRef)
  }, [entered, returnFocusRef, surfaceKey, surfaceRef])
}

export default useRestoreFocusOnExit
