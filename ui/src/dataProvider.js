import { fetchUtils } from 'react-admin'
import jsonServerProvider from 'ra-data-json-server'

const baseUrl = '/app/api'

const httpClient = (url, options = {}) => {
  url = url.replace(baseUrl + '/albumSong', baseUrl + '/song')
  if (!options.headers) {
    options.headers = new Headers({ Accept: 'application/json' })
  }
  const token = localStorage.getItem('token')
  if (token) {
    options.headers.set('Authorization', `Bearer ${token}`)
  }
  return fetchUtils.fetchJson(url, options).then((response) => {
    const token = response.headers.get('authorization')
    if (token) {
      localStorage.setItem('token', token)
      localStorage.removeItem('initialAccountCreation')
    }
    return response
  })
}

const dataProvider = jsonServerProvider(baseUrl, httpClient)

export default dataProvider
