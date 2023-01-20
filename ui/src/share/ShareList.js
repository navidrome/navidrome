import {
  Datagrid,
  FunctionField,
  List,
  NumberField,
  TextField,
} from 'react-admin'
import React from 'react'
import { DateField, QualityInfo } from '../common'
import { shareUrl } from '../utils'
import { Link } from '@material-ui/core'

export const FormatInfo = ({ record, size }) => {
  const r = { suffix: record.format, bitRate: record.maxBitRate }
  // TODO Get DefaultDownsamplingFormat
  r.suffix = r.suffix || (r.bitRate ? 'opus' : 'Original')
  return <QualityInfo record={r} size={size} />
}

const ShareList = (props) => {
  return (
    <List
      {...props}
      sort={{ field: 'createdAt', order: 'DESC' }}
      exporter={false}
    >
      <Datagrid rowClick="edit">
        <FunctionField
          source={'id'}
          render={(r) => (
            <Link
              href={shareUrl(r.id)}
              label="URL"
              target="_blank"
              rel="noopener noreferrer"
            >
              {r.id}
            </Link>
          )}
        />
        <TextField source="username" />
        <TextField source="description" />
        <DateField source="contents" />
        <FormatInfo source="format" />
        <NumberField source="visitCount" />
        <DateField source="expiresAt" showTime />
        <DateField source="lastVisitedAt" showTime sortByOrder={'DESC'} />
      </Datagrid>
    </List>
  )
}

export default ShareList
