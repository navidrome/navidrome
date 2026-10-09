import React from 'react'
import { isDateSet } from '../utils/validations'
import { DateField as RADateField, useRecordContext } from 'react-admin'
import { useDateLocale } from '../i18n/useDateLocale'

export const DateField = (props) => {
  const record = useRecordContext(props)
  const locale = useDateLocale()
  const value = record?.[props.source]
  if (!isDateSet(value)) return null
  return <RADateField locales={locale} {...props} />
}

DateField.defaultProps = {
  addLabel: true,
}
