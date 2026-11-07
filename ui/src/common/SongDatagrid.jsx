import React, {
  isValidElement,
  cloneElement,
  useMemo,
  useCallback,
  useState,
  useRef,
  useEffect,
  useLayoutEffect,
  forwardRef,
} from 'react'
import { useDispatch } from 'react-redux'
import {
  Datagrid,
  PureDatagridBody,
  PureDatagridRow,
  useTranslate,
} from 'react-admin'
import {
  TableBody,
  TableCell,
  TableRow,
  Typography,
  useMediaQuery,
} from '@material-ui/core'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core/styles'
import AlbumIcon from '@material-ui/icons/Album'
import clsx from 'clsx'
import { useDrag } from 'react-dnd'
import Lightbox from 'react-image-lightbox'
import 'react-image-lightbox/style.css'
import { playTracks } from '../actions'
import subsonic from '../subsonic'
import { AlbumContextMenu } from '../common'
import { DraggableTypes } from '../consts'
import { formatFullDate } from '../utils'

const useStyles = makeStyles({
  subtitle: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    verticalAlign: 'middle',
    display: 'flex',
    alignItems: 'center',
  },
  discIcon: {
    marginRight: '14px',
  },
  discCoverArt: {
    width: '48px',
    height: '48px',
    marginRight: '14px',
    objectFit: 'cover',
    borderRadius: '4px',
    flexShrink: 0,
    cursor: 'pointer',
  },
  row: {
    cursor: 'pointer',
    '&:hover': {
      '& $contextMenu': {
        visibility: 'visible',
      },
    },
  },
  missingRow: {
    cursor: 'inherit',
    opacity: 0.3,
  },
  headerStyle: {
    '& thead': {
      boxShadow: '0px 3px 3px rgba(0, 0, 0, 0.15)',
    },
    '& th': {
      fontWeight: 'bold',
      padding: '15px',
    },
  },
  contextMenu: {
    visibility: (props) => (props.isDesktop ? 'hidden' : 'visible'),
  },
})

export const DiscSubtitleRow = forwardRef(
  ({ record, onClick, colSpan, contextAlwaysVisible }, ref) => {
    const translate = useTranslate()
    const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
    const classes = useStyles({ isDesktop })
    const [imageError, setImageError] = useState(false)
    const [isLightboxOpen, setLightboxOpen] = useState(false)
    const lightboxClosedAt = React.useRef(0)
    const handlePlaySubset = (discNumber) => () => {
      // Ignore clicks shortly after the lightbox was closed to prevent
      // mobile touch events from "falling through" the overlay and
      // triggering playback.
      if (Date.now() - lightboxClosedAt.current < 400) {
        return
      }
      onClick(discNumber)
    }

    const coverArtUrl = subsonic.getDiscCoverArtUrl(
      record.albumId,
      record.discNumber,
      record.updatedAt,
      96,
    )

    const fullImageUrl = subsonic.getDiscCoverArtUrl(
      record.albumId,
      record.discNumber,
      record.updatedAt,
    )

    const handleOpenLightbox = useCallback(
      (e) => {
        if (!imageError) {
          e.stopPropagation()
          setLightboxOpen(true)
        }
      },
      [imageError],
    )

    const handleCloseLightbox = useCallback(() => {
      lightboxClosedAt.current = Date.now()
      setLightboxOpen(false)
    }, [])

    const subtitle = record.discSubtitle
      ? record.discSubtitle
      : translate('resources.song.fields.disc', {
          discNumber: record.discNumber,
        })

    return (
      <TableRow
        hover
        ref={ref}
        onClick={handlePlaySubset(record.discNumber)}
        className={classes.row}
      >
        <TableCell colSpan={colSpan}>
          <Typography variant="h6" className={classes.subtitle}>
            {!imageError ? (
              <img
                src={coverArtUrl}
                className={classes.discCoverArt}
                alt=""
                onClick={handleOpenLightbox}
                onError={() => setImageError(true)}
              />
            ) : (
              <AlbumIcon className={classes.discIcon} fontSize={'small'} />
            )}
            {subtitle}
          </Typography>
          {isLightboxOpen && !imageError && (
            // Lightbox portals out of the row, but React still bubbles its
            // events up this tree, where the row's onClick would play the disc.
            <span onClick={(e) => e.stopPropagation()}>
              <Lightbox
                imagePadding={50}
                animationDuration={200}
                imageTitle={record.album + ' - ' + subtitle}
                mainSrc={fullImageUrl}
                onCloseRequest={handleCloseLightbox}
              />
            </span>
          )}
        </TableCell>
        <TableCell>
          <AlbumContextMenu
            record={{ id: record.albumId }}
            discNumber={record.discNumber}
            showLove={false}
            className={classes.contextMenu}
            hideShare={true}
            hideInfo={true}
            visible={contextAlwaysVisible}
          />
        </TableCell>
      </TableRow>
    )
  },
)

