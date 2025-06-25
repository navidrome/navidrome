import React, { useState, useEffect } from 'react'
import Checkbox from '@material-ui/core/Checkbox'
import CheckBoxIcon from '@material-ui/icons/CheckBox'
import CheckBoxOutlineBlankIcon from '@material-ui/icons/CheckBoxOutlineBlank'
import {
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Typography,
} from '@material-ui/core'
import { useGetList } from 'react-admin'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core'

const useStyles = makeStyles((theme) => ({
  root: {
    width: '960px', // MD breakpoint width
    maxWidth: '100%',
  },
  libraryList: {
    height: '120px', // 3 rows of libraries
    overflow: 'auto',
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    backgroundColor: theme.palette.background.paper,
  },
  listItem: {
    paddingTop: 0,
    paddingBottom: 0,
  },
  emptyMessage: {
    padding: theme.spacing(2),
    textAlign: 'center',
    color: theme.palette.text.secondary,
  },
}))

const EmptyLibraryMessage = () => {
  const classes = useStyles()

  return (
    <div className={classes.emptyMessage}>
      <Typography variant="body2">No libraries available</Typography>
    </div>
  )
}

const LibraryListItem = ({ library, isSelected, onToggle }) => {
  const classes = useStyles()

  return (
    <ListItem
      className={classes.listItem}
      button
      onClick={() => onToggle(library)}
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
      <ListItemText primary={library.name} />
    </ListItem>
  )
}

export const SelectLibraryInput = ({ onChange, value = [] }) => {
  const classes = useStyles()
  const [selectedLibraryIds, setSelectedLibraryIds] = useState([])

  const { ids, data } = useGetList(
    'library',
    { page: 1, perPage: -1 },
    { field: 'name', order: 'ASC' },
    { smart: false },
  )

  const options = (ids && ids.map((id) => data[id])) || []

  // Update selectedLibraryIds when value prop changes (for editing mode)
  useEffect(() => {
    if (Array.isArray(value)) {
      const libraryIds = value.map((item) =>
        typeof item === 'object' ? item.id : item,
      )
      setSelectedLibraryIds(libraryIds)
    }
  }, [value])

  const isLibrarySelected = (library) => selectedLibraryIds.includes(library.id)

  const handleLibraryToggle = (library) => {
    const isSelected = selectedLibraryIds.includes(library.id)
    let newSelection

    if (isSelected) {
      newSelection = selectedLibraryIds.filter((id) => id !== library.id)
    } else {
      newSelection = [...selectedLibraryIds, library.id]
    }

    setSelectedLibraryIds(newSelection)
    onChange(newSelection)
  }

  return (
    <div className={classes.root}>
      <List className={classes.libraryList}>
        {options.length === 0 ? (
          <EmptyLibraryMessage />
        ) : (
          options.map((library) => (
            <LibraryListItem
              key={library.id}
              library={library}
              isSelected={isLibrarySelected(library)}
              onToggle={handleLibraryToggle}
            />
          ))
        )}
      </List>
    </div>
  )
}

SelectLibraryInput.propTypes = {
  onChange: PropTypes.func.isRequired,
  value: PropTypes.array,
}

LibraryListItem.propTypes = {
  library: PropTypes.object.isRequired,
  isSelected: PropTypes.bool.isRequired,
  onToggle: PropTypes.func.isRequired,
}
