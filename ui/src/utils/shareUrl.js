import config from '../config'

export const shareUrl = (path) => {
  const url = new URL(config.shareBaseUrl + '/' + path, window.location.href)
  return url.href
}
