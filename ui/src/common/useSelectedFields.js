import React, { useState, useEffect } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { setOmittedFields, setToggleableFields } from '../actions'

const useSelectedFields = ({ resource, columns, omittedColumns = [] }) => {
  const dispatch = useDispatch()
  const resourceFields = useSelector(
    (state) => state.settings.toggleableFields
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
      const obj = {}
      for (const key of Object.keys(columns)) {
        obj[key] = true
      }
      dispatch(setToggleableFields({ [resource]: obj }))
    }
    if (!omittedFields) {
      dispatch(setOmittedFields({ [resource]: omittedColumns }))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resourceFields, columns])

  return React.Children.toArray(filteredComponents)
}

export default useSelectedFields

useSelectedFields.propTypes = {
  resource: PropTypes.string,
  columns: PropTypes.object,
  omittedColumns: PropTypes.arrayOf(PropTypes.string),
}

useSelectedFields.defaultProps = {
  omittedColumns: [],
}
