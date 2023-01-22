import config from '../config'

export const baseUrl = (path) => {
  const base = config.baseURL || ''
  const parts = [base]
  parts.push(path.replace(/^\//, ''))
  return parts.join('/')
}

export const shareUrl = (path) => {
  const url = new URL(config.publicBaseUrl + '/' + path, window.location.href)
  return url.href
}

export const shareStreamUrl = (id) => {
  return baseUrl(config.publicBaseUrl + '/s/' + id)
}

export const shareCoverUrl = (id) => {
  return baseUrl(config.publicBaseUrl + '/img/' + id + '?size=300')
}

export const docsUrl = (path) => `https://www.navidrome.org${path}`
