import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'
import { withWidth } from '@material-ui/core'

export const useGetHandleArtistClick = (width) => {
  return (id) => {
    return `/artist/${id}`
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
