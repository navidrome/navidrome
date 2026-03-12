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
    const version = useVersion()

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
