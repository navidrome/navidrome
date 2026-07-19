export const defaultRowsPerPageOptions = [15, 25, 50]

const key = (resource) => `perPage.${resource}`

export const getStoredPerPage = (resource, options, fallback) => {
  const stored = parseInt(localStorage.getItem(key(resource)), 10)
  return options.includes(stored) ? stored : fallback
}

export const setStoredPerPage = (resource, perPage) =>
  localStorage.setItem(key(resource), String(perPage))
