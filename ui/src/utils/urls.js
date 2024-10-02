import config from '../config'

export const baseUrl = (path) => {
  const base = config.baseURL || ''
  const parts = [base]
  parts.push(path.replace(/^\//, ''))
  return parts.join('/')
}

export const shareUrl = (path) => {
  if (config.shareURL !== '') {
    const base = config.shareURL || ''
    const parts = [base]
    parts.push(path.replace(/^\//, ''))
    return parts.join('/')
  }
  return baseUrl(path)
}

export const sharePlayerUrl = (id) => {
  const url = new URL(
    shareUrl(config.publicBaseUrl + '/' + id),
    window.location.href,
  )
  return url.href
}

export const shareStreamUrl = (id) => {
  return shareUrl(config.publicBaseUrl + '/s/' + id)
}

export const shareDownloadUrl = (id) => {
  return shareUrl(config.publicBaseUrl + '/d/' + id)
}

export const shareCoverUrl = (id) => {
  return shareUrl(config.publicBaseUrl + '/img/' + id + '?size=300')
}

export const docsUrl = (path) => `https://www.navidrome.org${path}`
