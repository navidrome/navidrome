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

export default {
  all: {
    icon: AlbumOutlinedIcon,
    onActive: AlbumIcon,
    params: 'sort=name&order=ASC',
  },
  random: {
    icon: ShuffleIcon,
    onActive: ShuffleIcon,
    params: 'sort=random&order=ASC',
  },
  ...(config.enableFavourites && {
    starred: {
      icon: FavoriteBorderIcon,
      onActive: FavoriteIcon,
      params: 'sort=starred_at&order=DESC&filter={"starred":true}',
    },
  }),
  ...(config.enableStarRating && {
    topRated: {
      icon: StarBorderIcon,
      onActive: StarIcon,
      params: 'sort=rating&order=DESC&filter={"has_rating":true}',
    },
  }),
  recentlyAdded: {
    icon: LibraryAddOutlinedIcon,
    onActive: LibraryAddIcon,
    params: 'sort=recently_added&order=DESC',
  },
  recentlyPlayed: {
    icon: VideoLibraryOutlinedIcon,
    onActive: VideoLibraryIcon,
    params: 'sort=play_date&order=DESC&filter={"recently_played":true}',
  },
  mostPlayed: {
    icon: RepeatIcon,
    onActive: RepeatIcon,
    params: 'sort=play_count&order=DESC&filter={"recently_played":true}',
  },
}

export const defaultAlbumList = 'recentlyAdded'
