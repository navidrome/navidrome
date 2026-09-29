import React from 'react'
import SvgIcon from '@material-ui/core/SvgIcon'
import PlaylistPlayIcon from '@material-ui/icons/PlaylistPlay'
import { MdOutlineAutoAwesome } from 'react-icons/md'
import { useTranslate } from 'react-admin'
import { isSmartPlaylist } from './playlistUtils'

// Both icons name themselves: MenuItemLink clones its leftIcon with
// titleAccess=primaryText, which for a playlist entry is a node, not a string.
// Neither holds a ref (react-icons components don't accept one), so wrap them
// in an element of your own to hang a Tooltip off.

export const SmartPlaylistIcon = (props) => {
  const translate = useTranslate()
  const label = translate('resources.playlist.message.smartPlaylist')
  return (
    <SvgIcon
      {...props}
      component={MdOutlineAutoAwesome}
      // react-icons replaces SvgIcon's children, dropping the <title> it builds
      // from titleAccess, so pass its own `title` prop too. titleAccess still
      // earns the icon its role="img".
      title={label}
      titleAccess={label}
    />
  )
}

export const RegularPlaylistIcon = (props) => {
  const translate = useTranslate()
  return (
    <PlaylistPlayIcon
      {...props}
      titleAccess={translate('resources.playlist.name', { smart_count: 1 })}
    />
  )
}

export const PlaylistTypeIcon = ({ record, ...props }) =>
  isSmartPlaylist(record) ? (
    <SmartPlaylistIcon {...props} />
  ) : (
    <RegularPlaylistIcon {...props} />
  )
