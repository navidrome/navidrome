import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'

export const ArtistLinkField = ({ record, className }) => {
  const filter = { artist_id: record.albumArtistId }
  const url = `/album?filter=${JSON.stringify(
    filter
  )}&order=ASC&sort=maxYear&displayedFilters={"compilation":true}`
  return (
    <Link to={url} onClick={(e) => e.stopPropagation()} className={className}>
      {record.albumArtist}
    </Link>
  )
}

ArtistLinkField.propTypes = {
  className: PropTypes.string,
  source: PropTypes.string,
}

ArtistLinkField.defaultProps = {
  source: 'artistId',
  addLabel: true,
}
