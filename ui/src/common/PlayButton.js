import React from 'react'
import PropTypes from 'prop-types'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { IconButton } from '@material-ui/core'
import { useDispatch } from 'react-redux'
import { useDataProvider } from 'react-admin'
import { playTracks } from '../actions'

export const PlayButton = ({ record, size, className }) => {
  let extractSongsData = function (response) {
    const data = response.data.reduce(
      (acc, cur) => ({ ...acc, [cur.id]: cur }),
      {},
    )
    const ids = response.data.map((r) => r.id)
    return { data, ids }
  }
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const playAlbum = (record) => {
    dataProvider
      .getList('song', {
        pagination: { page: 1, perPage: -1 },
        sort: { field: 'releaseDate, discNumber, trackNumber', order: 'ASC' },
        filter: {
          album_id: record.id,
          release_date: record.releaseDate,
          disc_number: record.discNumber,
        },
      })
      .then((response) => {
        let { data, ids } = extractSongsData(response)
        dispatch(playTracks(data, ids))
      })
  }

  return (
    <IconButton
      onClick={(e) => {
        e.stopPropagation()
        e.preventDefault()
        playAlbum(record)
      }}
      aria-label="play"
      className={className}
      size={size}
    >
      <PlayArrowIcon fontSize={size} />
    </IconButton>
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
