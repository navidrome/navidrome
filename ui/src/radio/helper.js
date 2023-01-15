export function songFromRadio(radio) {
  if (!radio) {
    return undefined
  }

  let cover

  try {
    const url = new URL(radio.homePageUrl ?? radio.streamUrl)
    url.pathname = '/favicon.ico'
    cover = url.toString()
  } catch (_) {}

  return {
    ...radio,
    title: radio.name,
    album: radio.homePageUrl || radio.name,
    artist: radio.name,
    cover,
    isRadio: true,
  }
}
