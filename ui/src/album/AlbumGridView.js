import React from 'react'
import { GridList, GridListTile, GridListTileBar } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import withWidth from '@material-ui/core/withWidth'
import { Link } from 'react-router-dom'
import { linkToRecord, Loading } from 'react-admin'
import subsonic from '../subsonic'
import { ArtistLinkField } from './ArtistLinkField'
import AlbumContextMenu from './AlbumContextMenu.js'

const useStyles = makeStyles((theme) => ({
  root: {
    margin: '20px',
  },
  gridListTile: {
    minHeight: '180px',
    minWidth: '180px',
  },
  cover: {
    display: 'inline-block',
    width: '100%',
    height: '100%',
  },
  tileBar: {
    textAlign: 'left',
    background:
      'linear-gradient(to top, rgba(0,0,0,1) 0%,rgba(0,0,0,0.4) 70%,rgba(0,0,0,0) 100%)',
  },
  albumArtistName: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    textAlign: 'left',
    fontSize: '1em',
  },
  artistLink: {
    color: theme.palette.primary.light,
  },
}))

const getColsForWidth = (width) => {
  if (width === 'xs') return 2
  if (width === 'sm') return 4
  if (width === 'md') return 5
  if (width === 'lg') return 6
  return 7
}

const LoadedAlbumGrid = ({ ids, data, basePath, width }) => {
  const classes = useStyles()

  return (
    <div className={classes.root}>
      <GridList cellHeight={'auto'} cols={getColsForWidth(width)} spacing={20}>
        {ids.map((id) => (
          <GridListTile
            className={classes.gridListTile}
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
                <div className={classes.albumArtistName}>
                  <ArtistLinkField
                    record={data[id]}
                    className={classes.artistLink}
                  >
                    {data[id].albumArtist}
                  </ArtistLinkField>
                </div>
              }
              actionIcon={<AlbumContextMenu id={id} />}
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
