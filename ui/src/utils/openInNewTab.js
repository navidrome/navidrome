export const openInNewTab = (url) => {
  const win = window.open(url, '_blank')
  win.focus()
  return win
}
