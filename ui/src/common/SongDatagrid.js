import React, { isValidElement, useMemo, useCallback } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Datagrid, PureDatagridBody, PureDatagridRow } from 'react-admin'
import { TableCell, TableRow, Typography } from '@material-ui/core'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core/styles'
import AlbumIcon from '@material-ui/icons/Album'
import clsx from 'clsx'
import { playTracks } from '../actions'
import { AlbumContextMenu } from '../common'
import { get } from 'lodash'

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
    visibility: 'hidden',
  },
})

const DiscSubtitleRow = ({
  record,
  onClick,
  colSpan,
  contextAlwaysVisible,
}) => {
  const classes = useStyles()
  const handlePlayDisc = (discNumber) => () => {
    onClick(discNumber)
  }
  return (
    <TableRow
      hover
      onClick={handlePlayDisc(record.discNumber)}
      className={classes.row}
    >
      <TableCell colSpan={colSpan}>
        <Typography variant="h6" className={classes.subtitle}>
          <AlbumIcon className={classes.discIcon} fontSize={'small'} />
          {record.discNumber}
          {record.discSubtitle && `: ${record.discSubtitle}`}
        </Typography>
      </TableCell>
      <TableCell>
        <AlbumContextMenu
          record={{ id: record.albumId }}
          discNumber={record.discNumber}
          showLove={false}
          className={classes.contextMenu}
          visible={contextAlwaysVisible}
        />
      </TableCell>
    </TableRow>
  )
}

export const SongDatagridRow = ({
  record,
  children,
  firstTracks,
  contextAlwaysVisible,
  onClickDiscSubtitle,
  className,
  ...rest
}) => {
  const classes = useStyles()
  const fields = React.Children.toArray(children).filter((c) =>
    isValidElement(c)
  )
  if (!record || !record.title) {
    return null
  }
  const childCount = fields.length
  return (
    <>
      {firstTracks.has(record.id) && (
        <DiscSubtitleRow
          record={record}
          onClick={onClickDiscSubtitle}
          contextAlwaysVisible={contextAlwaysVisible}
          colSpan={childCount + (rest.expand ? 1 : 0)}
        />
      )}
      <PureDatagridRow
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
  firstTracks: PropTypes.instanceOf(Set),
  contextAlwaysVisible: PropTypes.bool,
  onClickDiscSubtitle: PropTypes.func,
}

SongDatagridRow.defaultProps = {
  onClickDiscSubtitle: () => {},
}

const SongDatagridBody = ({
  contextAlwaysVisible,
  showDiscSubtitles,
  ...rest
}) => {
  const dispatch = useDispatch()
  const { ids, data } = rest
  const albumOrPlaylistId = useSelector((state) =>
    get(state, 'recentAlbumOrPlaylist.id', '')
  )

  const playDisc = useCallback(
    (discNumber) => {
      const idsToPlay = ids.filter((id) => data[id].discNumber === discNumber)
      dispatch(playTracks(data, idsToPlay, undefined, albumOrPlaylistId))
    },
    [dispatch, data, ids, albumOrPlaylistId]
  )

  const firstTracks = useMemo(() => {
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
            (last && data[id].discNumber !== data[last].discNumber)
          ) {
            acc.push(id)
          }
          return acc
        }, [])
    )
    if (!showDiscSubtitles || set.size < 2) {
      set.clear()
    }
    return set
  }, [ids, data, showDiscSubtitles])

  return (
    <PureDatagridBody
      {...rest}
      row={
        <SongDatagridRow
          firstTracks={firstTracks}
          contextAlwaysVisible={contextAlwaysVisible}
          onClickDiscSubtitle={playDisc}
        />
      }
    />
  )
}

export const SongDatagrid = ({
  contextAlwaysVisible,
  showDiscSubtitles,
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
        />
      }
    />
  )
}

SongDatagrid.propTypes = {
  contextAlwaysVisible: PropTypes.bool,
  showDiscSubtitles: PropTypes.bool,
  classes: PropTypes.object,
}
