import config from '../config'

const baseUrl = (path) => {
  const base = config.baseURL || ''
  const parts = [base]
  parts.push(path.replace(/^\//, ''))
  return parts.join('/')
}

export default baseUrl
