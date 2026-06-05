import React, { useState, useEffect, useMemo } from 'react'
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
} from '@material-ui/core'
import { useGetList, useTranslate } from 'react-admin'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core'

const useStyles = makeStyles((theme) => ({
  root: {
    width: '960px',
    maxWidth: '100%',
  },
  headerContainer: {
    display: 'flex',
    alignItems: 'center',
    marginBottom: theme.spacing(1),
    paddingLeft: theme.spacing(1),
  },
  masterCheckbox: {
    padding: '7px',
    marginLeft: '-9px',
    marginRight: theme.spacing(1),
  },
  libraryList: {
    height: '120px',
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

export const SelectLibraryInput = ({
  onChange,
  value = [],
  isNewUser = false,
}) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [selectedLibraryIds, setSelectedLibraryIds] = useState([])
  const [hasInitialized, setHasInitialized] = useState(false)

  const { ids, data, isLoading } = useGetList(
    'library',
    { page: 1, perPage: -1 },
    { field: 'name', order: 'ASC' },
  )

  const options = useMemo(
    () => (ids && ids.map((id) => data[id])) || [],
    [ids, data],
  )

  // Reset initialization state when isNewUser changes
  useEffect(() => {
    if (isNewUser) {
      setHasInitialized(false)
    }
  }, [isNewUser])

  // Pre-select default libraries for new users
  useEffect(() => {
    if (
      isNewUser &&
      !isLoading &&
      options.length > 0 &&
      !hasInitialized &&
      Array.isArray(value) &&
      value.length === 0
    ) {
      const defaultLibraryIds = options
        .filter((lib) => lib.defaultNewUsers)
        .map((lib) => lib.id)

      if (defaultLibraryIds.length > 0) {
        setSelectedLibraryIds(defaultLibraryIds)
        onChange(defaultLibraryIds)
      }

      setHasInitialized(true)
    }
  }, [isNewUser, isLoading, options, hasInitialized, value, onChange])

  // Update selectedLibraryIds when value prop changes (for editing mode and pre-selection)
  useEffect(() => {
    // For new users, only sync from value prop if it has actual data
    // This prevents empty initial state from overriding our pre-selection
    if (isNewUser && Array.isArray(value) && value.length === 0) {
      return
    }

    if (Array.isArray(value)) {
      const libraryIds = value.map((item) =>
        typeof item === 'object' ? item.id : item,
      )
      setSelectedLibraryIds(libraryIds)
    } else if (value.length === 0) {
      // Handle case where value is explicitly set to empty array (for existing users)
      setSelectedLibraryIds([])
    }
  }, [value, isNewUser, hasInitialized])

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

  const handleMasterCheckboxChange = () => {
    const isAllSelected = selectedLibraryIds.length === options.length
    const newSelection = isAllSelected ? [] : options.map((lib) => lib.id)

    setSelectedLibraryIds(newSelection)
    onChange(newSelection)
  }

  const selectedCount = selectedLibraryIds.length
  const totalCount = options.length
  const isAllSelected = selectedCount === totalCount && totalCount > 0
  const isIndeterminate = selectedCount > 0 && selectedCount < totalCount

  return (
    <div className={classes.root}>
      {options.length > 1 && (
        <Box className={classes.headerContainer}>
          <Checkbox
            checked={isAllSelected}
            indeterminate={isIndeterminate}
            onChange={handleMasterCheckboxChange}
            size="small"
            className={classes.masterCheckbox}
          />
          <Typography variant="body2">
            {translate('resources.user.message.selectAllLibraries')}
          </Typography>
        </Box>
      )}
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
  isNewUser: PropTypes.bool,
}

LibraryListItem.propTypes = {
  library: PropTypes.object.isRequired,
  isSelected: PropTypes.bool.isRequired,
  onToggle: PropTypes.func.isRequired,
}
