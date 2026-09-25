import { useEffect, useState } from 'react'

const useEnterExitTransition = (active, transitionMs) => {
  const [rendered, setRendered] = useState(active)
  const [entered, setEntered] = useState(active)

  useEffect(() => {
    if (active) {
      setRendered(true)
      const frameId = window.requestAnimationFrame(() => setEntered(true))
      return () => window.cancelAnimationFrame(frameId)
    }

    setEntered(false)
    const timerId = window.setTimeout(() => setRendered(false), transitionMs)
    return () => window.clearTimeout(timerId)
  }, [active, transitionMs])

  return { rendered, entered }
}

export default useEnterExitTransition
