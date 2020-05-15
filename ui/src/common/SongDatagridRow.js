import React from 'react'
import { DatagridRow, useTranslate } from 'react-admin'
import { TableRow, TableCell, Typography } from '@material-ui/core'
import PropTypes from 'prop-types'
import RangeField from './RangeField'

const SongDatagridRow = ({ record, children, multiDisc, ...rest }) => {
  const translate = useTranslate()
  return (
    <>
      {multiDisc && (
        <TableRow>
          {record.trackNumber === 1 && (
            <TableCell colSpan={children.length + 1}>
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
      <DatagridRow record={record} {...rest}>
        {children}
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
