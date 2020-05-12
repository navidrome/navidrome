import { fetchUtils } from 'react-admin'
import baseUrl from '../utils/baseUrl'
import config from '../config'

const customAuthorizationHeader = 'X-ND-Authorization'

const httpClient = (url, options = {}) => {
  url = baseUrl(url)
  if (!options.headers) {
    options.headers = new Headers({ Accept: 'application/json' })
  }
  const token = localStorage.getItem('token')
  if (token) {
    options.headers.set(customAuthorizationHeader, `Bearer ${token}`)
  }
  return fetchUtils.fetchJson(url, options).then((response) => {
    const token = response.headers.get(customAuthorizationHeader)
    if (token) {
      localStorage.setItem('token', token)
      // Avoid going to create admin dialog after logout/login without a refresh
      config.firstTime = false
    }
    return response
  })
}

export default httpClient
