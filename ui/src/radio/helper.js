export function songFromRadio(radio, resourceName) {
  if (!radio) {
    return undefined
  }

  const url = new URL(radio.homePageUrl ?? radio.streamUrl)
  url.pathname = '/favicon.ico'

  return {
    ...radio,
    title: radio.name,
    album: radio.homePageUrl || resourceName,
    artist: resourceName,
    cover: url.toString(),
    isRadio: true,
  }
}
