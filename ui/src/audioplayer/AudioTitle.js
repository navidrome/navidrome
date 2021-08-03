import React from 'react'
import { useMediaQuery } from '@material-ui/core'
import { Link } from 'react-router-dom'
import clsx from 'clsx'
import { QualityInfo } from '../common'
import useStyle from './styles'

const AudioTitle = React.memo(({ audioInfo, isMobile }) => {
  const classes = useStyle()
  const className = classes.audioTitle
  const isDesktop = useMediaQuery('(min-width:810px)')

  if (!audioInfo.song) {
    return ''
  }

  const song = audioInfo.song
  const qi = { suffix: song.suffix, bitRate: song.bitRate }

  return (
    <Link to={`/album/${song.albumId}/show`} className={className}>
      <span>
        <span className={clsx(classes.songTitle, 'songTitle')}>
          {song.title}
        </span>
        {isDesktop && (
          <QualityInfo record={qi} className={classes.qualityInfo} />
        )}
      </span>
      {!isMobile && (
        <span className={clsx(classes.songInfo)}>
          {`${song.artist} - ${song.album}` +
            (song.year ? ` - ${song.year}` : '')}
        </span>
      )}
      {isMobile && (
        <>
          <span className={clsx(classes.songInfo, classes.songArtist)}>
            {`${song.artist}`}
          </span>
          <span className={clsx(classes.songInfo, classes.songAlbum)}>
            {song.year ? `${song.album} - ${song.year}` : `${song.album}`}
          </span>
        </>
      )}
    </Link>
  )
})

export default AudioTitle
