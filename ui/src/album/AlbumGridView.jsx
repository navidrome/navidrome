import React, { useLayoutEffect, useRef, useState } from 'react'
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
import { linkToRecord, useListContext, Loading } from 'react-admin'
import { withContentRect } from 'react-measure'
import { useRollChanged } from './useRollChanged'
import { useDrag } from 'react-dnd'
import {
  AlbumContextMenu,
  PlayButton,
  ArtistLinkField,
  OverflowTooltip,
} from '../common'
import { DraggableTypes } from '../consts'
import clsx from 'clsx'
import { AlbumDatesField } from './AlbumDatesField.jsx'
import { Artwork } from '../common/Artwork'

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
    missingAlbum: {
      opacity: 0.3,
    },
    albumVersion: {
      fontSize: '12px',
      color: theme.palette.type === 'dark' ? '#c5c5c5' : '#696969',
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
      '&:hover $tileBar, &:focus-within $tileBar': {
        opacity: 1,
        pointerEvents: 'auto',
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
  { name: 'NDAlbumGridView' },
)

const useCoverStyles = makeStyles({
  coverContainer: {
    width: '100%',
    aspectRatio: '1',
    overflow: 'hidden',
  },
  cover: {
    display: 'inline-block',
    width: '100%',
    objectFit: 'contain',
    // The image fills this box absolutely, so it lends no height: a remount that has not been
    // re-measured yet would collapse the tile and blank the cover for a frame.
    aspectRatio: '1',
    height: (props) => props.height,
    transition: 'opacity 0.3s ease-in-out',
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
  // Force height to be the same as the width determined by the GridList
  // noinspection JSSuspiciousNameCombination
  const classes = useCoverStyles({ height: contentRect.bounds.width })
  const [, dragAlbumRef] = useDrag(
    () => ({
      type: DraggableTypes.ALBUM,
      item: { albumIds: [record.id] },
      options: { dropEffect: 'copy' },
    }),
    [record],
  )

  return (
    <div ref={measureRef} className={classes.coverContainer}>
      <div ref={dragAlbumRef}>
        <Artwork
          record={record}
          square
          className={classes.cover}
          title={record.name}
        />
      </div>
    </div>
  )
})

const AlbumGridTile = ({ showArtist, record, basePath, ...props }) => {
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'), {
    noSsr: true,
  })
  if (!record) {
    return null
  }
  const computedClasses = clsx(
    classes.albumContainer,
    record.missing && classes.missingAlbum,
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
          subtitle={
            !record.missing && (
              <PlayButton
                className={classes.albumPlayButton}
                record={record}
                size="small"
              />
            )
          }
          actionIcon={<AlbumContextMenu record={record} color={'white'} />}
        />
      </Link>
      <Link
        className={classes.albumLink}
        to={linkToRecord(basePath, record.id, 'show')}
      >
        <span>
          <OverflowTooltip title={record.name}>
            <Typography className={classes.albumName}>{record.name}</Typography>
          </OverflowTooltip>
          {record.tags && record.tags['albumversion'] && (
            <Typography className={classes.albumVersion}>
              {record.tags['albumversion']}
            </Typography>
          )}
        </span>
      </Link>
      {showArtist ? (
        <ArtistLinkField record={record} className={classes.albumSubtitle} />
      ) : (
        <AlbumDatesField record={record} className={classes.albumSubtitle} />
      )}
    </div>
  )
}

// Nearest scrollable ancestor, or window. Same detection as the album track
// list's VirtualizedDatagridBody (SongDatagrid.jsx).
const getScrollParent = (node) => {
  let el = node?.parentElement
  while (el) {
    const { overflowY } = window.getComputedStyle(el)
    if (overflowY === 'auto' || overflowY === 'scroll') {
      return el
    }
    el = el.parentElement
  }
  return window
}

// Rows of tiles drawn beyond each edge of the viewport.
const OVERSCAN_ROWS = 2

