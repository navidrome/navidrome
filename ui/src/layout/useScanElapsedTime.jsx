import { useEffect, useState, useRef } from 'react'
import { useInterval } from '../common'

export const useScanElapsedTime = (scanning, elapsedTime) => {
  const [elapsed, setElapsed] = useState(Number(elapsedTime) || 0)
  const prevScanningRef = useRef(scanning)

  useEffect(() => {
    // Only update from server when scan starts or stops
    const prevScanning = prevScanningRef.current
    if (!prevScanning && scanning) {
      // Scan just started - initialize with server value
      setElapsed(Number(elapsedTime) || 0)
    } else if (prevScanning && !scanning) {
      // Scan just finished - use final server value
      setElapsed(Number(elapsedTime) || 0)
    }
    // Update ref for next comparison
    prevScanningRef.current = scanning
  }, [scanning, elapsedTime])

  useInterval(() => setElapsed((prev) => prev + 1e9), scanning ? 1000 : null)

  return elapsed
}
