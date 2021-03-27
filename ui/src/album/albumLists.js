import ShuffleIcon from '@material-ui/icons/Shuffle'
import LibraryAddIcon from '@material-ui/icons/LibraryAdd'
import VideoLibraryIcon from '@material-ui/icons/VideoLibrary'
import RepeatIcon from '@material-ui/icons/Repeat'
import AlbumIcon from '@material-ui/icons/Album'
import FavoriteIcon from '@material-ui/icons/Favorite'
import config from '../config'

export default {
  all: {
    icon: AlbumIcon,
    params: 'sort=name&order=ASC',
  },
  random: { icon: ShuffleIcon, params: 'sort=random' },
  ...(config.enableFavourites && {
    starred: {
      icon: FavoriteIcon,
      params: 'sort=starred_at&order=DESC&filter={"starred":true}',
    },
  }),
  recentlyAdded: {
    icon: LibraryAddIcon,
    params: 'sort=recently_added&order=DESC',
  },
  recentlyPlayed: {
    icon: VideoLibraryIcon,
    params: 'sort=play_date&order=DESC&filter={"recently_played":true}',
  },
  mostPlayed: {
    icon: RepeatIcon,
    params: 'sort=play_count&order=DESC&filter={"recently_played":true}',
  },
}

export const defaultAlbumList = 'recentlyAdded'
