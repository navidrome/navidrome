import { SelectInput, useTranslate } from 'react-admin'
import { getDefaultViewChoices, getStoredDefaultView } from './defaultViews'

export const SelectDefaultView = (props) => {
  const translate = useTranslate()
  const current = getStoredDefaultView()
  const choices = getDefaultViewChoices(translate)

  return (
    <SelectInput
      {...props}
      source="defaultView"
      label={translate('menu.personal.options.defaultView')}
      defaultValue={current}
      choices={choices}
      translateChoice={false}
      onChange={(event) => {
        localStorage.setItem('defaultView', event.target.value)
      }}
    />
  )
}
