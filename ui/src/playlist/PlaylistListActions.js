import React from 'react'
import {
  sanitizeListRestProps,
  TopToolbar,
  CreateButton,
  useTranslate,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import ToggleFieldsMenu from '../common/ToggleFieldsMenu'

const PlaylistListActions = ({ className, ...rest }) => {
  const isSmall = useMediaQuery((theme) => theme.breakpoints.up('sm'))
  const translate = useTranslate()

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <CreateButton
        basePath="/playlist"
        label={translate('ra.action.create')}
      />
      {isSmall && <ToggleFieldsMenu resource="playlist" />}
    </TopToolbar>
  )
}

export default PlaylistListActions
