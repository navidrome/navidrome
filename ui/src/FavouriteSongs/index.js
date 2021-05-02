import React from 'react'
import FavouriteSongList from './FavouriteSongList'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
export default {
  list: FavouriteSongList,
  icon: (
    <DynamicMenuIcon
      path={'favouriteSong'}
      icon={FavoriteBorderIcon}
      activeIcon={FavoriteIcon}
    />
  ),
}
