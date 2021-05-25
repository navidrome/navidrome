import React from 'react'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import LibraryAddIcon from '@material-ui/icons/LibraryAdd'
import VideoLibraryIcon from '@material-ui/icons/VideoLibrary'
import RepeatIcon from '@material-ui/icons/Repeat'
import AlbumIcon from '@material-ui/icons/Album'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import AlbumOutlinedIcon from '@material-ui/icons/AlbumOutlined'
import LibraryAddOutlinedIcon from '@material-ui/icons/LibraryAddOutlined'
import VideoLibraryOutlinedIcon from '@material-ui/icons/VideoLibraryOutlined'
import config from '../config'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'

const albumLists = {
  all: {
    icon: (
      <DynamicMenuIcon
        path={'album/all'}
        icon={AlbumOutlinedIcon}
        activeIcon={AlbumIcon}
      />
    ),
    params: 'sort=name&order=ASC&filter={}',
  },
  random: {
    icon: <ShuffleIcon />,
    params: 'sort=random&order=ASC&filter={}',
  },
  ...(config.enableFavourites && {
    starred: {
      icon: (
        <DynamicMenuIcon
          path={'album/starred'}
          icon={FavoriteBorderIcon}
          activeIcon={FavoriteIcon}
        />
      ),
      params: 'sort=starred_at&order=DESC&filter={"starred":true}',
    },
  }),
  ...(config.enableStarRating && {
    topRated: {
      icon: (
        <DynamicMenuIcon
          path={'album/topRated'}
          icon={StarBorderIcon}
          activeIcon={StarIcon}
        />
      ),
      params: 'sort=rating&order=DESC&filter={"has_rating":true}',
    },
  }),
  recentlyAdded: {
    icon: (
      <DynamicMenuIcon
        path={'album/recentlyAdded'}
        icon={LibraryAddOutlinedIcon}
        activeIcon={LibraryAddIcon}
      />
    ),
    params: 'sort=recently_added&order=DESC&filter={}',
  },
  recentlyPlayed: {
    icon: (
      <DynamicMenuIcon
        path={'album/recentlyPlayed'}
        icon={VideoLibraryOutlinedIcon}
        activeIcon={VideoLibraryIcon}
      />
    ),
    params: 'sort=play_date&order=DESC&filter={"recently_played":true}',
  },
  mostPlayed: {
    icon: <RepeatIcon />,
    params: 'sort=play_count&order=DESC&filter={"recently_played":true}',
  },
}

export default albumLists
export const defaultAlbumList = 'recentlyAdded'
