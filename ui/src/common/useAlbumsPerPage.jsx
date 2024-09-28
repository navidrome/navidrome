import { useSelector } from 'react-redux'

const getPerPage = (width) => {
  if (width === 'xs') return 12
  if (width === 'sm') return 12
  if (width === 'md') return 12
  if (width === 'lg') return 18
  return 36
}

const getPerPageOptions = (width) => {
  const options = [3, 6, 12]
  if (width === 'xs') return [12]
  if (width === 'sm') return [12]
  if (width === 'md') return options.map((v) => v * 4)
  return options.map((v) => v * 6)
}

export const useAlbumsPerPage = (width) => {
  const perPage =
    useSelector(
      (state) => state?.admin.resources?.album?.list?.params?.perPage,
    ) || getPerPage(width)

  return [perPage, getPerPageOptions(width)]
}
