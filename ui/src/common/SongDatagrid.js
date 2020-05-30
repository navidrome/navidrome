import React, { useState, isValidElement, cloneElement } from 'react'
import { useDispatch } from 'react-redux'
import { Datagrid, DatagridBody, DatagridRow } from 'react-admin'
import { TableCell, TableRow, Typography } from '@material-ui/core'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core/styles'
import AlbumIcon from '@material-ui/icons/Album'
import { playTracks } from '../audioplayer'
import AlbumContextMenu from './AlbumContextMenu'

const useStyles = makeStyles({
  row: {
    cursor: 'pointer',
  },
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
})

const DiscSubtitleRow = ({
  record,
  onClickDiscSubtitle,
  colSpan,
  contextAlwaysVisible,
}) => {
  const classes = useStyles()
  const [visible, setVisible] = useState(false)
  const handlePlayDisc = (discNumber) => () => {
    onClickDiscSubtitle(discNumber)
  }
  return (
    <TableRow
      hover
      onClick={handlePlayDisc(record.discNumber)}
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
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
          visible={contextAlwaysVisible || visible}
        />
      </TableCell>
    </TableRow>
  )
}

export const SongDatagridRow = ({
  record,
  children,
  multiDisc,
  contextAlwaysVisible,
  onClickDiscSubtitle,
  ...rest
}) => {
  const [visible, setVisible] = useState(false)
  const fields = React.Children.toArray(children).filter((c) =>
    isValidElement(c)
  )
  const childCount = fields.length
  return (
    <>
      {multiDisc && record.trackNumber === 1 && (
        <DiscSubtitleRow
          record={record}
          onClickDiscSubtitle={onClickDiscSubtitle}
          contextAlwaysVisible={contextAlwaysVisible}
          colSpan={childCount + (rest.expand ? 1 : 0)}
        />
      )}
      <DatagridRow
        record={record}
        onMouseEnter={() => setVisible(true)}
        onMouseLeave={() => setVisible(false)}
        {...rest}
      >
        {fields.map((child, index) =>
          index < childCount - 1
            ? child
            : cloneElement(child, {
                visible: contextAlwaysVisible || visible,
              })
        )}
      </DatagridRow>
    </>
  )
}

SongDatagridRow.propTypes = {
  record: PropTypes.object,
  children: PropTypes.node,
  multiDisc: PropTypes.bool,
  contextAlwaysVisible: PropTypes.bool,
  onClickDiscSubtitle: PropTypes.func,
}

SongDatagridRow.defaultProps = {
  onClickDiscSubtitle: () => {},
}

export const SongDatagrid = ({
  contextAlwaysVisible,
  showDiscSubtitles,
  ...rest
}) => {
  const dispatch = useDispatch()
  const { ids, data } = rest

  const playDisc = (discNumber) => {
    const idsToPlay = ids.filter((id) => data[id].discNumber === discNumber)
    dispatch(playTracks(data, idsToPlay))
  }

  const multiDisc =
    showDiscSubtitles &&
    new Set(
      ids
        .map((id) => data[id])
        .filter((r) => r) // remove null records
        .map((r) => r.discNumber)
    ).size > 1

  const SongDatagridBody = (props) => (
    <DatagridBody
      {...props}
      row={
        <SongDatagridRow
          multiDisc={multiDisc}
          contextAlwaysVisible={contextAlwaysVisible}
          onClickDiscSubtitle={playDisc}
        />
      }
    />
  )
  return <Datagrid {...rest} body={<SongDatagridBody />} />
}

SongDatagrid.propTypes = {
  contextAlwaysVisible: PropTypes.bool,
  showDiscSubtitles: PropTypes.bool,
}
