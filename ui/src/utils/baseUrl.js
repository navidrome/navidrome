import config from '../config'

export const baseUrl = (path) => {
  const base = config.baseURL || ''
  const parts = [base]
  parts.push(path.replace(/^\//, ''))
  return parts.join('/')
}
