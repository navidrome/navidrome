import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-admin'
import { withWidth } from '@material-ui/core'
import { useGetHandleArtistClick } from './useGetHandleArtistClick'
import { intersperse } from '../utils/index.js'
import { useDispatch } from 'react-redux'
import { closeExtendedInfoDialog } from '../actions/dialogs.js'

const ALink = withWidth()((props) => {
  const { artist, width, ...rest } = props
  const artistLink = useGetHandleArtistClick(width)
  const dispatch = useDispatch()

  return (
    <Link
      key={artist.id}
      to={artistLink(artist.id)}
      onClick={(e) => {
        e.stopPropagation()
        dispatch(closeExtendedInfoDialog())
      }}
      {...rest}
    >
      {artist.name}
      {artist.subroles?.length > 0 ? ` (${artist.subroles.join(', ')})` : ''}
    </Link>
  )
})

const parseAndReplaceArtists = (
  displayAlbumArtist,
  albumArtists,
  className,
) => {
  let result = []
  let lastIndex = 0

  albumArtists?.forEach((artist) => {
    const index = displayAlbumArtist.indexOf(artist.name, lastIndex)
    if (index !== -1) {
      // Add text before the artist name
      if (index > lastIndex) {
        result.push(displayAlbumArtist.slice(lastIndex, index))
      }
      // Add the artist link
      result.push(<ALink artist={artist} className={className} />)
      lastIndex = index + artist.name.length
    }
  })

  if (lastIndex === 0) {
    return []
  }

  // Add any remaining text after the last artist name
  if (lastIndex < displayAlbumArtist.length) {
    result.push(displayAlbumArtist.slice(lastIndex))
  }

  return result
}

export const ArtistLinkField = ({ record, className, limit, source }) => {
  const role = source.toLowerCase()

  // Get artists array with fallback
  let artists = []
  if (record?.participants?.[role]) {
    artists = [...record.participants[role]]

    // Add remixers when applicable
    if (role === 'artist' && record.participants.remixer?.length) {
      artists = [...artists, ...record.participants.remixer]
    }
  } else if (record[source]) {
    artists = [{ name: record[source], id: record[source + 'Id'] }]
  }

  // Special handling for albumartist
  if (role === 'albumartist') {
    const artistsLinks = parseAndReplaceArtists(
      record[source],
      artists,
      className,
    )
    if (artistsLinks.length > 0) {
      return <div className={className}>{artistsLinks}</div>
    }
  }

  // Dedupe artists and collect subroles
  const seen = new Map()
  const dedupedArtists = []
  let limitedShow = false

  for (const artist of artists) {
    if (!artist?.id) continue

    if (!seen.has(artist.id)) {
      if (dedupedArtists.length < limit) {
        seen.set(artist.id, dedupedArtists.length)
        dedupedArtists.push({
          ...artist,
          subroles: artist.subRole ? [artist.subRole] : [],
        })
      } else {
        limitedShow = true
      }
    } else {
      const position = seen.get(artist.id)
      const existing = dedupedArtists[position]
      if (artist.subRole && !existing.subroles.includes(artist.subRole)) {
        existing.subroles.push(artist.subRole)
      }
    }
  }

  // Create artist links
  const artistsList = dedupedArtists.map((artist) => (
    <ALink artist={artist} className={className} key={artist.id} />
  ))

  if (limitedShow) {
    artistsList.push(<span key="more">...</span>)
  }

  return <>{intersperse(artistsList, ' • ')}</>
}

ArtistLinkField.propTypes = {
  limit: PropTypes.number,
  record: PropTypes.object,
  className: PropTypes.string,
  source: PropTypes.string,
}

ArtistLinkField.defaultProps = {
  addLabel: true,
  limit: 3,
  source: 'albumArtist',
}
