import React, { useCallback } from 'react'
import { useLocation } from 'react-router-dom'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import { LoveButton, useToggleLove } from '../common'
import { keyMap } from '../hotkeys'
import { ThemeProvider } from '@material-ui/styles'
import { createMuiTheme } from '@material-ui/core/styles'
import useCurrentTheme from '../themes/useCurrentTheme'
import config from '../config'

const Placeholder = () => <LoveButton disabled={true} resource={'song'} />

const Toolbar = ({ id }) => {
  const location = useLocation()
  const theme = useCurrentTheme()
  const resource = location.pathname.startsWith('/song') ? 'song' : 'albumSong'
  const { data, loading } = useGetOne(resource, id)
  const [toggleLove, toggling] = useToggleLove(resource, data)

  const handlers = {
    TOGGLE_LOVE: useCallback(() => toggleLove(), [toggleLove]),
  }

  return (
    <ThemeProvider theme={createMuiTheme(theme)}>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      {config.enableFavourites && (
        <LoveButton
          record={data}
          resource={resource}
          disabled={loading || toggling}
        />
      )}
    </ThemeProvider>
  )
}

const PlayerToolbar = ({ id }) =>
  id ? <Toolbar id={id} /> : config.enableFavourites && <Placeholder />

export default PlayerToolbar
