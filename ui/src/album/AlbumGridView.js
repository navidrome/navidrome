import React, { useState } from 'react'
import {
  GridList,
  GridListTile,
  GridListTileBar,
  useMediaQuery,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import withWidth from '@material-ui/core/withWidth'
import { Link } from 'react-router-dom'
import { linkToRecord, Loading } from 'react-admin'
import { withContentRect } from 'react-measure'
import subsonic from '../subsonic'
import { ArtistLinkField, RangeField } from '../common'
import { AlbumContextMenu } from '../common'

const useStyles = makeStyles((theme) => ({
  root: {
    margin: '20px',
  },
  tileBar: {
    textAlign: 'left',
    marginBottom: '3px',
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

const useCoverStyles = makeStyles({
  cover: {
    display: 'inline-block',
    width: '100%',
    height: (props) => props.height,
  },
})

const getColsForWidth = (width) => {
  if (width === 'xs') return 2
  if (width === 'sm') return 3
  if (width === 'md') return 4
  if (width === 'lg') return 6
  if (width === 'xl') return 9
  if (width === 'xxl') return 12
  return 18
}

const Cover = withContentRect('bounds')(
  ({ album, measureRef, contentRect }) => {
    // Force height to be the same as the width determined by the GridList
    // noinspection JSSuspiciousNameCombination
    const classes = useCoverStyles({ height: contentRect.bounds.width })
    return (
      <div ref={measureRef}>
        <img
          src={subsonic.url('getCoverArt', album.coverArtId || 'not_found', {
            size: 300,
          })}
          alt={album.album}
          className={classes.cover}
        />
      </div>
    )
  }
)

const AlbumGridTile = ({ showArtist, record, basePath }) => {
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const classes = useStyles()
  const [visible, setVisible] = useState(false)

  return (
    <div
      onMouseMove={() => {
        setVisible(true)
      }}
      onMouseLeave={() => {
        setVisible(false)
      }}
    >
      <Link to={linkToRecord(basePath, record.id, 'show')}>
        <Cover album={record} />
        {(!isDesktop || visible) && (
          <GridListTileBar
            className={classes.tileBar}
            title={record.name}
            subtitle={
              <div className={classes.albumArtistName}>
                {showArtist ? (
                  <ArtistLinkField
                    record={record}
                    className={classes.artistLink}
                  />
                ) : (
                  <RangeField
                    record={record}
                    source={'year'}
                    sortBy={'maxYear'}
                    sortByOrder={'DESC'}
                  />
                )}
              </div>
            }
            actionIcon={<AlbumContextMenu record={record} color={'white'} />}
          />
        )}
      </Link>
    </div>
  )
}

const LoadedAlbumGrid = ({ ids, data, basePath, width, isArtistView }) => {
  const classes = useStyles()

  return (
    <div className={classes.root}>
      <GridList
        component={'div'}
        cellHeight={'auto'}
        cols={getColsForWidth(width)}
        spacing={20}
      >
        {ids.map((id) => (
          <GridListTile className={classes.gridListTile} key={id}>
            <AlbumGridTile
              record={data[id]}
              basePath={basePath}
              showArtist={!isArtistView}
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
