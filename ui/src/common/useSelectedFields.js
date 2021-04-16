import React, { useEffect } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { setOmittedFields, setToggleableFields } from '../actions'

const useSelectedFields = ({ resource, columns, omittedColumns }) => {
  const dispatch = useDispatch()
  const resourceFields = useSelector(
    (state) => state.settings.toggleableFields
  )?.[resource]
  const omittedFields = useSelector((state) => state.settings.omittedFields)?.[
    resource
  ]

  useEffect(() => {
    // for rehydrating redux store with new release
    if (
      !resourceFields ||
      Object.keys(columns).length !== Object.keys(resourceFields).length
    ) {
      const obj = {}
      for (const key of Object.keys(columns)) {
        obj[key] = true
      }
      dispatch(setToggleableFields({ [resource]: obj }))
    }
    if (!omittedFields) {
      dispatch(setOmittedFields({ [resource]: [] }))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resourceFields, omittedFields, dispatch])

  const filteredComponents = []
  const omitted = omittedColumns
  if (resourceFields) {
    for (const [key, val] of Object.entries(columns)) {
      if (!val) {
        omitted.push(key)
      } else if (resourceFields[key]) {
        filteredComponents.push(val)
      }
    }
  }

  return [React.Children.toArray(filteredComponents), { [resource]: omitted }]
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
