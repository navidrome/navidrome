import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import {
  GridList,
  GridListTile,
  GridListTileBar,
  Tabs,
  Tab
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import withWidth from '@material-ui/core/withWidth'
import { Link } from 'react-router-dom'
import { linkToRecord } from 'ra-core'
import { Loading } from 'react-admin'
import AllInclusiveIcon from '@material-ui/icons/AllInclusive'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import StarIcon from '@material-ui/icons/Star'
import LibraryAddIcon from '@material-ui/icons/LibraryAdd'
import VideoLibraryIcon from '@material-ui/icons/VideoLibrary'
import subsonic from '../subsonic'
import {
  ALBUM_LIST_ALL,
  ALBUM_LIST_NEWEST,
  ALBUM_LIST_RANDOM,
  ALBUM_LIST_RECENT,
  ALBUM_LIST_STARRED,
  selectAlbumList
} from './albumState'

const useStyles = makeStyles((theme) => ({
  root: {
    margin: '5px'
  },
  cover: {
    display: 'inline-block',
    width: '100%',
    height: '100%'
  },
  tileBar: {
    textAlign: 'center',
    background:
      'linear-gradient(to top, rgba(0,0,0,0.8) 0%,rgba(0,0,0,0.4) 70%,rgba(0,0,0,0) 100%)'
  },
  albumArtistName: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    textAlign: 'center',
    fontSize: '1em'
  }
}))

const getColsForWidth = (width) => {
  if (width === 'xs') return 2
  if (width === 'sm') return 3
  if (width === 'md') return 5
  if (width === 'lg') return 6
  return 7
}

const tabOrder = [
  ALBUM_LIST_ALL,
  ALBUM_LIST_RANDOM,
  ALBUM_LIST_NEWEST,
  ALBUM_LIST_RECENT,
  ALBUM_LIST_STARRED
]

const LoadedAlbumGrid = ({ ids, data, basePath, width }) => {
  const classes = useStyles()
  const dispatch = useDispatch()
  const albumView = useSelector((state) => state.albumView)
  const tabSelected = tabOrder.indexOf(albumView.list)

  const handleChange = (event, newValue) => {
    dispatch(selectAlbumList(tabOrder[newValue]))
  }

  return (
    <div className={classes.root}>
      <Tabs
        value={tabSelected}
        indicatorColor="primary"
        textColor="primary"
        aria-label="disabled tabs example"
        onChange={handleChange}
      >
        <Tab label="All" icon={<AllInclusiveIcon />} />
        <Tab label="Random" icon={<ShuffleIcon />} />
        <Tab label="Newest" icon={<LibraryAddIcon />} />
        <Tab label="Recently Played" icon={<VideoLibraryIcon />} />
        <Tab label="Starred" icon={<StarIcon />} disabled={true} />
      </Tabs>
      <GridList
        cellHeight={'auto'}
        cols={getColsForWidth(width)}
        className={classes.gridList}
        spacing={20}
      >
        {ids.map((id) => (
          <GridListTile
            component={Link}
            key={id}
            to={linkToRecord(basePath, data[id].id, 'show')}
          >
            <img
              src={subsonic.url(
                'getCoverArt',
                data[id].coverArtId || 'not_found',
                { size: 300 }
              )}
              alt={data[id].album}
              className={classes.cover}
            />
            <GridListTileBar
              className={classes.tileBar}
              title={data[id].name}
              subtitle={
                <>
                  <div className={classes.albumArtistName}>
                    {data[id].albumArtist}
                  </div>
                </>
              }
            />
          </GridListTile>
        ))}
      </GridList>
    </div>
  )
}

const AlbumGridView = ({ loading, ...props }) =>
  loading ? <Loading /> : <LoadedAlbumGrid {...props} />

export default withWidth()(AlbumGridView)
