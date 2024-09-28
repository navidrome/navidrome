import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'
import { withWidth } from '@material-ui/core'
import { useGetHandleArtistClick } from './useGetHandleArtistClick'

export const ArtistLinkField = withWidth()(({
  record,
  className,
  width,
  source,
}) => {
  const artistLink = useGetHandleArtistClick(width)

  const id = record[source + 'Id']
  return (
    <>
      {id ? (
        <Link
          to={artistLink(id)}
          onClick={(e) => e.stopPropagation()}
          className={className}
        >
          {record[source]}
        </Link>
      ) : (
        record[source]
      )}
    </>
  )
})

ArtistLinkField.propTypes = {
  record: PropTypes.object,
  className: PropTypes.string,
  source: PropTypes.string,
}

ArtistLinkField.defaultProps = {
  addLabel: true,
  source: 'albumArtist',
}
