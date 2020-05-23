import React, { useState, isValidElement, cloneElement } from 'react'
import { DatagridRow, useTranslate } from 'react-admin'
import { TableCell, TableRow, Typography } from '@material-ui/core'
import PropTypes from 'prop-types'
import RangeField from './RangeField'

const SongDatagridRow = ({ record, children, multiDisc, ...rest }) => {
  const translate = useTranslate()
  const [visible, setVisible] = useState(false)
  return (
    <>
      {multiDisc && (
        <TableRow>
          {record.trackNumber === 1 && (
            <TableCell colSpan={children.length + 2}>
              <Typography variant="h6">
                {record.discSubtitle
                  ? translate('message.discSubtitle', {
                      subtitle: record.discSubtitle,
                      number: record.discNumber,
                    })
                  : translate('message.discWithoutSubtitle', {
                      number: record.discNumber,
                    })}
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
        {React.Children.map(children, (child) =>
          child &&
          isValidElement(child) &&
          child.type.name === 'SongContextMenu'
            ? cloneElement(child, { visible, ...rest })
            : child
        )}
      </DatagridRow>
    </>
  )
}

RangeField.propTypes = {
  record: PropTypes.object,
  children: PropTypes.node,
  multiDisc: PropTypes.bool,
}

export default SongDatagridRow
