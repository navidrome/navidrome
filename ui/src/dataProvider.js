import { fetchUtils } from 'react-admin'
import jsonServerProvider from 'ra-data-json-server'

const httpClient = (url, options = {}) => {
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

const dataProvider = jsonServerProvider('/app/api', httpClient)

export default dataProvider
