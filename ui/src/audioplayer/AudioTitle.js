import React from 'react'
import { useMediaQuery } from '@material-ui/core'
import { Link } from 'react-router-dom'
import clsx from 'clsx'
import { QualityInfo } from '../common'
import useStyle from './styles'

const AudioTitle = React.memo(({ audioInfo, gainInfo, isMobile }) => {
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

  let link

  if (audioInfo.isRadio) {
    if (audioInfo.infoId) {
      const display = JSON.stringify({ id: true })
      const filter = JSON.stringify({ id: audioInfo.infoId })
      link = `/radioInfo?${display}=${display}&filter=${filter}&page=1`
    } else {
      link = `/radio/${audioInfo.trackId}/show`
    }
  } else {
    link = `/album/${song.albumId}/show`
  }

  return (
    <Link to={link} className={className}>
      <span>
        <span className={clsx(classes.songTitle, 'songTitle')}>
          {song.title}
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
            <span className={'songArtist'}>{song.artist}</span>
          </span>
          <span className={clsx(classes.songInfo, classes.songAlbum)}>
            <span className={'songAlbum'}>{song.album}</span>
            {song.year ? ` - ${song.year}` : ''}
          </span>
        </>
      ) : (
        <span className={classes.songInfo}>
          <span className={'songArtist'}>{song.artist}</span> -{' '}
          <span className={'songAlbum'}>{song.album}</span>
          {song.year ? ` - ${song.year}` : ''}
        </span>
      )}
    </Link>
  )
})

export default AudioTitle
