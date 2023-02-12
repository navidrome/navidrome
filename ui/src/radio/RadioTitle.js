import React from 'react'
import clsx from 'clsx'

import useStyle from '../audioplayer/styles'
import { Link } from 'react-admin'

const RadioTitle = React.memo(({ homePageUrl, isMobile, name, metadata }) => {
  const classes = useStyle()
  const className = classes.audioTitle

  let artist
  let title

  if (metadata.StreamTitle) {
    const split = metadata.StreamTitle.split(' - ')
    artist = split[0]
    title = split.slice(1).join(' - ')
  } else {
    title = homePageUrl
    artist = ''
  }

  return (
    <Link
      to={`/radio?displayedFilters={}&filter={"name":"${name}"}`}
      className={className}
    >
      <span>
        <span className={clsx(classes.songTitle, 'songTitle')}>{title}</span>
      </span>
      {isMobile ? (
        <>
          <span className={classes.songInfo}>
            <span className="songArtist">{artist}</span>
          </span>
          <span className={clsx(classes.songInfo, classes.songAlbum)}>
            <span className="songAlbum">{name}</span>
          </span>
        </>
      ) : (
        <span className={classes.songInfo}>
          <span className="songArtist">{artist}</span> -{' '}
          <span className="songAlbum">{name}</span>
        </span>
      )}
    </Link>
  )
})

export default RadioTitle
