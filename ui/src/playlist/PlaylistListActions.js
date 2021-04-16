import React from 'react'
import { sanitizeListRestProps, TopToolbar, CreateButton } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import ToggleFieldsMenu from '../common/ToggleFieldsMenu'

const PlaylistListActions = ({ className, ...rest }) => {
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <CreateButton basePath="/playlist" label="Create Playlist" />
      {isDesktop && <ToggleFieldsMenu resource="playlist" />}
    </TopToolbar>
  )
}

export default PlaylistListActions
