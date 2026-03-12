import { useEffect } from 'react'
import { ThemeProvider, makeStyles } from '@material-ui/core/styles'
import {
  createMuiTheme,
  useRefresh,
  useSetLocale,
  useVersion,
} from 'react-admin'

import useCurrentTheme from '../themes/useCurrentTheme'
import config from '../config'
import { retrieveTranslation } from '../i18n'

const withTheme = (Component) => {
  const WithTheme = (props) => {
    const theme = useCurrentTheme()
    const setLocale = useSetLocale()
    const refresh = useRefresh()
    const version = useVersion()

    useEffect(() => {
      if (config.defaultLanguage !== '' && !localStorage.getItem('locale')) {
        retrieveTranslation(config.defaultLanguage)
          .then(() => {
            setLocale(config.defaultLanguage).then(() => {
              localStorage.setItem('locale', config.defaultLanguage)
            })
            refresh(true)
          })
          .catch((e) => {
            throw new Error(
              'Cannot load language "' + config.defaultLanguage + '": ' + e,
            )
          })
      }
    }, [refresh, setLocale])

    return (
      <ThemeProvider theme={createMuiTheme(theme)}>
        <Component key={version} {...props} />
      </ThemeProvider>
    )
  }

  WithTheme.displayName = `WithTheme(${Component.displayName ?? Component.name ?? 'Component'})`

  return WithTheme
}

export default withTheme
