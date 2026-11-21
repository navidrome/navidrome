import { createContext, useContext } from 'react'

const noop = () => {}

export const defaultCastContextValue = {
  castAvailable: false,
  isCasting: false,
  isConnecting: false,
  deviceName: '',
  isCastingRef: { current: false },
  requestSession: noop,
  stopCasting: noop,
  onPlayIndexChange: noop,
}

export const CastContext = createContext(defaultCastContextValue)

export const useCast = () => useContext(CastContext)
