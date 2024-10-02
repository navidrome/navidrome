import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { setOmittedFields, setToggleableFields } from '../actions'

// TODO Refactor
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
      Object.keys(resourceFields).length !== Object.keys(columns).length ||
      !Object.keys(columns).every((c) => c in resourceFields)
    ) {
      const obj = {}
      for (const key of Object.keys(columns)) {
        obj[key] = !defaultOff.includes(key)
      }
      dispatch(setToggleableFields({ [resource]: obj }))
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
      const filtered = []
      const omitted = omittedColumns
      for (const [key, val] of Object.entries(columns)) {
        if (!val) omitted.push(key)
        else if (resourceFields[key]) filtered.push(val)
      }
      if (filteredComponents.length !== filtered.length)
        setFilteredComponents(filtered)
      if (omittedFields.length !== omitted.length)
        dispatch(setOmittedFields({ [resource]: omitted }))
    }
  }, [
    resourceFields,
    columns,
    dispatch,
    omittedColumns,
    omittedFields,
    resource,
    filteredComponents.length,
  ])

  return React.Children.toArray(filteredComponents)
}

useSelectedFields.propTypes = {
  resource: PropTypes.string,
  columns: PropTypes.object,
  omittedColumns: PropTypes.arrayOf(PropTypes.string),
  defaultOff: PropTypes.arrayOf(PropTypes.string),
}

export const useSetToggleableFields = (
  resource,
  toggleableColumns,
  defaultOff = [],
) => {
  const current = useSelector((state) => state.settings.toggleableFields)?.album
  const dispatch = useDispatch()
  useEffect(() => {
    if (!current) {
      dispatch(
        setToggleableFields({
          [resource]: toggleableColumns.reduce((acc, cur) => {
            return {
              ...acc,
              ...{ [cur]: true },
            }
          }, {}),
        }),
      )
      dispatch(setOmittedFields({ [resource]: defaultOff }))
    }
  }, [resource, toggleableColumns, dispatch, current, defaultOff])
}

useSetToggleableFields.propTypes = {
  resource: PropTypes.string,
  toggleableColumns: PropTypes.arrayOf(PropTypes.string),
  defaultOff: PropTypes.arrayOf(PropTypes.string),
}
