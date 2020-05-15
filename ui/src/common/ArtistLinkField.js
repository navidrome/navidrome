import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'

const artistLink = (id) => {
  return `/album?filter={"artist_id":"${id}"}&order=ASC&sort=maxYear&displayedFilters={"compilation":true}`
}

const ArtistLinkField = ({ record, className }) => {
  return (
    <Link
      to={artistLink(record.albumArtistId)}
      onClick={(e) => e.stopPropagation()}
      className={className}
    >
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

export { artistLink }

export default ArtistLinkField
