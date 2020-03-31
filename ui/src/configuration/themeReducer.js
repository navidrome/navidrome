import { CHANGE_THEME } from './actions'

export default (previousState = 'dark', { type, payload }) => {
  if (type === CHANGE_THEME) {
    return payload
  }
  return previousState
}
