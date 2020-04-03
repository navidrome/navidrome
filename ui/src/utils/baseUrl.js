const baseUrl = (path) => {
  const base = localStorage.getItem('baseURL') || ''
  const parts = [base]
  parts.push(path.replace(/^\//, ''))
  return parts.join('/')
}

export default baseUrl
