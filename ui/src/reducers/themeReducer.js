import { CHANGE_THEME } from '../actions'
import config from '../config'
import themes from '../themes'

const defaultTheme = () => {
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
