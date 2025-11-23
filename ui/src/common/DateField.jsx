import React from 'react'
import { isDateSet } from '../utils/validations'
import { DateField as RADateField } from 'react-admin'

export const DateField = (props) => {
  const { record, source } = props
  const value = record?.[source]
  if (!isDateSet(value)) return null
  return <RADateField {...props} />
}

DateField.defaultProps = {
  addLabel: true,
}
