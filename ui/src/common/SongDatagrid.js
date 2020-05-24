import React, { useState, isValidElement, cloneElement } from 'react'
import { Datagrid, DatagridBody, DatagridRow } from 'react-admin'
import { TableCell, TableRow, Typography } from '@material-ui/core'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core/styles'
import AlbumIcon from '@material-ui/icons/Album'

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
})

export const SongDatagridRow = ({
  record,
  children,
  multiDisc,
  contextVisible,
  ...rest
}) => {
  const classes = useStyles()
  const [visible, setVisible] = useState(false)
  const childCount = React.Children.count(children)
  return (
    <>
      {multiDisc && (
        <TableRow>
          {record.trackNumber === 1 && (
            <TableCell colSpan={children.length + 2}>
              <Typography variant="h6" className={classes.subtitle}>
                <AlbumIcon className={classes.discIcon} fontSize={'small'} />
                {record.discNumber}
                {record.discSubtitle && `: ${record.discSubtitle}`}
              </Typography>
            </TableCell>
          )}
        </TableRow>
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
                  visible: contextVisible || visible,
                  ...child.props,
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
  contextVisible: PropTypes.bool,
}

export const SongDatagrid = ({ multiDisc, contextVisible, ...rest }) => {
  const SongDatagridBody = (props) => (
    <DatagridBody
      {...props}
      row={
        <SongDatagridRow
          multiDisc={multiDisc}
          contextVisible={contextVisible}
        />
      }
    />
  )
  return <Datagrid {...rest} body={<SongDatagridBody />} />
}

SongDatagrid.propTypes = {
  contextVisible: PropTypes.bool,
  multiDisc: PropTypes.bool,
}
