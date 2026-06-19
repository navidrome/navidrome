import { ThemeProvider } from '@material-ui/core/styles'
import { createMuiTheme, useVersion } from 'react-admin'

import useCurrentTheme from '../themes/useCurrentTheme'

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
