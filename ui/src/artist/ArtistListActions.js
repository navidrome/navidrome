import React from 'react'
import { sanitizeListRestProps, TopToolbar } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import ToggleFieldsMenu from '../common/ToggleFieldsMenu'

const ArtistListActions = ({ className, ...rest }) => {
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {isDesktop && <ToggleFieldsMenu resource="artist" />}
    </TopToolbar>
  )
}

export default ArtistListActions
