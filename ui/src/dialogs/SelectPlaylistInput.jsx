import React, { useState } from 'react'
import TextField from '@material-ui/core/TextField'
import Checkbox from '@material-ui/core/Checkbox'
import CheckBoxIcon from '@material-ui/icons/CheckBox'
import CheckBoxOutlineBlankIcon from '@material-ui/icons/CheckBoxOutlineBlank'
import {
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Typography,
  Box,
  InputAdornment,
  IconButton,
} from '@material-ui/core'
import AddIcon from '@material-ui/icons/Add'
import { useGetList, useTranslate } from 'react-admin'
import PropTypes from 'prop-types'
import { isWritable } from '../common'
import { makeStyles } from '@material-ui/core'

const useStyles = makeStyles((theme) => ({
  root: {
    width: '100%',
    height: '100%',
    display: 'flex',
    flexDirection: 'column',
  },
  searchField: {
    marginBottom: theme.spacing(2),
    width: '100%',
    flexShrink: 0,
  },
  playlistList: {
    flex: 1,
    minHeight: 0,
    overflow: 'auto',
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    backgroundColor: theme.palette.background.paper,
  },
  listItem: {
    paddingTop: 0,
    paddingBottom: 0,
  },
  createIcon: {
    fontSize: '1.25rem',
    margin: '9px',
  },
  selectedPlaylistsContainer: {
    marginTop: theme.spacing(2),
    flexShrink: 0,
    maxHeight: '30%',
    overflow: 'auto',
  },
  selectedPlaylist: {
    display: 'inline-flex',
    alignItems: 'center',
    margin: theme.spacing(0.5),
    padding: theme.spacing(0.5, 1),
    backgroundColor: theme.palette.primary.main,
    color: theme.palette.primary.contrastText,
    borderRadius: theme.shape.borderRadius,
    fontSize: '0.875rem',
  },
  removeButton: {
    marginLeft: theme.spacing(0.5),
    padding: 2,
    color: 'inherit',
  },
  emptyMessage: {
    padding: theme.spacing(2),
    textAlign: 'center',
    color: theme.palette.text.secondary,
  },
}))

const PlaylistSearchField = ({
  searchText,
  onSearchChange,
  onCreateNew,
  onKeyDown,
  canCreateNew,
}) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <TextField
      autoFocus
      variant="outlined"
      className={classes.searchField}
      label={translate('resources.playlist.fields.name')}
      value={searchText}
      onChange={(e) => onSearchChange(e.target.value)}
      onKeyDown={onKeyDown}
      placeholder={translate('resources.playlist.actions.searchOrCreate')}
      InputProps={{
        endAdornment: canCreateNew && (
          <InputAdornment position="end">
            <IconButton
              onClick={onCreateNew}
              size="small"
              title={translate('resources.playlist.actions.addNewPlaylist', {
                name: searchText,
              })}
            >
              <AddIcon />
            </IconButton>
          </InputAdornment>
        ),
      }}
    />
  )
}

const EmptyPlaylistMessage = ({ searchText, canCreateNew }) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <div className={classes.emptyMessage}>
      <Typography variant="body2">
        {searchText
          ? translate('resources.playlist.message.noPlaylistsFound')
          : translate('resources.playlist.message.noPlaylists')}
      </Typography>
      {canCreateNew && (
        <Typography variant="body2" color="primary">
          {translate('resources.playlist.actions.pressEnterToCreate')}
        </Typography>
      )}
    </div>
  )
}

const PlaylistListItem = ({ playlist, isSelected, onToggle }) => {
  const classes = useStyles()

  return (
    <ListItem
      className={classes.listItem}
      button
      onClick={() => onToggle(playlist)}
      dense
    >
      <ListItemIcon>
        <Checkbox
          icon={<CheckBoxOutlineBlankIcon fontSize="small" />}
          checkedIcon={<CheckBoxIcon fontSize="small" />}
          checked={isSelected}
          tabIndex={-1}
          disableRipple
        />
      </ListItemIcon>
      <ListItemText primary={playlist.name} />
    </ListItem>
  )
}

const CreatePlaylistItem = ({ searchText, onCreateNew }) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <ListItem className={classes.listItem} button onClick={onCreateNew} dense>
      <ListItemIcon>
        <AddIcon className={classes.createIcon} />
      </ListItemIcon>
      <ListItemText
        primary={translate('resources.playlist.actions.addNewPlaylist', {
          name: searchText,
        })}
      />
    </ListItem>
  )
}

const PlaylistList = ({
  filteredOptions,
  selectedPlaylists,
  onPlaylistToggle,
  searchText,
  canCreateNew,
  onCreateNew,
}) => {
  const classes = useStyles()

  const isPlaylistSelected = (playlist) =>
    selectedPlaylists.some((p) => p.id === playlist.id)

  return (
    <List className={classes.playlistList}>
      {filteredOptions.length === 0 ? (
        <EmptyPlaylistMessage
          searchText={searchText}
          canCreateNew={canCreateNew}
        />
      ) : (
        filteredOptions.map((playlist) => (
          <PlaylistListItem
            key={playlist.id}
            playlist={playlist}
            isSelected={isPlaylistSelected(playlist)}
            onToggle={onPlaylistToggle}
          />
        ))
      )}
      {canCreateNew && filteredOptions.length > 0 && (
        <CreatePlaylistItem searchText={searchText} onCreateNew={onCreateNew} />
      )}
    </List>
  )
}

