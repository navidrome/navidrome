import { Link } from 'react-admin'
import React from 'react'

export const ArtistLinkField = (props) => {
  const filter = { artist_id: props.record.albumArtistId }
  const url = `/album?filter=${JSON.stringify(
    filter
  )}&order=ASC&sort=maxYear&displayedFilters={"compilation":true}`
  return (
    <Link to={url} onClick={(e) => e.stopPropagation()}>
      {props.record.albumArtist}
    </Link>
  )
}

ArtistLinkField.defaultProps = {
  source: 'artistId',
  addLabel: true
}
