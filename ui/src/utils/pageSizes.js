/* Groups page-size logic, since it's used in several different parts of the UI */
export const defaultPageSizeMultiplier = '1'
export const pageSizeMultipliers = ['1', '4', '10', '1000']
export const defaultBasePageSize = 15
export const defaultBasePageSizes = [15, 25, 50]

// Stored as a string for consistency, so we provide a getter. Function, not
// constant, because it's read from localStorage (which might change)
export const pageSizeMultiplier = () =>
  parseInt(
    localStorage.getItem('pageSizeMultiplier') || defaultPageSizeMultiplier,
  )

export const defaultPageSize = () => defaultBasePageSize * pageSizeMultiplier()
export const defaultPageSizes = () =>
  defaultBasePageSizes.map((size) => size * pageSizeMultiplier())
