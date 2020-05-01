import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Card } from '@material-ui/core'
import {
  SelectInput,
  SimpleForm,
  Title,
  useGetList,
  useLocale,
  useSetLocale,
  useTranslate,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import HelpOutlineIcon from '@material-ui/icons/HelpOutline'
import { changeTheme } from './actions'
import themes from '../themes'
import { docsUrl } from '../utils/docsUrl'

const useStyles = makeStyles({
  root: { marginTop: '1em' },
})

const helpKey = '_help'

function openInNewTab(url) {
  const win = window.open(url, '_blank')
  win.focus()
}

const HelpMsg = ({ caption }) => (
  <>
    <HelpOutlineIcon />
    &nbsp;&nbsp; {caption}
  </>
)

const SelectLanguage = (props) => {
  const { ids, data, loaded } = useGetList(
    'translation',
    { page: 1, perPage: -1 },
    { field: '', order: '' },
    {}
  )

  const translate = useTranslate()
  const setLocale = useSetLocale()
  const locale = useLocale()

  const langChoices = [{ id: 'en', name: 'English' }]
  if (loaded) {
    ids.forEach((id) => langChoices.push({ id: id, name: data[id].name }))
  }
  langChoices.sort((a, b) => a.name.localeCompare(b.name))
  langChoices.push({
    id: helpKey,
    name: <HelpMsg caption={'Help to translate'} />,
  })

  return (
    <SelectInput
      {...props}
      source="language"
      label={translate('menu.personal.options.language')}
      defaultValue={locale}
      choices={langChoices}
      translateChoice={false}
      onChange={(event) => {
        if (event.target.value === helpKey) {
          openInNewTab(docsUrl('/docs/developers/translations/'))
          return
        }
        setLocale(event.target.value)
        localStorage.setItem('locale', event.target.value)
      }}
    />
  )
}

const SelectTheme = (props) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const currentTheme = useSelector((state) => state.theme)
  const themeChoices = Object.keys(themes).map((key) => {
    return { id: key, name: themes[key].themeName }
  })
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

const Personal = () => {
  const translate = useTranslate()
  const classes = useStyles()

  return (
    <Card className={classes.root}>
      <Title title={'Navidrome - ' + translate('menu.personal.name')} />
      <SimpleForm toolbar={null}>
        <SelectTheme />
        <SelectLanguage />
      </SimpleForm>
    </Card>
  )
}

export default Personal
