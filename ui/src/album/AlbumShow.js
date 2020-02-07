import React from 'react'
import { Show } from 'react-admin'
import { Title } from '../common'
import { makeStyles } from '@material-ui/core/styles'
import { AlbumSongList } from './AlbumSongList'
import { AlbumDetails } from './AlbumDetails'

const AlbumTitle = ({ record }) => {
  return <Title subTitle={record ? record.name : ''} />
}

const useStyles = makeStyles({
  container: { minWidth: '24em', padding: '1em' },
  rightAlignedCell: { textAlign: 'right' },
  boldCell: { fontWeight: 'bold' },
  albumCover: {
    display: 'inline-block',
    height: '8em',
    width: '8em'
  },
  albumDetails: {
    display: 'inline-block',
    verticalAlign: 'top',
    width: '14em'
  },
  albumTitle: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis'
  }
})

const AlbumShow = (props) => {
  const classes = useStyles()
  return (
    <>
      <AlbumDetails classes={classes} {...props} />
      <Show title={<AlbumTitle />} {...props}>
        <AlbumSongList {...props} />
      </Show>
    </>
  )
}

export default AlbumShow
