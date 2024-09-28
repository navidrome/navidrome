import { useEffect } from 'react'
import useCurrentTheme from './themes/useCurrentTheme'

const useChangeThemeColor = () => {
  const theme = useCurrentTheme()
  const color =
    theme.palette?.primary?.light || theme.palette?.primary?.main || '#ffffff'
  useEffect(() => {
    const themeColor = document.querySelector("meta[name='theme-color']")
    themeColor.setAttribute('content', color)
  }, [color])
}

export default useChangeThemeColor
