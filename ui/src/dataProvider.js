import { fetchUtils } from 'react-admin'
import jsonServerProvider from 'ra-data-json-server'
import baseUrl from './utils/baseUrl'

const restUrl = '/app/api'

const httpClient = (url, options = {}) => {
  url = baseUrl(url)
  url = url.replace(restUrl + '/albumSong', restUrl + '/song')
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

const dataProvider = jsonServerProvider(restUrl, httpClient)

export default dataProvider
