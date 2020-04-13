import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Card } from '@material-ui/core'
import { Title, SimpleForm, SelectInput, useTranslate } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { changeTheme } from './actions'
import themes from '../themes'

const useStyles = makeStyles({
  root: { marginTop: '1em' }
})

const Personal = () => {
  const translate = useTranslate()
  const classes = useStyles()
  const currentTheme = useSelector((state) => state.theme)
  const dispatch = useDispatch()
  const themeChoices = Object.keys(themes).map((key) => {
    return { id: key, name: themes[key].themeName }
  })

  return (
    <Card className={classes.root}>
      <Title title={'Navidrome - ' + translate('menu.personal.name')} />
      <SimpleForm toolbar={null}>
        <SelectInput
          source="theme"
          label={translate('menu.personal.options.theme')}
          defaultValue={currentTheme}
          choices={themeChoices}
          onChange={(event) => {
            dispatch(changeTheme(event.target.value))
          }}
        />
      </SimpleForm>
    </Card>
  )
}

export default Personal