DiscSubtitleRow.displayName = 'DiscSubtitleRow'

export const SongDatagridRow = ({
  record,
  children,
  firstTracksOfDiscs,
  contextAlwaysVisible,
  onClickSubset,
  className,
  ...rest
}) => {
  const classes = useStyles()
  const fields = React.Children.toArray(children).filter((c) =>
    isValidElement(c),
  )

  const [, dragDiscRef] = useDrag(
    () => ({
      type: DraggableTypes.DISC,
      item: {
        discs: [
          {
            albumId: record?.albumId,
            discNumber: record?.discNumber,
          },
        ],
      },
      options: { dropEffect: 'copy' },
    }),
    [record],
  )

  const [, dragSongRef] = useDrag(
    () => ({
      type: DraggableTypes.SONG,
      item: { ids: [record?.mediaFileId || record?.id] },
      options: { dropEffect: 'copy' },
    }),
    [record],
  )

  if (!record || !record.title) {
    return null
  }

  const rowClick = record.missing ? undefined : rest.rowClick

  const computedClasses = clsx(
    className,
    classes.row,
    record.missing && classes.missingRow,
  )
  const childCount = fields.length
  return (
    <>
      {firstTracksOfDiscs.has(record.id) && (
        <DiscSubtitleRow
          ref={dragDiscRef}
          record={record}
          onClick={onClickSubset}
          contextAlwaysVisible={contextAlwaysVisible}
          colSpan={childCount + (rest.expand ? 1 : 0)}
        />
      )}
      <PureDatagridRow
        ref={dragSongRef}
        record={record}
        {...rest}
        rowClick={rowClick}
        className={computedClasses}
      >
        {fields}
      </PureDatagridRow>
    </>
  )
}

SongDatagridRow.propTypes = {
  record: PropTypes.object,
  children: PropTypes.node,
  firstTracksOfDiscs: PropTypes.instanceOf(Set),
  contextAlwaysVisible: PropTypes.bool,
  onClickSubset: PropTypes.func,
}

SongDatagridRow.defaultProps = {
  onClickSubset: () => {},
}

// Nearest scrollable ancestor, or window. The album view historically scrolls
// the page (react-admin layout), but a themed/embedded layout could introduce
// an overflow container, so detect rather than assume window.
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

// Rough starting guess; corrected at runtime by measuring rendered rows. Only
// affects how many rows are drawn per frame, never correctness.
const ESTIMATED_ROW_HEIGHT = 52
const OVERSCAN = 10

// Windowed TableBody: renders only the visible slice of rows, padding the rest
// with two spacer <tr>s so the scrollbar/height stay correct. This is what
// stops 300+ track albums from freezing the tab: react-admin's default body
// mounts every row (each with 2 useDrag hooks + ~8 field components), which is
// O(n) DOM + JS on open. See navidrome issue #1186 (non-virtualized Datagrid).
//
// Deliberately reuses `row` (the <SongDatagridRow/>) and clones it with the
// exact same per-row props react-admin's PureDatagridBody would, so row
// internals (selection, disc subtitles, drag, context menu) are unchanged.
const VirtualizedDatagridBody = ({
  row,
  children,
  ids,
  data,
  basePath,
  resource,
  hasBulkActions,
  onToggleItem,
  rowClick,
  selectedIds,
  isRowSelectable,
  expand,
  hover,
  classes,
  className,
  rowStyle,
}) => {
  const tbodyRef = useRef(null)
  const rowHeightRef = useRef(ESTIMATED_ROW_HEIGHT)
  const [range, setRange] = useState({ start: 0, end: OVERSCAN * 3 })

  const total = ids ? ids.length : 0

  useLayoutEffect(() => {
    const scroller = getScrollParent(tbodyRef.current)
    const recompute = () => {
      const el = tbodyRef.current
      if (!el) {
        return
      }
      const rowH = rowHeightRef.current || ESTIMATED_ROW_HEIGHT
      const rect = el.getBoundingClientRect()
      // How far the table's top has scrolled above the viewport top, and how
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
      const start = Math.max(0, Math.floor(above / rowH) - OVERSCAN)
      const end = Math.min(
        total,
        Math.ceil((above + viewH) / rowH) + OVERSCAN,
      )
      setRange((prev) =>
        prev.start === start && prev.end === end ? prev : { start, end },
      )
    }
    recompute()
    const target = scroller === window ? window : scroller
    target.addEventListener('scroll', recompute, { passive: true })
    window.addEventListener('resize', recompute)
    return () => {
      target.removeEventListener('scroll', recompute)
      window.removeEventListener('resize', recompute)
    }
  }, [total])

  // Correct the row-height estimate from what actually rendered, then let the
  // next scroll/paint pick up the better value. Guards against the estimate
  // being so far off that the visible window leaves blank gaps.
  useEffect(() => {
    const el = tbodyRef.current
    const rendered = range.end - range.start
    if (el && rendered > 0) {
      // Take the spacer rows' height off, leaving only the rendered rows.
      const measured = el.getBoundingClientRect().height
      const spacerH =
        range.start * rowHeightRef.current +
        (total - range.end) * rowHeightRef.current
      const bodyH = measured - spacerH
      if (bodyH > 0) {
        const avg = bodyH / rendered
        if (avg > 0 && Math.abs(avg - rowHeightRef.current) > 4) {
          rowHeightRef.current = avg
        }
      }
    }
  }, [range, total])

  const colSpan =
    React.Children.count(children) + (hasBulkActions ? 1 : 0) + (expand ? 1 : 0)

  const topH = range.start * rowHeightRef.current
  const bottomH = (total - range.end) * rowHeightRef.current

  const visible = ids ? ids.slice(range.start, range.end) : []

  return (
    <TableBody ref={tbodyRef} className={className}>
      {topH > 0 && (
        <TableRow style={{ height: topH }}>
          <TableCell
            colSpan={colSpan}
            style={{ height: topH, padding: 0, border: 0 }}
          />
        </TableRow>
      )}
      {visible.map((id, i) =>
        cloneElement(
          row,
          {
            basePath,
            classes,
            expand,
            hasBulkActions,
            hover,
            id,
            key: id,
            onToggleItem,
            record: data[id],
            resource,
            rowClick,
            selectable: !isRowSelectable || isRowSelectable(data[id]),
            selected: selectedIds?.includes(id),
            style: rowStyle ? rowStyle(data[id], range.start + i) : null,
          },
          children,
        ),
      )}
      {bottomH > 0 && (
        <TableRow style={{ height: bottomH }}>
          <TableCell
            colSpan={colSpan}
            style={{ height: bottomH, padding: 0, border: 0 }}
          />
        </TableRow>
      )}
    </TableBody>
  )
}

