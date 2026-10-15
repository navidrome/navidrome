import React from 'react'
import { isDateSet } from '../utils/validations'
import { DateField as RADateField } from 'react-admin'
import { useDateLocale } from '../i18n/useDateLocale'

export const DateField = (props) => {
  const { record, source } = props
  const locale = useDateLocale()
  const value = record?.[source]
  if (!isDateSet(value)) return null
  return <RADateField locales={locale} {...props} />
}

DateField.defaultProps = {
  addLabel: true,
}
