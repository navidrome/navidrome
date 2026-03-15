import { CHANGE_THEME } from '../actions'
import { AUTO_THEME_ID, AUTO_THEME_CONFIG_VALUE } from '../consts'
import config from '../config'
import themes from '../themes'

const defaultTheme = () => {
  if (config.defaultTheme === AUTO_THEME_CONFIG_VALUE) {
    return AUTO_THEME_ID
  }
  return (
    Object.keys(themes).find(
      (t) => themes[t].themeName === config.defaultTheme,
    ) || 'DarkTheme'
  )
}

export const themeReducer = (
  previousState = defaultTheme(),
  { type, payload },
) => {
  if (type === CHANGE_THEME) {
    return payload
  }
  return previousState
}
