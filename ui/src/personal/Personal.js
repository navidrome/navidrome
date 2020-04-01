import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Card, CardContent, MenuItem, Select } from '@material-ui/core'
import { Title, useTranslate } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { changeTheme } from './actions'
import themes from '../themes'

const useStyles = makeStyles({
  label: { width: '10em', display: 'inline-block' },
  select: { minWidth: 200 }
})

const Personal = () => {
  const translate = useTranslate()
  const classes = useStyles()
  const theme = useSelector((state) => state.theme)
  const dispatch = useDispatch()
  const themeNames = Object.keys(themes).sort()

  return (
    <Card>
      <Title title={'Navidrome - ' + translate('menu.personal')} />
      <CardContent>
        <div className={classes.label}>{translate('menu.theme')}</div>
        <Select
          className={classes.select}
          value={theme}
          variant="filled"
          onChange={(event) => {
            dispatch(changeTheme(event.target.value))
          }}
        >
          {themeNames.map((t) => (
            <MenuItem value={t}>{themes[t].themeName}</MenuItem>
          ))}
        </Select>
      </CardContent>
    </Card>
  )
}

export default Personal
