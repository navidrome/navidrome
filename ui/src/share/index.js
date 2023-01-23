import ShareList from './ShareList'
import { ShareEdit } from './ShareEdit'
import ShareIcon from '@material-ui/icons/Share'
import ShareOutlinedIcon from '@material-ui/icons/ShareOutlined'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import React from 'react'

export default {
  list: ShareList,
  edit: ShareEdit,
  icon: (
    <DynamicMenuIcon
      path={'share'}
      icon={ShareOutlinedIcon}
      activeIcon={ShareIcon}
    />
  ),
}
