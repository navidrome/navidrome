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
  const artists = record['participations']
    ? record['participations'][source]
    : [{ name: record[source], id: record[source + 'Id'] }]

  // When showing artists for a track, add any remixers to the list of artists
  if (
    source === 'artist' &&
    record['participations'] &&
    record['participations']['remixer']
  ) {
    record['participations']['remixer'].forEach((remixer) => {
      artists.push(remixer)
    })
  }

  // Dedupe artists
  const seen = new Set()
  const dedupedArtists = []
  artists.forEach((artist) => {
    if (!seen.has(artist.id)) {
      seen.add(artist.id)
      dedupedArtists.push(artist)
    }
  })

  return (
    <>
      {dedupedArtists.map((artist, index) => {
        const id = artist.id
        return (
          <>
            <Link
              to={artistLink(id)}
              onClick={(e) => e.stopPropagation()}
              className={className}
            >
              {artist.name}
            </Link>
            {index < dedupedArtists.length - 1 && ' • '}
          </>
        )
      })}
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
