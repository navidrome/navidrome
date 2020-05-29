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

const DiscSubtitleRow = ({ record, onClickDiscSubtitle, length }) => {
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
      <TableCell colSpan={length}>
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
          visible={visible}
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
  contextMenu,
  onClickDiscSubtitle,
  ...rest
}) => {
  const [visible, setVisible] = useState(false)
  const childCount = React.Children.count(children)
  return (
    <>
      {multiDisc && record.trackNumber === 1 && (
        <DiscSubtitleRow
          record={record}
          onClickDiscSubtitle={onClickDiscSubtitle}
          length={childCount}
        />
      )}
      <DatagridRow
        record={record}
        onMouseEnter={() => setVisible(true)}
        onMouseLeave={() => setVisible(false)}
        {...rest}
      >
        {React.Children.map(
          children,
          (child, index) =>
            child &&
            isValidElement(child) &&
            (index < childCount - 1
              ? child
              : cloneElement(child, {
                  visible: contextAlwaysVisible || visible,
                  ...rest,
                }))
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

export const SongDatagrid = ({ multiDisc, contextAlwaysVisible, ...rest }) => {
  const dispatch = useDispatch()
  const playDisc = (discNumber) => {
    const ids = rest.ids.filter((id) => rest.data[id].discNumber === discNumber)
    dispatch(playTracks(rest.data, ids))
  }
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
  multiDisc: PropTypes.bool,
}
