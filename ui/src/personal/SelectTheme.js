import { SelectInput, useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { AUTO_THEME_ID } from '../consts'
import themes from '../themes'
import { HelpMsg } from './HelpMsg'
import { docsUrl, openInNewTab } from '../utils'
import { changeTheme } from '../actions'

const helpKey = '_help'

export const SelectTheme = (props) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const currentTheme = useSelector((state) => state.theme)
  const themeChoices = [
    {
      id: AUTO_THEME_ID,
      name: 'Auto',
    },
  ]
  themeChoices.push(
    ...Object.keys(themes).map((key) => {
      return { id: key, name: themes[key].themeName }
    }),
  )
  themeChoices.push({
    id: helpKey,
    name: <HelpMsg caption={'Create your own'} />,
  })
  return (
    <SelectInput
      {...props}
      source="theme"
      label={translate('menu.personal.options.theme')}
      defaultValue={currentTheme}
      translateChoice={false}
      choices={themeChoices}
      onChange={(event) => {
        if (event.target.value === helpKey) {
          openInNewTab(docsUrl('/docs/developers/creating-themes/'))
          return
        }
        dispatch(changeTheme(event.target.value))
      }}
    />
  )
}
