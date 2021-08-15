import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'
import { useAlbumsPerPage } from './index'
import { withWidth } from '@material-ui/core'
import ArtistView from './ArtistDetail'
import { Route } from 'react-router-dom'

export const useGetHandleArtistClick = (width) => {
  const [perPage] = useAlbumsPerPage(width)

  // return (id) => {
  //   return `/album?filter={"artist_id":"${id}"}&order=ASC&sort=maxYear&displayedFilters={"compilation":true}&perPage=${perPage}`
  // }
  return (id) => {
    return `/iartist/${id}`
  }
}

export const ArtistLinkField = withWidth()(({ record, className, width }) => {
  const artistLink = useGetHandleArtistClick(width)

  const handleclick = (props) => {
    <ArtistView artist={props.artistId} />
  }

  return (
    <Link
      to={artistLink(record.albumArtistId)}
      onClick={handleclick(record)}
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
