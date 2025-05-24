import { useEffect, useState } from 'react'
import { useInterval } from '../common'

export const useScanElapsedTime = (scanning, elapsedTime) => {
  const [elapsed, setElapsed] = useState(Number(elapsedTime) || 0)

  useEffect(() => {
    setElapsed(Number(elapsedTime) || 0)
  }, [elapsedTime])

  useInterval(() => setElapsed((prev) => prev + 1e9), scanning ? 1000 : null)

  return elapsed
}