const SongDatagridBody = ({
  contextAlwaysVisible,
  showDiscSubtitles,
  virtualize,
  ...rest
}) => {
  const dispatch = useDispatch()
  const { ids, data } = rest

  const playSubset = useCallback(
    (discNumber) => {
      let idsToPlay = []
      if (discNumber !== undefined) {
        idsToPlay = ids.filter((id) => data[id].discNumber === discNumber)
      }
      dispatch(
        playTracks(
          data,
          idsToPlay?.filter((id) => !data[id].missing),
        ),
      )
    },
    [dispatch, data, ids],
  )

  const firstTracksOfDiscs = useMemo(() => {
    if (!ids) {
      return new Set()
    }
    let foundSubtitle = false
    const set = new Set(
      ids
        .filter((i) => data[i])
        .reduce((acc, id) => {
          const last = acc && acc[acc.length - 1]
          foundSubtitle = foundSubtitle || data[id].discSubtitle
          if (
            acc.length === 0 ||
            (last && data[id].discNumber !== data[last].discNumber)
          ) {
            acc.push(id)
          }
          return acc
        }, []),
    )
    if (!showDiscSubtitles || (set.size < 2 && !foundSubtitle)) {
      set.clear()
    }
    return set
  }, [ids, data, showDiscSubtitles])

  const row = (
    <SongDatagridRow
      firstTracksOfDiscs={firstTracksOfDiscs}
      contextAlwaysVisible={contextAlwaysVisible}
      onClickSubset={playSubset}
    />
  )

  // Virtualization is opt-in (album view only). SongList is paginated and
  // PlaylistSongs wraps this in <ReorderableList nodeSelector="tr"> which needs
  // every row present in the DOM to drag-reorder, so both keep the default
  // body that renders all rows.
  if (virtualize) {
    return <VirtualizedDatagridBody {...rest} row={row} />
  }

  return <PureDatagridBody {...rest} row={row} />
}

export const SongDatagrid = ({
  contextAlwaysVisible,
  showDiscSubtitles,
  virtualize,
  ...rest
}) => {
  const classes = useStyles()
  return (
    <Datagrid
      className={classes.headerStyle}
      isRowSelectable={(r) => !r?.missing}
      {...rest}
      body={
        <SongDatagridBody
          contextAlwaysVisible={contextAlwaysVisible}
          showDiscSubtitles={showDiscSubtitles}
          virtualize={virtualize}
        />
      }
    />
  )
}

SongDatagrid.propTypes = {
  contextAlwaysVisible: PropTypes.bool,
  showDiscSubtitles: PropTypes.bool,
  virtualize: PropTypes.bool,
  classes: PropTypes.object,
}
