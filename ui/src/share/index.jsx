import ShareList from './ShareList'
import { ShareEdit } from './ShareEdit'
import ShareIcon from '@mui/icons-material/Share'
import ShareOutlinedIcon from '@mui/icons-material/ShareOutlined'
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
