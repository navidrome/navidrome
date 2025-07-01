import { useEffect, useState, useRef } from 'react'
import { useInterval } from '../common'

export const useScanElapsedTime = (scanning, elapsedTime) => {
  const [elapsed, setElapsed] = useState(Number(elapsedTime) || 0)
  const prevScanningRef = useRef(scanning)

  useEffect(() => {
    const prevScanning = prevScanningRef.current
    const serverElapsed = Number(elapsedTime) || 0

    if (scanning !== prevScanning) {
      // Scan has just started or stopped - sync with server value
      setElapsed(serverElapsed)
    } else if (!scanning) {
      // Not scanning -> always reflect server value (initial load or after finish)
      setElapsed(serverElapsed)
    }

    prevScanningRef.current = scanning
  }, [scanning, elapsedTime])

  useInterval(() => setElapsed((prev) => prev + 1e9), scanning ? 1000 : null)

  return elapsed
}
