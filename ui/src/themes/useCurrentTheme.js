import { useSelector } from 'react-redux'
import useMediaQuery from '@material-ui/core/useMediaQuery'
import LightTheme from './light'
import DarkTheme from './dark'
import { AUTO_THEME_ID } from '../consts'

export default () => {
  const prefersLightMode = useMediaQuery('(prefers-color-scheme: light)')
  return useSelector((state) => {
    if (state.theme === AUTO_THEME_ID) {
      return prefersLightMode ? LightTheme : DarkTheme
    }
    return state.theme === 'LightTheme' ? LightTheme : DarkTheme
  })
}
