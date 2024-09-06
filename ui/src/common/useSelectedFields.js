import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { setOmittedFields, setToggleableFields } from '../actions'

export const useSelectedFields = ({
  resource,
  columns,
  omittedColumns = [],
  defaultOff = [],
}) => {
  const dispatch = useDispatch()
  const resourceFields = useSelector(
    (state) => state.settings.toggleableFields,
  )?.[resource]
  const omittedFields = useSelector((state) => state.settings.omittedFields)?.[
    resource
  ]

  const [filteredComponents, setFilteredComponents] = useState([])

  useEffect(() => {
    if (
      !resourceFields ||
      Object.keys(resourceFields).length !== Object.keys(columns).length
    ) {
      const fieldState = Object.fromEntries(
        Object.keys(columns).map((key) => [key, !defaultOff.includes(key)]),
      )
      dispatch(setToggleableFields({ [resource]: fieldState }))
    }

    if (!omittedFields) {
      dispatch(setOmittedFields({ [resource]: omittedColumns }))
    }
  }, [
    columns,
    defaultOff,
    dispatch,
    omittedColumns,
    omittedFields,
    resource,
    resourceFields,
  ])

  useEffect(() => {
    if (resourceFields) {
      const filtered = Object.entries(columns)
        .filter(([key, val]) => resourceFields[key] && val)
        .map(([_, val]) => val)

      const omitted = omittedColumns.concat(
        Object.keys(columns).filter((key) => !columns[key]),
      )
      if (filteredComponents.length !== filtered.length)
        setFilteredComponents(filtered)
      if (omittedFields?.length !== omitted.length)
        dispatch(setOmittedFields({ [resource]: omitted }))
    }
  }, [
    resourceFields,
    columns,
    defaultOff,
    omittedFields,
    omittedColumns,
    dispatch,
    resource,
    filteredComponents.length,
  ])

  return React.Children.toArray(filteredComponents)
}

useSelectedFields.propTypes = {
  resource: PropTypes.string.isRequired,
  columns: PropTypes.object.isRequired,
  omittedColumns: PropTypes.arrayOf(PropTypes.string),
  defaultOff: PropTypes.arrayOf(PropTypes.string),
}

export const useSetToggleableFields = (
  resource,
  toggleableColumns,
  defaultOff = [],
) => {
  const currentFields = useSelector(
    (state) => state.settings.toggleableFields?.[resource],
  )
  const dispatch = useDispatch()

  useEffect(() => {
    if (!currentFields) {
      const initialToggleState = Object.fromEntries(
        toggleableColumns.map((col) => [col, true]),
      )
      dispatch(setToggleableFields({ [resource]: initialToggleState }))
      dispatch(setOmittedFields({ [resource]: defaultOff }))
    }
  }, [currentFields, toggleableColumns, defaultOff, dispatch, resource])
}

useSetToggleableFields.propTypes = {
  resource: PropTypes.string.isRequired,
  toggleableColumns: PropTypes.arrayOf(PropTypes.string).isRequired,
  defaultOff: PropTypes.arrayOf(PropTypes.string),
}
