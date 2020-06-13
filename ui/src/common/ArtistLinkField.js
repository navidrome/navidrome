import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'
import { useAlbumsPerPage } from './index'
import { withWidth } from '@material-ui/core'

const useGetHandleArtistClick = (width) => {
  const [perPage] = useAlbumsPerPage(width)

  return (id) => {
    return `/album?filter={"artist_id":"${id}"}&order=ASC&sort=maxYear&displayedFilters={"compilation":true}&perPage=${perPage}`
  }
}

const ArtistLinkField = ({ record, className, width }) => {
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
}

ArtistLinkField.propTypes = {
  record: PropTypes.object,
  className: PropTypes.string,
}

ArtistLinkField.defaultProps = {
  addLabel: true,
}

export { useGetHandleArtistClick }

export default withWidth()(ArtistLinkField)
