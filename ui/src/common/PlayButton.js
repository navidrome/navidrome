import React from 'react'
import PropTypes from 'prop-types'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { IconButton } from '@material-ui/core'
import { useDispatch } from 'react-redux'
import { useDataProvider } from 'react-admin'
import { playTracks } from '../actions'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles({
  icon: {
    color: (props) => props.color,
  },
})

export const PlayButton = ({ record, color, size, ...rest }) => {
  const classes = useStyles({ color })
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
      {...rest}
      className={classes.icon}
      size={size}
    >
      <PlayArrowIcon fontSize={size} />
    </IconButton>
  )
}

PlayButton.propTypes = {
  record: PropTypes.object,
  color: PropTypes.string,
  size: PropTypes.string,
}

PlayButton.defaultProps = {
  size: 'small',
}
