import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import PauseIcon from '@material-ui/icons/Pause'
import { IconButton } from '@material-ui/core'
import { useDispatch, useSelector } from 'react-redux'
import { useDataProvider, useTranslate } from 'react-admin'
import { pausePlayer, playTracks, recentAlbum } from '../actions'
import { Button } from 'react-admin'
import { get } from 'lodash'
import { playingInAlbumOrPlaylist } from './index'

export const PlayButton = ({
  record,
  size,
  className,
  buttonType,
  handlePlay,
}) => {
  let extractSongsData = function (response) {
    const data = response.data.reduce(
      (acc, cur) => ({ ...acc, [cur.id]: cur }),
      {}
    )
    const ids = response.data.map((r) => r.id)
    return { data, ids }
  }
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const playAlbum = (record) => {
    dataProvider
      .getList('albumSong', {
        pagination: { page: 1, perPage: -1 },
        sort: { field: 'discNumber, trackNumber', order: 'ASC' },
        filter: { album_id: record.id, disc_number: record.discNumber },
      })
      .then((response) => {
        let { data, ids } = extractSongsData(response)
        dispatch(playTracks(data, ids, undefined, record.id))
      })
  }

  const ButtonType = buttonType === 'button' ? Button : IconButton
  const translate = useTranslate()
  const [playing, setPlaying] = useState(false)

  const currentTrack = useSelector((state) => get(state, 'queue.current', {}))
  const albumOrPlaylistId = useSelector((state) =>
    get(state, 'queue.recentAlbumOrPlaylist.id', '')
  )
  const songAlbumOrPlaylistId = useSelector((state) =>
    get(state, 'queue.albumOrPlaylistId', '')
  )

  useEffect(() => {
    setPlaying(
      playingInAlbumOrPlaylist(
        currentTrack,
        albumOrPlaylistId,
        songAlbumOrPlaylistId
      )
    )
  }, [currentTrack, albumOrPlaylistId, songAlbumOrPlaylistId])

  console.log(playing)

  return (
    <ButtonType
      onClick={(e) => {
        e.stopPropagation()
        e.preventDefault()
        playing
          ? dispatch(pausePlayer())
          : buttonType === 'button'
          ? handlePlay()
          : playAlbum(record)
        dispatch(recentAlbum(record.id))
      }}
      aria-label="play"
      className={className}
      size={size}
      label={
        playing
          ? translate('resources.album.actions.pause')
          : translate('resources.album.actions.playAll')
      }
    >
      {playing ? (
        <PauseIcon fontSize={size} />
      ) : (
        <PlayArrowIcon fontSize={size} />
      )}
    </ButtonType>
  )
}

PlayButton.propTypes = {
  record: PropTypes.object.isRequired,
  size: PropTypes.string,
  className: PropTypes.string,
}

PlayButton.defaultProps = {
  size: 'small',
}
