import React from 'react'
import { DateField as RADateField } from 'react-admin'

export const DateField = (props) => {
  const { record, source } = props
  const value = record?.[source]
  if (value === '0001-01-01T00:00:00Z' || value === null) return null
  return <RADateField {...props} />
}

DateField.defaultProps = {
  addLabel: true,
}
