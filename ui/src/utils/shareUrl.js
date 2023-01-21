import config from '../config'

export const shareUrl = (path) => {
  const url = new URL(config.publicBaseUrl + '/' + path, window.location.href)
  return url.href
}
