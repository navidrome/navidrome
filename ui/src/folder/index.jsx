import React from 'react'
import FolderList from './FolderList'
import FolderShow from './FolderShow'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import FolderOutlinedIcon from '@material-ui/icons/FolderOutlined'
import FolderIcon from '@material-ui/icons/Folder'

export default {
  list: FolderList,
  show: FolderShow,
  icon: (
    <DynamicMenuIcon
      path={'folder'}
      icon={FolderOutlinedIcon}
      activeIcon={FolderIcon}
    />
  ),
}
