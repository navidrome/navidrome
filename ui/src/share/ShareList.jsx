import {
  Datagrid,
  FunctionField,
  BooleanField,
  List,
  NumberField,
  SimpleList,
  TextField,
  useNotify,
  useTranslate,
} from 'react-admin'
import React from 'react'
import { IconButton, Link, useMediaQuery } from '@material-ui/core'
import ShareIcon from '@material-ui/icons/Share'
import { DateField, QualityInfo } from '../common'
import { sharePlayerUrl } from '../utils'
import config from '../config'

export const FormatInfo = ({ record, size }) => {
  const r = { suffix: record.format, bitRate: record.maxBitRate }
  r.suffix =
    r.suffix || (r.bitRate ? config.defaultDownsamplingFormat : 'Original')
  return <QualityInfo record={r} size={size} />
}

const ShareList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('lg'))
  const translate = useTranslate()
  const notify = useNotify()

  const handleShare = (r) => (e) => {
    const url = sharePlayerUrl(r?.id)
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard
        .writeText(url)
        .then(() => {
          notify(translate('message.shareSuccess', { url }), {
            type: 'info',
            multiLine: true,
            duration: 0,
          })
        })
        .catch((err) => {
          notify(
            translate('message.shareFailure', { url }) + ': ' + err.message,
            {
              type: 'warning',
              multiLine: true,
              duration: 0,
            },
          )
        })
    } else prompt(translate('message.shareCopyToClipboard'), url)

    e.preventDefault()
    e.stopPropagation()
  }

  return (
    <List
      {...props}
      sort={{ field: 'createdAt', order: 'DESC' }}
      exporter={false}
    >
      {isXsmall ? (
        <SimpleList
          leftIcon={(r) => (
            <IconButton onClick={handleShare(r)}>
              <ShareIcon />
            </IconButton>
          )}
          primaryText={(r) => r.description || r.contents || r.id}
          secondaryText={(r) => (
            <>
              {translate('resources.share.fields.expiresAt')}:{' '}
              <DateField record={r} source={'expiresAt'} />
            </>
          )}
          tertiaryText={(r) =>
            `${translate('resources.share.fields.visitCount')}: ${
              r.visitCount || '0'
            }`
          }
        />
      ) : (
        <Datagrid rowClick="edit">
          <FunctionField
            source={'id'}
            render={(r) => (
              <Link
                href={sharePlayerUrl(r.id)}
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
          {isDesktop && <TextField source="contents" />}
          {isDesktop && <FormatInfo source="format" />}
          {config.enableDownloads && <BooleanField source="downloadable" />}
          <NumberField source="visitCount" />
          {isDesktop && (
            <DateField source="lastVisitedAt" showTime sortByOrder={'DESC'} />
          )}
          <DateField source="expiresAt" showTime />
        </Datagrid>
      )}
    </List>
  )
}

export default ShareList
