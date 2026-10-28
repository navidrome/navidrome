import React from 'react'
import SongList from './SongList'
import MusicNoteOutlinedIcon from '@mui/icons-material/MusicNoteOutlined'
import MusicNoteIcon from '@mui/icons-material/MusicNote'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'

export default {
  list: SongList,
  icon: (
    <DynamicMenuIcon
      path={'song'}
      icon={MusicNoteOutlinedIcon}
      activeIcon={MusicNoteIcon}
    />
  ),
}
