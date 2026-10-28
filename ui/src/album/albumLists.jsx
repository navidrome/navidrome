import React from 'react'
import ShuffleIcon from '@mui/icons-material/Shuffle'
import LibraryAddIcon from '@mui/icons-material/LibraryAdd'
import VideoLibraryIcon from '@mui/icons-material/VideoLibrary'
import RepeatIcon from '@mui/icons-material/Repeat'
import AlbumIcon from '@mui/icons-material/Album'
import FavoriteIcon from '@mui/icons-material/Favorite'
import FavoriteBorderIcon from '@mui/icons-material/FavoriteBorder'
import StarIcon from '@mui/icons-material/Star'
import StarBorderIcon from '@mui/icons-material/StarBorder'
import AlbumOutlinedIcon from '@mui/icons-material/AlbumOutlined'
import LibraryAddOutlinedIcon from '@mui/icons-material/LibraryAddOutlined'
import VideoLibraryOutlinedIcon from '@mui/icons-material/VideoLibraryOutlined'
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
