/* eslint-disable no-use-before-define */
import React from 'react'
import TextField from '@material-ui/core/TextField'
import Autocomplete, {
  createFilterOptions,
} from '@material-ui/lab/Autocomplete'
import { useGetList, useTranslate } from 'react-admin'
import PropTypes from 'prop-types'
import { isWritable } from '../common'

const filter = createFilterOptions()

export const SelectPlaylistInput = ({ onChange }) => {
  const translate = useTranslate()
  const { ids, data } = useGetList(
    'playlist',
    { page: 1, perPage: -1 },
    { field: 'name', order: 'ASC' },
    {}
  )

  const options =
    ids &&
    ids.map((id) => data[id]).filter((option) => isWritable(option.owner))

  const handleOnChange = (event, newValue) => {
    if (newValue == null) {
      onChange({})
    } else if (typeof newValue === 'string') {
      onChange({
        name: newValue,
      })
    } else if (newValue && newValue.inputValue) {
      // Create a new value from the user input
      onChange({
        name: newValue.inputValue,
      })
    } else {
      onChange(newValue)
    }
  }

  return (
    <Autocomplete
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
      renderOption={(option) => option.name}
      style={{ width: 300 }}
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
