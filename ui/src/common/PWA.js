import useCurrentTheme from '../themes/useCurrentTheme'

const ChangeColor = () => {
  const theme = useCurrentTheme()
  const query = document.querySelector("meta[name='theme-color']")

  try {
    const color =
      theme.palette.primary.light !== undefined
        ? theme.palette.primary.light
        : theme.palette.primary.main
    if (query.content !== color) {
      query.setAttribute('content', color)
    }
  } catch {
    query.setAttribute('content', '#ffffff')
  }
  return null
}

export default ChangeColor
