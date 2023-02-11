import React from 'react'
import { useMediaQuery } from '@material-ui/core'
import { Link } from 'react-router-dom'
import clsx from 'clsx'
import { QualityInfo } from '../common'
import useStyle from './styles'

const AudioTitle = React.memo(({ audioInfo, gainInfo, isMobile, radioInfo }) => {
  const classes = useStyle()
  const className = classes.audioTitle
  const isDesktop = useMediaQuery('(min-width:810px)')

  if (!audioInfo.song) {
    return ''
  }

  const song = audioInfo.song
  const qi = {
    suffix: song.suffix,
    bitRate: song.bitRate,
    albumGain: song.rgAlbumGain,
    trackGain: song.rgTrackGain,
  }

  let album;
  let artist;
  let title;

  if (radioInfo) {
    const split = radioInfo.StreamTitle.split(" - ")
    artist = split[0]
    title = split.slice(1).join(" - ")
    album = song.title
  } else {
    artist = song.artist
    title = song.title
    album = song.album
  }

  return (
    <Link
      to={
        radioInfo
          ? `/radio/${audioInfo.trackId}/show`
          : `/album/${song.albumId}/show`
      }
      className={className}
    >
      <span>
        <span className={clsx(classes.songTitle, 'songTitle')}>
          {title}
        </span>
        {isDesktop && (
          <QualityInfo
            record={qi}
            className={classes.qualityInfo}
            {...gainInfo}
          />
        )}
      </span>
      {isMobile ? (
        <>
          <span className={classes.songInfo}>
            <span className={'songArtist'}>{artist}</span>
          </span>
          <span className={clsx(classes.songInfo, classes.songAlbum)}>
            <span className={'songAlbum'}>{album}</span>
            {song.year ? ` - ${song.year}` : ''}
          </span>
        </>
      ) : (
        <span className={classes.songInfo}>
          <span className={'songArtist'}>{artist}</span> -{' '}
          <span className={'songAlbum'}>{album}</span>
          {song.year ? ` - ${song.year}` : ''}
        </span>
      )}
    </Link>
  )
})

export default AudioTitle
