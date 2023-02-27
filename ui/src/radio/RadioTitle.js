import React from 'react'
import clsx from 'clsx'

import useStyle from '../audioplayer/styles'
import { Link } from 'react-admin'
import FixMetadataButton from './FixMetadataButton'

const RadioTitle = React.memo(({ id, isMobile, name, metadata, onFix }) => {
  const classes = useStyle()
  const className = classes.audioTitle

  const isMissingArtist = metadata.title && !metadata.artist

  return (
    <Link to={`/radio/${id}/show`} className={className}>
      <span>
        <span className={clsx(classes.songTitle, 'songTitle')}>
          {metadata.title}
        </span>
      </span>
      {isMobile ? (
        <>
          <span className={classes.songInfo}>
            <span className="songArtist">
              {isMissingArtist ? <FixMetadataButton /> : metadata.artist}
            </span>
          </span>
          <span className={clsx(classes.songInfo, classes.songAlbum)}>
            <span className="songAlbum">{name}</span>
          </span>
        </>
      ) : (
        <span className={classes.songInfo}>
          {isMissingArtist ? (
            <span className="songArtist">
              {<FixMetadataButton onFix={onFix} />}
            </span>
          ) : (
            <>
              <span className="songArtist">{metadata.artist}</span> -{' '}
              <span className="songAlbum">{name}</span>
            </>
          )}
        </span>
      )}
    </Link>
  )
})

export default RadioTitle
