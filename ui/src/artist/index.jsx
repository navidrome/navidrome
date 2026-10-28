import React from 'react'
import ArtistList from './ArtistList'
import ArtistShow from './ArtistShow'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import MicNoneOutlinedIcon from '@mui/icons-material/MicNoneOutlined'
import MicIcon from '@mui/icons-material/Mic'

export default {
  list: ArtistList,
  show: ArtistShow,
  icon: (
    <DynamicMenuIcon
      path={'artist'}
      icon={MicNoneOutlinedIcon}
      activeIcon={MicIcon}
    />
  ),
}
