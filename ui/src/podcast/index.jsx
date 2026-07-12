import PodcastChannelCreate from './PodcastChannelCreate'
import PodcastChannelEdit from './PodcastChannelEdit'
import PodcastChannelList from './PodcastChannelList'
import PodcastChannelShow from './PodcastChannelShow'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import RssFeedIcon from '@material-ui/icons/RssFeed'
import RssFeedOutlinedIcon from '@material-ui/icons/RssFeedOutlined'
import React from 'react'

const all = {
  list: PodcastChannelList,
  show: PodcastChannelShow,
  icon: (
    <DynamicMenuIcon
      path={'podcastChannel'}
      icon={RssFeedOutlinedIcon}
      activeIcon={RssFeedIcon}
    />
  ),
}

const admin = {
  ...all,
  create: PodcastChannelCreate,
  edit: PodcastChannelEdit,
}

export default { all, admin }
