import { SelectInput, useTranslate } from 'react-admin'
import { getStoredDateFormat } from '../i18n/useDateLocale'

export const SelectDateFormat = (props) => {
  const translate = useTranslate()

  return (
    <SelectInput
      {...props}
      source="dateFormat"
      label={translate('menu.personal.options.dateFormat')}
      defaultValue={getStoredDateFormat()}
      choices={[
        { id: 'language', name: 'menu.personal.options.dateFormats.language' },
        { id: 'browser', name: 'menu.personal.options.dateFormats.browser' },
      ]}
      onChange={(event) => {
        localStorage.setItem('dateFormat', event.target.value)
      }}
    />
  )
}
