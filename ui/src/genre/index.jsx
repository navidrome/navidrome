import React from 'react'
import GenreList from './GenreList'
import GenreShow from './GenreShow'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import CategoryOutlinedIcon from '@material-ui/icons/CategoryOutlined'
import CategoryIcon from '@material-ui/icons/Category'

export default {
  list: GenreList,
  show: GenreShow,
  icon: (
    <DynamicMenuIcon
      path={'genre'}
      icon={CategoryOutlinedIcon}
      activeIcon={CategoryIcon}
    />
  ),
}
