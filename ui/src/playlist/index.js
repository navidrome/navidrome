import React from 'react'
import QueueMusicOutlinedIcon from '@material-ui/icons/QueueMusicOutlined'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import PlaylistList from './PlaylistList'
import PlaylistEdit from './PlaylistEdit'
import PlaylistCreate from './PlaylistCreate'
import PlaylistShow from './PlaylistShow'

export default {
  list: PlaylistList,
  create: PlaylistCreate,
  edit: PlaylistEdit,
  show: PlaylistShow,
  icon: (
    <DynamicMenuIcon
      path={'playlist'}
      icon={QueueMusicOutlinedIcon}
      activeIcon={QueueMusicIcon}
    />
  ),
}
