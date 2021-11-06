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

const songsFilteredByArtist = (artist) => {
  return `/song?filter={"artist":"${artist}"}`
}

export const ArtistLinkField = withWidth()(
  ({ record, className, width, source }) => {
    const artistLink = useGetHandleArtistClick(width)

    const id = record[source + 'Id']
    const link = id ? artistLink(id) : songsFilteredByArtist(record[source])
    return (
      <Link
        to={link}
        onClick={(e) => e.stopPropagation()}
        className={className}
      >
        {record[source]}
      </Link>
    )
  }
)

ArtistLinkField.propTypes = {
  record: PropTypes.object,
  className: PropTypes.string,
  source: PropTypes.string,
}

ArtistLinkField.defaultProps = {
  addLabel: true,
  source: 'albumArtist',
}
