import React, { useCallback } from 'react'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import { AddToPlaylistButton, LoveButton, useToggleLove } from '../common'
import { keyMap } from '../hotkeys'
import { makeStyles } from '@material-ui/styles'

const useStyles = makeStyles({
  flexRow: {
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'nowrap',
    gap: '0.5em',
  },
})

const Placeholder = () => {
  const styles = useStyles()
  return (
    <div className={styles.flexRow}>
      <AddToPlaylistButton selectedIds={[]} disabled compact />
      <LoveButton disabled={true} resource={'song'} />
    </div>
  )
}

const Toolbar = ({ id }) => {
  const { data, loading } = useGetOne('song', id)
  const [toggleLove, toggling] = useToggleLove('song', data)
  const styles = useStyles()

  const handlers = {
    TOGGLE_LOVE: useCallback(() => toggleLove(), [toggleLove]),
  }

  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      <div className={styles.flexRow}>
        <AddToPlaylistButton selectedIds={[id]} compact />
        <LoveButton
          record={data}
          resource={'song'}
          disabled={loading || toggling}
        />
      </div>
    </>
  )
}

const PlayerToolbar = ({ id, isRadio }) =>
  id && !isRadio ? <Toolbar id={id} /> : <Placeholder />

export default PlayerToolbar
