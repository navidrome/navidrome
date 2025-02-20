import { useSelector } from 'react-redux'
import { pageSizeMultiplier } from '../utils/pageSizes'

const getPerPage = (width) => {
  let baseSize
  if (width === 'xs') baseSize = 12
  else if (width === 'sm') baseSize = 12
  else if (width === 'md') baseSize = 12
  else if (width === 'lg') baseSize = 18
  else baseSize = 36
  return baseSize * pageSizeMultiplier()
}

const getPerPageOptions = (width) => {
  const options = [3, 6, 12]
  let sizeOptions
  if (width === 'xs') sizeOptions = [12]
  else if (width === 'sm') sizeOptions = [12]
  else if (width === 'md') sizeOptions = options.map((v) => v * 4)
  else sizeOptions = options.map((v) => v * 6)
  return sizeOptions.map((size) => size * pageSizeMultiplier())
}

export const useAlbumsPerPage = (width) => {
  const perPage =
    useSelector(
      (state) => state?.admin.resources?.album?.list?.params?.perPage,
    ) || getPerPage(width)

  return [perPage, getPerPageOptions(width)]
}
