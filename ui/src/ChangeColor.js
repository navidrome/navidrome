import { useEffect } from 'react'
import useCurrentTheme from './themes/useCurrentTheme'

const ChangeColor = () => {
  const theme = useCurrentTheme()
  useEffect(() => {
    const query = document.querySelector("meta[name='theme-color']")
    try {
      const color =
        theme.palette.primary.light !== undefined
          ? theme.palette.primary.light
          : theme.palette.primary.main
      query.setAttribute('content', color)
    } catch {
      query.setAttribute('content', '#ffffff')
    }
  })
  return null
}

export default ChangeColor
