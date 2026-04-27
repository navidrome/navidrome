import React from 'react'
import MicIcon from '@material-ui/icons/Mic'
import MicNoneIcon from '@material-ui/icons/MicNone'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import PodcastList from './PodcastList'
import PodcastShow from './PodcastShow'
import PodcastCreate from './PodcastCreate'

const all = {
  list: PodcastList,
  show: PodcastShow,
  icon: (
    <DynamicMenuIcon
      path={'podcast'}
      icon={MicNoneIcon}
      activeIcon={MicIcon}
    />
  ),
}

const admin = {
  ...all,
  create: PodcastCreate,
}

export default { all, admin }
