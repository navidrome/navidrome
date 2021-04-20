import {
  GridList,
  GridListTile,
  GridListTileBar,
  Typography,
  useMediaQuery,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import withWidth from '@material-ui/core/withWidth'
import React from 'react'
import { linkToRecord, Loading, useListContext } from 'react-admin'
import { withContentRect } from 'react-measure'
import { Link } from 'react-router-dom'
import {
  AlbumContextMenu,
  ArtistLinkField,
  PlayButton,
  RangeField,
} from '../common'
import subsonic from '../subsonic'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      margin: '20px',
    },
    tileBar: {
      transition: 'all 150ms ease-out',
      opacity: 0,
      textAlign: 'left',
      marginBottom: '3px',
      background:
        'linear-gradient(to top, rgba(0,0,0,0.7) 0%,rgba(0,0,0,0.4) 70%,rgba(0,0,0,0) 100%)',
    },
    tileBarMobile: {
      textAlign: 'left',
      marginBottom: '3px',
      background:
        'linear-gradient(to top, rgba(0,0,0,0.7) 0%,rgba(0,0,0,0.4) 70%,rgba(0,0,0,0) 100%)',
    },
    albumArtistName: {
      whiteSpace: 'nowrap',
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      textAlign: 'left',
      fontSize: '1em',
    },
    albumName: {
      fontSize: '14px',
      color: theme.palette.type === 'dark' ? '#eee' : 'black',
      overflow: 'hidden',
      whiteSpace: 'nowrap',
      textOverflow: 'ellipsis',
    },
    albumSubtitle: {
      fontSize: '12px',
      color: theme.palette.type === 'dark' ? '#c5c5c5' : '#696969',
      overflow: 'hidden',
      whiteSpace: 'nowrap',
      textOverflow: 'ellipsis',
    },
    link: {
      position: 'relative',
      display: 'block',
      textDecoration: 'none',
      '&:hover $tileBar': {
        opacity: 1,
      },
    },
    albumLink: {
      position: 'relative',
      display: 'block',
      textDecoration: 'none',
    },
    albumContainer: {},
    albumPlayButton: { color: 'white' },
  }),
  { name: 'NDAlbumGridView' }
)

const useCoverStyles = makeStyles({
  cover: {
    display: 'inline-block',
    width: '100%',
    objectFit: 'contain',
    height: (props) => props.height,
  },
})

const getColsForWidth = (width) => {
  if (width === 'xs') return 2
  if (width === 'sm') return 3
  if (width === 'md') return 4
  if (width === 'lg') return 6
  return 9
}

const Cover = withContentRect('bounds')(
  ({ album, measureRef, contentRect }) => {
    // Force height to be the same as the width determined by the GridList
    // noinspection JSSuspiciousNameCombination
    const classes = useCoverStyles({ height: contentRect.bounds.width })
    return (
      <div ref={measureRef}>
        <img
          src={subsonic.getCoverArtUrl(album, 300)}
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

  return (
    <div className={classes.albumContainer}>
      <Link
        className={classes.link}
        to={linkToRecord(basePath, record.id, 'show')}
      >
        <Cover album={record} />
        <GridListTileBar
          className={isDesktop ? classes.tileBar : classes.tileBarMobile}
          subtitle={
            <PlayButton
              className={classes.albumPlayButton}
              record={record}
              size="small"
              buttonType="iconButton"
            />
          }
          actionIcon={<AlbumContextMenu record={record} color={'white'} />}
        />
      </Link>
      <Link
        className={classes.albumLink}
        to={linkToRecord(basePath, record.id, 'show')}
      >
        <Typography className={classes.albumName}>{record.name}</Typography>
      </Link>
      {showArtist ? (
        <ArtistLinkField record={record} className={classes.albumSubtitle} />
      ) : (
        <RangeField
          record={record}
          source={'year'}
          sortBy={'maxYear'}
          sortByOrder={'DESC'}
          className={classes.albumSubtitle}
        />
      )}
    </div>
  )
}

const LoadedAlbumGrid = (props) => {
  const { ids, data, basePath, width } = props
  const classes = useStyles()
  const { filterValues } = useListContext()
  const isArtistView = !!(filterValues && filterValues.artist_id)

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