const SelectedPlaylistChip = ({ playlist, onRemove }) => {
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <span className={classes.selectedPlaylist}>
      {playlist.name}
      <IconButton
        className={classes.removeButton}
        size="small"
        onClick={() => onRemove(playlist)}
        title={translate('resources.playlist.actions.removeFromSelection')}
      >
        {'×'}
      </IconButton>
    </span>
  )
}

const SelectedPlaylistsDisplay = ({ selectedPlaylists, onRemoveSelected }) => {
  const classes = useStyles()
  const translate = useTranslate()

  if (selectedPlaylists.length === 0) {
    return null
  }

  return (
    <Box className={classes.selectedPlaylistsContainer}>
      <Box>
        {selectedPlaylists.map((playlist, index) => (
          <SelectedPlaylistChip
            key={playlist.id || `new-${index}`}
            playlist={playlist}
            onRemove={onRemoveSelected}
          />
        ))}
      </Box>
    </Box>
  )
}

export const SelectPlaylistInput = ({ onChange }) => {
  const classes = useStyles()
  const [searchText, setSearchText] = useState('')
  const [selectedPlaylists, setSelectedPlaylists] = useState([])

  const { ids, data } = useGetList(
    'playlist',
    { page: 1, perPage: -1 },
    { field: 'name', order: 'ASC' },
    { smart: false },
  )

  const options =
    ids &&
    ids.map((id) => data[id]).filter((option) => isWritable(option.ownerId))

  // Filter playlists based on search text
  const filteredOptions =
    options?.filter((option) =>
      option.name.toLowerCase().includes(searchText.toLowerCase()),
    ) || []

  const handlePlaylistToggle = (playlist) => {
    const isSelected = selectedPlaylists.some((p) => p.id === playlist.id)
    let newSelection

    if (isSelected) {
      newSelection = selectedPlaylists.filter((p) => p.id !== playlist.id)
    } else {
      newSelection = [...selectedPlaylists, playlist]
    }

    setSelectedPlaylists(newSelection)
    onChange(newSelection)
  }

  const handleRemoveSelected = (playlistToRemove) => {
    const newSelection = selectedPlaylists.filter(
      (p) => p.id !== playlistToRemove.id,
    )
    setSelectedPlaylists(newSelection)
    onChange(newSelection)
  }

  const handleCreateNew = () => {
    if (searchText.trim()) {
      const newPlaylist = { name: searchText.trim() }
      const newSelection = [...selectedPlaylists, newPlaylist]
      setSelectedPlaylists(newSelection)
      onChange(newSelection)
      setSearchText('')
    }
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && searchText.trim()) {
      e.preventDefault()
      handleCreateNew()
    }
  }

  const canCreateNew = Boolean(
    searchText.trim() &&
      !filteredOptions.some(
        (option) =>
          option.name.toLowerCase() === searchText.toLowerCase().trim(),
      ) &&
      !selectedPlaylists.some((p) => p.name === searchText.trim()),
  )

  return (
    <div className={classes.root}>
      <PlaylistSearchField
        searchText={searchText}
        onSearchChange={setSearchText}
        onCreateNew={handleCreateNew}
        onKeyDown={handleKeyDown}
        canCreateNew={canCreateNew}
      />

      <PlaylistList
        filteredOptions={filteredOptions}
        selectedPlaylists={selectedPlaylists}
        onPlaylistToggle={handlePlaylistToggle}
        searchText={searchText}
        canCreateNew={canCreateNew}
        onCreateNew={handleCreateNew}
      />

      <SelectedPlaylistsDisplay
        selectedPlaylists={selectedPlaylists}
        onRemoveSelected={handleRemoveSelected}
      />
    </div>
  )
}

SelectPlaylistInput.propTypes = {
  onChange: PropTypes.func.isRequired,
}

// PropTypes for sub-components
PlaylistSearchField.propTypes = {
  searchText: PropTypes.string.isRequired,
  onSearchChange: PropTypes.func.isRequired,
  onCreateNew: PropTypes.func.isRequired,
  onKeyDown: PropTypes.func.isRequired,
  canCreateNew: PropTypes.bool.isRequired,
}

EmptyPlaylistMessage.propTypes = {
  searchText: PropTypes.string.isRequired,
  canCreateNew: PropTypes.bool.isRequired,
}

PlaylistListItem.propTypes = {
  playlist: PropTypes.object.isRequired,
  isSelected: PropTypes.bool.isRequired,
  onToggle: PropTypes.func.isRequired,
}

CreatePlaylistItem.propTypes = {
  searchText: PropTypes.string.isRequired,
  onCreateNew: PropTypes.func.isRequired,
}

PlaylistList.propTypes = {
  filteredOptions: PropTypes.array.isRequired,
  selectedPlaylists: PropTypes.array.isRequired,
  onPlaylistToggle: PropTypes.func.isRequired,
  searchText: PropTypes.string.isRequired,
  canCreateNew: PropTypes.bool.isRequired,
  onCreateNew: PropTypes.func.isRequired,
}

SelectedPlaylistChip.propTypes = {
  playlist: PropTypes.object.isRequired,
  onRemove: PropTypes.func.isRequired,
}

SelectedPlaylistsDisplay.propTypes = {
  selectedPlaylists: PropTypes.array.isRequired,
  onRemoveSelected: PropTypes.func.isRequired,
}
