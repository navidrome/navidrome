import { useAlbumsPerPage } from './useAlbumsPerPage'
import config from '../config.js'

export const useGetHandleArtistClick = (width) => {
  const [perPage] = useAlbumsPerPage(width)
  return (id) => {
    return config.devShowArtistPage && id !== config.variousArtistsId
      ? `/artist/${id}/show`
      : `/album?filter={"artist_id":"${id}"}&order=ASC&sort=max_year&displayedFilters={"compilation":true}&perPage=${perPage}`
  }
}
