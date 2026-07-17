import React, { useCallback, useEffect, useState } from 'react'
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
  InputAdornment,
  IconButton,
} from '@material-ui/core'
import AddIcon from '@material-ui/icons/Add'
import { useNotify, useTranslate } from 'react-admin'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core'
import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'

const useStyles = makeStyles((theme) => ({
  root: {
    width: '100%',
  },
  searchField: {
    marginBottom: theme.spacing(2),
    width: '100%',
  },
  tagList: {
    maxHeight: '17em',
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
  emptyMessage: {
    padding: theme.spacing(2),
    textAlign: 'center',
    color: theme.palette.text.secondary,
  },
}))

const jsonHeaders = () =>
  new Headers({
    Accept: 'application/json',
    'Content-Type': 'application/json',
  })

export const SelectTagInput = ({ mediaFileId }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const [searchText, setSearchText] = useState('')
  const [allTags, setAllTags] = useState([])
  const [songTags, setSongTags] = useState([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    Promise.all([
      httpClient(`${REST_URL}/mediaFileTag/names`),
      httpClient(
        `${REST_URL}/mediaFileTag?media_file_id=${encodeURIComponent(mediaFileId)}`,
      ),
    ])
      .then(([names, tags]) => {
        setAllTags(names.json || [])
        setSongTags(tags.json || [])
      })
      .catch(() => notify('ra.page.error', { type: 'warning' }))
  }, [mediaFileId, notify])

  const toggleTag = useCallback(
    (tagName) => {
      const isTagged = songTags.includes(tagName)
      setLoading(true)
      httpClient(`${REST_URL}/mediaFileTag`, {
        method: isTagged ? 'DELETE' : 'POST',
        headers: jsonHeaders(),
        body: JSON.stringify({ mediaFileId, tagName }),
      })
        .then(() => {
          setSongTags((prev) =>
            isTagged ? prev.filter((t) => t !== tagName) : [...prev, tagName],
          )
          setAllTags((prev) =>
            prev.includes(tagName) ? prev : [...prev, tagName].sort(),
          )
        })
        .catch(() => notify('ra.page.error', { type: 'warning' }))
        .finally(() => setLoading(false))
    },
    [mediaFileId, songTags, notify],
  )

  const handleCreateNew = () => {
    const name = searchText.trim()
    if (name) {
      toggleTag(name)
      setSearchText('')
    }
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && searchText.trim()) {
      e.preventDefault()
      handleCreateNew()
    }
  }

  const filteredTags = allTags.filter((tag) =>
    tag.toLowerCase().includes(searchText.toLowerCase()),
  )

  const canCreateNew = Boolean(
    searchText.trim() &&
    !allTags.some(
      (tag) => tag.toLowerCase() === searchText.toLowerCase().trim(),
    ),
  )

  return (
    <div className={classes.root}>
      <TextField
        autoFocus
        variant="outlined"
        className={classes.searchField}
        label={translate('resources.song.message.selectTags')}
        value={searchText}
        onChange={(e) => setSearchText(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={translate('resources.song.message.searchOrCreateTag')}
        InputProps={{
          endAdornment: canCreateNew && (
            <InputAdornment position="end">
              <IconButton
                onClick={handleCreateNew}
                size="small"
                disabled={loading}
                title={translate('resources.song.message.addNewTag', {
                  name: searchText,
                })}
              >
                <AddIcon />
              </IconButton>
            </InputAdornment>
          ),
        }}
      />
      <List className={classes.tagList}>
        {filteredTags.length === 0 ? (
          <div className={classes.emptyMessage}>
            <Typography variant="body2">
              {searchText
                ? translate('resources.song.message.noTagsFound')
                : translate('resources.song.message.noTags')}
            </Typography>
            {canCreateNew && (
              <Typography variant="body2" color="primary">
                {translate('resources.song.message.pressEnterToCreateTag')}
              </Typography>
            )}
          </div>
        ) : (
          filteredTags.map((tag) => (
            <ListItem
              key={tag}
              className={classes.listItem}
              button
              onClick={() => toggleTag(tag)}
              disabled={loading}
              dense
            >
              <ListItemIcon>
                <Checkbox
                  icon={<CheckBoxOutlineBlankIcon fontSize="small" />}
                  checkedIcon={<CheckBoxIcon fontSize="small" />}
                  checked={songTags.includes(tag)}
                  tabIndex={-1}
                  disableRipple
                />
              </ListItemIcon>
              <ListItemText primary={tag} />
            </ListItem>
          ))
        )}
        {canCreateNew && filteredTags.length > 0 && (
          <ListItem
            className={classes.listItem}
            button
            onClick={handleCreateNew}
            disabled={loading}
            dense
          >
            <ListItemIcon>
              <AddIcon className={classes.createIcon} />
            </ListItemIcon>
            <ListItemText
              primary={translate('resources.song.message.addNewTag', {
                name: searchText,
              })}
            />
          </ListItem>
        )}
      </List>
    </div>
  )
}

SelectTagInput.propTypes = {
  mediaFileId: PropTypes.string.isRequired,
}
