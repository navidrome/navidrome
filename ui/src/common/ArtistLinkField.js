import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'
import { withWidth } from '@material-ui/core'
import { useAlbumsPerPage } from './index'
import config from '../config'

export const useGetHandleArtistClick = (width) => {
  const [perPage] = useAlbumsPerPage(width)
  return (id) => {
    return config.devShowArtistPage && id !== config.variousArtistsId
      ? `/artist/${id}/show`
      : `/album?filter={"artist_id":"${id}"}&order=ASC&sort=maxYear&displayedFilters={"compilation":true}&perPage=${perPage}`
  }
}

export const ArtistLinkField = withWidth()(({ record, className, width }) => {
  const artistLink = useGetHandleArtistClick(width)

  return (
    <Link
      to={artistLink(record.albumArtistId)}
      onClick={(e) => e.stopPropagation()}
      className={className}
    >
      {record.albumArtist}
    </Link>
  )
})

ArtistLinkField.propTypes = {
  record: PropTypes.object,
  className: PropTypes.string,
}

ArtistLinkField.defaultProps = {
  addLabel: true,
}