// Windowed grid: renders only the visible rows of tiles, with padding above
// and below standing in for the rest so the scrollbar stays correct. Lets the
// list run at 256 albums per page: the stock grid mounts every tile, and every
// tile queues a cover fetch behind useImageUrl's 4-at-a-time limit, so the
// covers in view wait for all those above them. Unmounting a tile aborts its
// queued fetch.
const LoadedAlbumGrid = ({ ids, data, basePath, width }) => {
  const classes = useStyles()
  const { filterValues } = useListContext()
  const isArtistView = !!(filterValues && filterValues.artist_id)
  const cols = getColsForWidth(width)
  const rows = Math.ceil(ids.length / cols)
  const rootRef = useRef(null)
  const gridRef = useRef(null)
  // Rough guess, replaced by the measured row pitch after the first render.
  const rowHeightRef = useRef(300)
  const measuredRef = useRef(false)
  const recomputeRef = useRef(null)
  const [range, setRange] = useState({ start: 0, end: 1 })

  useLayoutEffect(() => {
    const scroller = getScrollParent(rootRef.current)
    const recompute = () => {
      const el = rootRef.current
      if (!el) {
        return
      }
      const rowH = rowHeightRef.current
      const rect = el.getBoundingClientRect()
      // How far the grid's top has scrolled above the viewport top, and how
      // tall the viewport is, in the scroller's coordinate space.
      let above
      let viewH
      if (scroller === window) {
        above = -rect.top
        viewH = window.innerHeight
      } else {
        const srect = scroller.getBoundingClientRect()
        above = srect.top - rect.top
        viewH = srect.height
      }
      const start = Math.min(
        rows,
        Math.max(0, Math.floor(above / rowH) - OVERSCAN_ROWS),
      )
      const end = Math.min(
        rows,
        Math.ceil((above + viewH) / rowH) + OVERSCAN_ROWS,
      )
      setRange((prev) =>
        prev.start === start && prev.end === end ? prev : { start, end },
      )
    }
    recomputeRef.current = recompute
    measuredRef.current = false
    recompute()
    const target = scroller === window ? window : scroller
    target.addEventListener('scroll', recompute, { passive: true })
    window.addEventListener('resize', recompute)
    return () => {
      target.removeEventListener('scroll', recompute)
      window.removeEventListener('resize', recompute)
    }
  }, [rows, cols])

  // Row pitch is the column width (square cover) plus one to three text
  // lines, so take it from what rendered. Only the first measurement re-windows
  // at once: rows differ in height (album version line), so later corrections
  // wait for the next scroll rather than risk re-windowing back and forth.
  useLayoutEffect(() => {
    const rendered = range.end - range.start
    if (!gridRef.current || rendered <= 0) {
      return
    }
    const avg = gridRef.current.offsetHeight / rendered
    if (avg > 0 && Math.abs(avg - rowHeightRef.current) > 4) {
      rowHeightRef.current = avg
      if (!measuredRef.current) {
        measuredRef.current = true
        recomputeRef.current()
      }
    }
  }, [range, cols])

  const rowH = rowHeightRef.current
  const start = Math.min(range.start, rows)
  const end = Math.min(range.end, rows)
  return (
    <div
      ref={rootRef}
      className={classes.root}
      style={{
        paddingTop: start * rowH,
        paddingBottom: (rows - end) * rowH,
      }}
    >
      <GridList
        ref={gridRef}
        component={'div'}
        cellHeight={'auto'}
        cols={cols}
        spacing={20}
      >
        {ids.slice(start * cols, end * cols).map((id) => (
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

const AlbumGridView = ({
  albumListType,
  loaded,
  loading,
  seed,
  shownSeed,
  ...props
}) => {
  // ArtistShow renders this grid too, with no roll to track, so own a ref when none is passed.
  const ownSeed = useRef(null)
  // A re-roll replaces every album, so the previous roll must not linger while it loads.
  const rerolling =
    useRollChanged(shownSeed ?? ownSeed, seed, loading) &&
    albumListType === 'random'
  const hide = rerolling || !props.data || !props.ids
  return hide ? <Loading /> : <LoadedAlbumGrid {...props} />
}

const AlbumGridViewWithWidth = withWidth()(AlbumGridView)

export default AlbumGridViewWithWidth
