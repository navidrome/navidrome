import React from 'react'
import {
  GridList,
  GridListTile,
  Typography,
  GridListTileBar,
  useMediaQuery,
} from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import withWidth from '@material-ui/core/withWidth'
import { Link } from 'react-router-dom'
import { linkToRecord, Loading } from 'react-admin'
import { withContentRect } from 'react-measure'
import subsonic from '../subsonic'
import {
  FolderContextMenu,
  PlayButton,
  OverflowTooltip,
  useImageUrl,
} from '../common'
import config from '../config'
import clsx from 'clsx'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      margin: '20px',
      display: 'grid',
    },
    tileBar: {
      transition: 'all 150ms ease-out',
      opacity: 0,
      pointerEvents: 'none',
      textAlign: 'left',
      background:
        'linear-gradient(to top, rgba(0,0,0,0.7) 0%,rgba(0,0,0,0.4) 70%,rgba(0,0,0,0) 100%)',
    },
    tileBarMobile: {
      textAlign: 'left',
      background:
        'linear-gradient(to top, rgba(0,0,0,0.7) 0%,rgba(0,0,0,0.4) 70%,rgba(0,0,0,0) 100%)',
    },
    folderName: {
      fontSize: '14px',
      color: theme.palette.type === 'dark' ? '#eee' : 'black',
      overflow: 'hidden',
      whiteSpace: 'nowrap',
      textOverflow: 'ellipsis',
      marginTop: '8px',
    },
    missingFolder: {
      opacity: 0.3,
    },
    link: {
      position: 'relative',
      display: 'block',
      textDecoration: 'none',
      '&:hover $tileBar, &:focus-within $tileBar': {
        opacity: 1,
        pointerEvents: 'auto',
      },
    },
    folderLink: {
      position: 'relative',
      display: 'block',
      textDecoration: 'none',
    },
    folderContainer: {},
    folderPlayButton: { color: 'white' },
  }),
  { name: 'NDFolderGridView' },
)

const useCoverStyles = makeStyles({
  coverContainer: {
    width: '100%',
    aspectRatio: '1',
    overflow: 'hidden',
    position: 'relative',
  },
  cover: {
    display: 'inline-block',
    width: '100%',
    objectFit: 'contain',
    height: (props) => props.height,
    transition: 'opacity 0.3s ease-in-out',
  },
  coverLoading: {
    opacity: 0,
  },
})

const getColsForWidth = (width) => {
  if (width === 'xs') return 2
  if (width === 'sm') return 3
  if (width === 'md') return 4
  if (width === 'lg') return 6
  return 9
}

const Cover = withContentRect('bounds')(({
  record,
  measureRef,
  contentRect,
}) => {
  const classes = useCoverStyles({ height: contentRect.bounds.width })
  const url = subsonic.getCoverArtUrl(record, config.uiCoverArtSize, true)
  const { imgUrl, loading: imageLoading } = useImageUrl(url)

  return (
    <div ref={measureRef} className={classes.coverContainer}>
        <img
            src={imgUrl || undefined}
            alt={record.name}
            className={`${classes.cover} ${imageLoading ? classes.coverLoading : ''}`}
        />
    </div>
  )
})

const FolderGridTile = ({ record, basePath }) => {
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'), {
    noSsr: true,
  })
  if (!record) {
    return null
  }
  const computedClasses = clsx(
    classes.folderContainer,
    record.missing && classes.missingFolder,
  )
  return (
    <div className={computedClasses}>
      <Link
        className={classes.link}
        to={linkToRecord(basePath, record.id, 'show')}
      >
        <Cover record={record} />
        <GridListTileBar
          className={isDesktop ? classes.tileBar : classes.tileBarMobile}
          title={
            !record.missing && (
              <PlayButton
                className={classes.folderPlayButton}
                record={record}
                size="small"
                resource="folder"
              />
            )
          }
          subtitle={
            <Typography variant="caption" style={{ color: 'white' }}>
              {record.totalSongs} {record.totalSongs === 1 ? 'Song' : 'Songs'}
            </Typography>
          }
          actionIcon={<FolderContextMenu record={record} color={'white'} source="name" />}
        />
      </Link>
      <Link
        className={classes.folderLink}
        to={linkToRecord(basePath, record.id, 'show')}
      >
        <span>
          <OverflowTooltip title={record.name}>
            <Typography className={classes.folderName}>{record.name}</Typography>
          </OverflowTooltip>
        </span>
      </Link>
    </div>
  )
}

const LoadedFolderGrid = ({ ids, data, basePath, width }) => {
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
          <GridListTile key={id}>
            <FolderGridTile
              record={data[id]}
              basePath={basePath}
            />
          </GridListTile>
        ))}
      </GridList>
    </div>
  )
}

const FolderGridView = ({ loaded, loading, ...props }) => {
  const hide = loading || !props.data || !props.ids
  return hide ? <Loading /> : <LoadedFolderGrid {...props} />
}

const FolderGridViewWithWidth = withWidth()(FolderGridView)

export default FolderGridViewWithWidth
