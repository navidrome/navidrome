import { fetchUtils } from 'react-admin'
import jsonServerProvider from 'ra-data-json-server'
import baseUrl from './utils/baseUrl'

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
      localStorage.removeItem('initialAccountCreation')
    }
    return response
  })
}

const dataProvider = jsonServerProvider(restUrl, httpClient)

export default dataProvider
