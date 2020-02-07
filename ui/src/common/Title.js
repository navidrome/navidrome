import React from 'react'
import { useMediaQuery } from '@material-ui/core'

const Title = ({ subTitle }) => {
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))

  if (isDesktop) {
    return <span>Navidrome {subTitle ? ` - ${subTitle}` : ''}</span>
  }
  return <span>{subTitle ? subTitle : 'Navidrome'}</span>
}

export default Title
