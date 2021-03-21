import { useSelector } from 'react-redux'
import useMediaQuery from '@material-ui/core/useMediaQuery'
import themes from './index'
import { AUTO_THEME_ID } from '../consts'

export default () => {
  const prefersLightMode = useMediaQuery('(prefers-color-scheme: light)')
  return useSelector((state) => {
    if (state.theme === AUTO_THEME_ID) {
      return prefersLightMode ? themes.LightTheme : themes.DarkTheme
    }
    return themes[state.theme] || themes.DarkTheme
  })
}
