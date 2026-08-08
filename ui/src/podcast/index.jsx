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
  create: PodcastChannelCreate,
  icon: (
    <DynamicMenuIcon
      path={'podcastChannel'}
      icon={RssFeedOutlinedIcon}
      activeIcon={RssFeedIcon}
    />
  ),
}

// Subscribing (create) and viewing/managing a personal subscription (show) are available to any
// user - see core/podcasts.Subscribe/Unsubscribe, which only require an authenticated caller, not
// admin. Editing the shared channel's own metadata (title/url/cover art) stays admin-only, since
// that's genuinely shared infrastructure gated at the persistence layer.
const admin = {
  ...all,
  edit: PodcastChannelEdit,
}

export default { all, admin }
