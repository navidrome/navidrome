import { CHANGE_THEME } from '../actions'

export const themeReducer = (
  previousState = 'DarkTheme',
  { type, payload }
) => {
  if (type === CHANGE_THEME) {
    return payload
  }
  return previousState
}
