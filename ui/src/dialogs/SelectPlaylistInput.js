import React from 'react'
import TextField from '@material-ui/core/TextField'
import Checkbox from '@material-ui/core/Checkbox'
import CheckBoxIcon from '@material-ui/icons/CheckBox'
import CheckBoxOutlineBlankIcon from '@material-ui/icons/CheckBoxOutlineBlank'
import Autocomplete, {
  createFilterOptions,
} from '@material-ui/lab/Autocomplete'
import { useGetList, useTranslate } from 'react-admin'
import PropTypes from 'prop-types'
import { isWritable } from '../common'
import { makeStyles } from '@material-ui/core'

const filter = createFilterOptions()

const useStyles = makeStyles({
  root: { width: '100%' },
  checkbox: { marginRight: 8 },
})

export const SelectPlaylistInput = ({ onChange }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const { ids, data } = useGetList(
    'playlist',
    { page: 1, perPage: -1 },
    { field: 'name', order: 'ASC' },
    { smart: false },
  )

  const options =
    ids &&
    ids.map((id) => data[id]).filter((option) => isWritable(option.ownerId))

  const handleOnChange = (event, newValue) => {
    let newState = []
    if (newValue && newValue.length) {
      newValue.forEach((playlistObject) => {
        if (playlistObject.inputValue) {
          newState.push({
            name: playlistObject.inputValue,
          })
        } else if (typeof playlistObject === 'string') {
          newState.push({
            name: playlistObject,
          })
        } else {
          newState.push(playlistObject)
        }
      })
    }
    onChange(newState)
  }

  const icon = <CheckBoxOutlineBlankIcon fontSize="small" />
  const checkedIcon = <CheckBoxIcon fontSize="small" />

  return (
    <Autocomplete
      multiple
      disableCloseOnSelect
      onChange={handleOnChange}
      filterOptions={(options, params) => {
        const filtered = filter(options, params)

        // Suggest the creation of a new value
        if (params.inputValue !== '') {
          filtered.push({
            inputValue: params.inputValue,
            name: translate('resources.playlist.actions.addNewPlaylist', {
              name: params.inputValue,
            }),
          })
        }

        return filtered
      }}
      clearOnBlur
      handleHomeEndKeys
      openOnFocus
      selectOnFocus
      id="select-playlist-input"
      options={options}
      getOptionLabel={(option) => {
        // Value selected with enter, right from the input
        if (typeof option === 'string') {
          return option
        }
        // Add "xxx" option created dynamically
        if (option.inputValue) {
          return option.inputValue
        }
        // Regular option
        return option.name
      }}
      renderOption={(option, { selected }) => (
        <React.Fragment>
          <Checkbox
            icon={icon}
            checkedIcon={checkedIcon}
            className={classes.checkbox}
            checked={selected}
          />
          {option.name}
        </React.Fragment>
      )}
      className={classes.root}
      freeSolo
      renderInput={(params) => (
        <TextField
          autoFocus
          variant={'outlined'}
          {...params}
          label={translate('resources.playlist.fields.name')}
        />
      )}
    />
  )
}

SelectPlaylistInput.propTypes = {
  onChange: PropTypes.func.isRequired,
}
