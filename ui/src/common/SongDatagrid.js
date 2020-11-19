import React, {
  useState,
  isValidElement,
  cloneElement,
  useMemo,
  useCallback,
} from 'react'
import { useDispatch } from 'react-redux'
import { Datagrid, DatagridBody, DatagridRow } from 'react-admin'
import { TableCell, TableRow, Typography } from '@material-ui/core'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core/styles'
import AlbumIcon from '@material-ui/icons/Album'
import { playTracks } from '../actions'
import { AlbumContextMenu } from '../common'

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
  onClick,
  colSpan,
  contextAlwaysVisible,
}) => {
  const classes = useStyles()
  const [visible, setVisible] = useState(false)
  const handlePlayDisc = (discNumber) => () => {
    onClick(discNumber)
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
          showStar={false}
          visible={contextAlwaysVisible || visible}
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
  ...rest
}) => {
  const [visible, setVisible] = useState(false)
  const fields = React.Children.toArray(children).filter((c) =>
    isValidElement(c)
  )
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
      <DatagridRow
        record={record}
        onMouseMove={() => setVisible(true)}
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
  firstTracks: PropTypes.instanceOf(Set),
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

  const playDisc = useCallback(
    (discNumber) => {
      const idsToPlay = ids.filter((id) => data[id].discNumber === discNumber)
      dispatch(playTracks(data, idsToPlay))
    },
    [dispatch, data, ids]
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

  const SongDatagridBody = (props) => (
    <DatagridBody
      {...props}
      row={
        <SongDatagridRow
          firstTracks={firstTracks}
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
