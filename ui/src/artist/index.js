import React from 'react'
import ArtistList from './ArtistList'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import MicNoneOutlinedIcon from '@material-ui/icons/MicNoneOutlined'
import MicIcon from '@material-ui/icons/Mic'
import ArtistView from '../common/ArtistDetail'

export default {
  list: ArtistList,
  show: ArtistView,
  icon: (
    <DynamicMenuIcon
      path={'artist'}
      icon={MicNoneOutlinedIcon}
      activeIcon={MicIcon}
    />
  ),
}
