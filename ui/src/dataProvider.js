import { fetchUtils } from 'react-admin'
import jsonServerProvider from 'ra-data-json-server'
import baseUrl from './utils/baseUrl'
import config from './config'

const restUrl = '/app/api'
const customAuthorizationHeader = 'X-ND-Authorization'

const httpClient = (url, options = {}) => {
  url = baseUrl(url)
  url = url.replace(restUrl + '/albumSong', restUrl + '/song')
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

const dataProvider = jsonServerProvider(restUrl, httpClient)

export default dataProvider
