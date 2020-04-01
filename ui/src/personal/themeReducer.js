import { CHANGE_THEME } from './actions'

export default (previousState = 'DarkTheme', { type, payload }) => {
  if (type === CHANGE_THEME) {
    return payload
  }
  return previousState
}
