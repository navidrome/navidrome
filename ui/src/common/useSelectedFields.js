import React, { useEffect } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { setToggleableFields } from '../actions'

const useSelectedFields = ({ resource, columns }) => {
  const dispatch = useDispatch()
  const resourceFields = useSelector(
    (state) => state.settings.toggleableFields
  )?.[resource]

  useEffect(() => {
    if (!resourceFields || !Object.keys(resourceFields).length) {
      const obj = {}
      for (const key of Object.keys(columns)) {
        obj[key] = true
      }
      dispatch(setToggleableFields({ [resource]: obj }))
    }
  }, [resourceFields, dispatch])

  const filteredComponents = []
  if (resourceFields) {
    for (const [key, val] of Object.entries(columns)) {
      if (val && (resourceFields[key] || !resourceFields.hasOwnProperty(key))) {
        filteredComponents.push(val)
      }
    }
  }

  return React.Children.toArray(filteredComponents)
}

export default useSelectedFields

useSelectedFields.propTypes = {
  resource: PropTypes.string,
  columns: PropTypes.object,
}
