/* eslint-disable react-refresh/only-export-components */
import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
} from 'react'

const noop = () => {}

const fallbackContext = {
  desktopLyricsProps: null,
  setDesktopLyricsProps: noop,
}

const LyricsLayoutContext = createContext(fallbackContext)

export const LyricsLayoutProvider = ({ children }) => {
  const [desktopLyricsProps, setDesktopLyricsPropsState] = useState(null)

  const setDesktopLyricsProps = useCallback((nextProps) => {
    setDesktopLyricsPropsState(nextProps || null)
  }, [])

  const value = useMemo(
    () => ({
      desktopLyricsProps,
      setDesktopLyricsProps,
    }),
    [desktopLyricsProps, setDesktopLyricsProps],
  )

  return (
    <LyricsLayoutContext.Provider value={value}>
      {children}
    </LyricsLayoutContext.Provider>
  )
}

export const useLyricsLayout = () => useContext(LyricsLayoutContext)
