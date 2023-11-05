import React, { isValidElement, useMemo, useCallback, forwardRef } from 'react'
import { useDispatch } from 'react-redux'
import {
  Datagrid,
  PureDatagridBody,
  PureDatagridRow,
  useTranslate,
} from 'react-admin'
import {
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
import { playTracks } from '../actions'
import { AlbumContextMenu } from '../common'
import { DraggableTypes } from '../consts'
import { formatFullDate } from '../utils'

const useStyles = makeStyles({
  subtitle: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    verticalAlign: 'middle',
  },
  discIcon: {
    verticalAlign: 'text-top',
    marginRight: '4px',
  },
  row: {
    cursor: 'pointer',
    '&:hover': {
      '& $contextMenu': {
        visibility: 'visible',
      },
    },
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

const ReleaseRow = forwardRef(
  ({ record, onClick, colSpan, contextAlwaysVisible }, ref) => {
    const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
    const classes = useStyles({ isDesktop })
    const translate = useTranslate()
    const handlePlaySubset = (releaseDate, mbzAlbumId) => () => {
      onClick(releaseDate, mbzAlbumId)
    }

    let releaseTitle = []

    if (record.releaseDate) {
      releaseTitle.push(
        translate('resources.album.fields.released') +
          ' ' +
          formatFullDate(record.releaseDate)
      )
    }
    if (record.catalogNum && isDesktop) {
      releaseTitle.push('Cat # ' + record.catalogNum)
    }

    return (
      <TableRow
        hover
        ref={ref}
        onClick={handlePlaySubset(record.releaseDate, record.mbzAlbumId)}
        className={classes.row}
      >
        <TableCell colSpan={colSpan}>
          <Typography variant="h6" className={classes.subtitle}>
            {releaseTitle.join(' · ')}
          </Typography>
        </TableCell>
        <TableCell>
          <AlbumContextMenu
            record={{ id: record.albumId }}
            releaseDate={record.releaseDate}
            mbzAlbumId={record.mbzAlbumId}
            showLove={false}
            className={classes.contextMenu}
            visible={contextAlwaysVisible}
          />
        </TableCell>
      </TableRow>
    )
  }
)

const DiscSubtitleRow = forwardRef(
  ({ record, onClick, colSpan, contextAlwaysVisible }, ref) => {
    const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
    const classes = useStyles({ isDesktop })
    const handlePlaySubset = (releaseDate, mbzAlbumId, discNumber) => () => {
      onClick(releaseDate, mbzAlbumId, discNumber)
    }

    let subtitle = []
    if (record.discNumber > 0) {
      subtitle.push(record.discNumber)
    }
    if (record.discSubtitle) {
      subtitle.push(record.discSubtitle)
    }

    return (
      <TableRow
        hover
        ref={ref}
        onClick={handlePlaySubset(
          record.releaseDate,
          record.mbzAlbumId,
          record.discNumber
        )}
        className={classes.row}
      >
        <TableCell colSpan={colSpan}>
          <Typography variant="h6" className={classes.subtitle}>
            <AlbumIcon className={classes.discIcon} fontSize={'small'} />
            {subtitle.join(': ')}
          </Typography>
        </TableCell>
        <TableCell>
          <AlbumContextMenu
            record={{ id: record.albumId }}
            discNumber={record.discNumber}
            releaseDate={record.releaseDate}
            mbzAlbumId={record.mbzAlbumId}
            showLove={false}
            className={classes.contextMenu}
            visible={contextAlwaysVisible}
          />
        </TableCell>
      </TableRow>
    )
  }
)

export const SongDatagridRow = ({
  record,
  children,
  firstTracksOfDiscs,
  firstTracksOfReleases,
  contextAlwaysVisible,
  onClickSubset,
  className,
  ...rest
}) => {
  const classes = useStyles()
  const fields = React.Children.toArray(children).filter((c) =>
    isValidElement(c)
  )

  const [, dragDiscRef] = useDrag(
    () => ({
      type: DraggableTypes.DISC,
      item: {
        discs: [
          {
            albumId: record?.albumId,
            releaseDate: record?.releaseDate,
            mbzAlbumId: record?.mbzAlbumId,
            discNumber: record?.discNumber,
          },
        ],
      },
      options: { dropEffect: 'copy' },
    }),
    [record]
  )

  const [, dragSongRef] = useDrag(
    () => ({
      type: DraggableTypes.SONG,
      item: { ids: [record?.mediaFileId || record?.id] },
      options: { dropEffect: 'copy' },
    }),
    [record]
  )

  if (!record || !record.title) {
    return null
  }

  const childCount = fields.length
  return (
    <>
      {firstTracksOfReleases.has(record.id) && (
        <ReleaseRow
          ref={dragDiscRef}
          record={record}
          onClick={onClickSubset}
          contextAlwaysVisible={contextAlwaysVisible}
          colSpan={childCount + (rest.expand ? 1 : 0)}
        />
      )}
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
        className={clsx(className, classes.row)}
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
  firstTracksOfReleases: PropTypes.instanceOf(Set),
  contextAlwaysVisible: PropTypes.bool,
  onClickSubset: PropTypes.func,
}

SongDatagridRow.defaultProps = {
  onClickSubset: () => {},
}

const SongDatagridBody = ({
  contextAlwaysVisible,
  showDiscSubtitles,
  showReleaseDivider,
  ...rest
}) => {
  const dispatch = useDispatch()
  const { ids, data } = rest

  const playSubset = useCallback(
    (releaseDate, mbzAlbumId, discNumber) => {
      let idsToPlay = []
      if (discNumber !== undefined) {
        idsToPlay = ids.filter(
          (id) =>
            data[id].releaseDate === releaseDate &&
            data[id].mbzAlbumId === mbzAlbumId &&
            data[id].discNumber === discNumber
        )
      } else {
        idsToPlay = ids.filter(
          (id) =>
            (releaseDate && data[id].releaseDate === releaseDate) ||
            (mbzAlbumId && data[id].mbzAlbumId === mbzAlbumId)
        )
      }
      dispatch(playTracks(data, idsToPlay))
    },
    [dispatch, data, ids]
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
            (last && data[id].discNumber !== data[last].discNumber) ||
            (last && data[id].mbzAlbumId !== data[last].mbzAlbumId) ||
            (last && data[id].releaseDate !== data[last].releaseDate)
          ) {
            acc.push(id)
          }
          return acc
        }, [])
    )
    if (!showDiscSubtitles || (set.size < 2 && !foundSubtitle)) {
      set.clear()
    }
    return set
  }, [ids, data, showDiscSubtitles])

  const firstTracksOfReleases = useMemo(() => {
    if (!ids) {
      return new Set()
    }
    const set = new Set(
      ids
        .filter((i) => data[i])
        .reduce((acc, id) => {
          const last = acc && acc[acc.length - 1]
          if (
            acc.length === 0 ||
            (last && data[id].releaseDate !== data[last].releaseDate) ||
            (last && data[id].mbzAlbumId !== data[last].mbzAlbumId)
          ) {
            acc.push(id)
          }
          return acc
        }, [])
    )
    if (!showReleaseDivider || set.size < 2) {
      set.clear()
    }
    return set
  }, [ids, data, showReleaseDivider])

  return (
    <PureDatagridBody
      {...rest}
      row={
        <SongDatagridRow
          firstTracksOfDiscs={firstTracksOfDiscs}
          firstTracksOfReleases={firstTracksOfReleases}
          contextAlwaysVisible={contextAlwaysVisible}
          onClickSubset={playSubset}
        />
      }
    />
  )
}

export const SongDatagrid = ({
  contextAlwaysVisible,
  showDiscSubtitles,
  showReleaseDivider,
  ...rest
}) => {
  const classes = useStyles()
  return (
    <Datagrid
      className={classes.headerStyle}
      {...rest}
      body={
        <SongDatagridBody
          contextAlwaysVisible={contextAlwaysVisible}
          showDiscSubtitles={showDiscSubtitles}
          showReleaseDivider={showReleaseDivider}
        />
      }
    />
  )
}

SongDatagrid.propTypes = {
  contextAlwaysVisible: PropTypes.bool,
  showDiscSubtitles: PropTypes.bool,
  showReleaseDivider: PropTypes.bool,
  classes: PropTypes.object,
}
