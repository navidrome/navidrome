import React, { cloneElement } from 'react'
import {
  sanitizeListRestProps,
  TopToolbar,
  CreateButton,
  useTranslate,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { ToggleFieldsMenu } from '../common'
import { ImportButton } from './ImportButton'
import config from '../config'

const PlaylistListActions = ({ className, ...rest }) => {
  const isNotSmall = useMediaQuery((theme) => theme.breakpoints.up('sm'))
  const translate = useTranslate()

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {cloneElement(rest.filters, { context: 'button' })}
      <CreateButton basePath="/playlist">
        {translate('ra.action.create')}
      </CreateButton>
      {config.listenBrainzEnabled && <ImportButton />}

      {isNotSmall && <ToggleFieldsMenu resource="playlist" />}
    </TopToolbar>
  )
}

export default PlaylistListActions
