import * as React from 'react'
import { LockOpen, Lock } from '@material-ui/icons'
import { BulkUpdateButton, useTranslate } from 'react-admin'

const ChangePublicStatusButton = (props) => {
  const translate = useTranslate()
  const playlists = { public: props?.public }
  const label = props?.public
    ? translate('resources.playlist.actions.makePublic')
    : translate('resources.playlist.actions.makePrivate')
  const icon = props?.public ? <LockOpen /> : <Lock />
  return (
    <BulkUpdateButton {...props} data={playlists} label={label} icon={icon} />
  )
}

export default ChangePublicStatusButton
