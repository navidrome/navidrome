import React from 'react'
import { Show } from 'react-admin'
import { Title } from '../common'
import { makeStyles } from '@material-ui/core/styles'
import AlbumSongList from './AlbumSongList'
import AlbumDetails from './AlbumDetails'

const AlbumTitle = ({ record }) => {
  return <Title subTitle={record ? record.name : ''} />
}

const useStyles = makeStyles((theme) => ({
  container: {
    [theme.breakpoints.down('xs')]: {
      padding: '0.7em',
      minWidth: '24em'
    },
    [theme.breakpoints.up('sm')]: {
      padding: '1em',
      minWidth: '32em'
    }
  },
  albumCover: {
    display: 'inline-block',
    [theme.breakpoints.down('xs')]: {
      height: '8em',
      width: '8em'
    },
    [theme.breakpoints.up('sm')]: {
      height: '15em',
      width: '15em'
    },
    [theme.breakpoints.up('lg')]: {
      height: '20em',
      width: '20em'
    }
  },
  albumDetails: {
    display: 'inline-block',
    verticalAlign: 'top',
    [theme.breakpoints.down('xs')]: {
      width: '14em'
    },
    [theme.breakpoints.up('sm')]: {
      width: '26em'
    },
    [theme.breakpoints.up('lg')]: {
      width: '38em'
    }
  },
  albumTitle: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis'
  }
}))

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
