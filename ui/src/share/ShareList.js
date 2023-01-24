import {
  Datagrid,
  FunctionField,
  List,
  NumberField,
  SimpleList,
  TextField,
  useTranslate,
} from 'react-admin'
import React from 'react'
import { DateField, QualityInfo } from '../common'
import { shareUrl } from '../utils'
import { Link, useMediaQuery } from '@material-ui/core'
import config from '../config'

export const FormatInfo = ({ record, size }) => {
  const r = { suffix: record.format, bitRate: record.maxBitRate }
  r.suffix =
    r.suffix || (r.bitRate ? config.defaultDownsamplingFormat : 'Original')
  return <QualityInfo record={r} size={size} />
}

const ShareList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const translate = useTranslate()
  return (
    <List
      {...props}
      sort={{ field: 'createdAt', order: 'DESC' }}
      exporter={false}
    >
      {isXsmall ? (
        <SimpleList
          primaryText={(r) => r.description || r.contents || r.id}
          secondaryText={(r) => (
            <>
              {translate('resources.share.fields.expiresAt')}:{' '}
              <DateField record={r} source={'expiresAt'} showTime />
            </>
          )}
        />
      ) : (
        <Datagrid rowClick="edit">
          <FunctionField
            source={'id'}
            render={(r) => (
              <Link
                href={shareUrl(r.id)}
                label="URL"
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => {
                  e.stopPropagation()
                }}
              >
                {r.id}
              </Link>
            )}
          />
          <TextField source="username" />
          <TextField source="description" />
          <TextField source="contents" />
          <FormatInfo source="format" />
          <NumberField source="visitCount" />
          <DateField source="lastVisitedAt" showTime sortByOrder={'DESC'} />
          <DateField source="expiresAt" showTime />
        </Datagrid>
      )}
    </List>
  )
}

export default ShareList
