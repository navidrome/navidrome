import { SelectInput, useLocale, useSetLocale, useTranslate } from 'react-admin'
import { useGetLanguageChoices } from '../i18n'
import { HelpMsg } from './HelpMsg'
import { docsUrl, openInNewTab } from '../utils'

const helpKey = '_help'

export const SelectLanguage = (props) => {
  const translate = useTranslate()
  const setLocale = useSetLocale()
  const locale = useLocale()
  const { choices } = useGetLanguageChoices()

  choices.push({
    id: helpKey,
    name: <HelpMsg caption={'Help to translate'} />,
  })

  return (
    <SelectInput
      {...props}
      source="language"
      label={translate('menu.personal.options.language')}
      defaultValue={locale}
      choices={choices}
      translateChoice={false}
      onChange={(event) => {
        if (event.target.value === helpKey) {
          openInNewTab(docsUrl('/docs/developers/translations/'))
          return
        }
        setLocale(event.target.value).then(() => {
          localStorage.setItem('locale', event.target.value)
          document.documentElement.lang = event.target.value
        })
      }}
    />
  )
}
