import React, { useCallback } from 'react'
import { useGetOne } from 'react-admin'
import { GlobalHotKeys } from 'react-hotkeys'
import { LoveButton, RatingField, useToggleLove } from '../common'
import { keyMap } from '../hotkeys'
import { useSelector } from 'react-redux'

const Placeholder = () => {
  const playerRatingControl = useSelector(
    (state) => state.settings.playerRatingControl,
  )

  switch (playerRatingControl) {
    case 'love':
      return <LoveButton disabled={true} resource={'song'} />
    case 'rating':
      return (
        <RatingField
          disabled={true}
          source={'rating'}
          resource={'song'}
          size={'small'}
        />
      )
    default:
      return null
  }
}

const Toolbar = ({ id }) => {
  const playerRatingControl = useSelector(
    (state) => state.settings.playerRatingControl,
  )
  const { data, loading } = useGetOne('song', id)
  const [toggleLove, toggling] = useToggleLove('song', data)

  const handlers = {
    TOGGLE_LOVE: useCallback(() => toggleLove(), [toggleLove]),
  }

  switch (playerRatingControl) {
    case 'love':
      return (
        <>
          <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
          <LoveButton
            record={data}
            resource={'song'}
            disabled={loading || toggling}
          />
        </>
      )
    case 'rating':
      return (
        <RatingField
          record={data}
          source={'rating'}
          resource={'song'}
          size={'small'}
          disabled={loading || toggling}
        />
      )
    default:
      return null
  }
}

const PlayerToolbar = ({ id, isRadio }) =>
  id && !isRadio ? <Toolbar id={id} /> : <Placeholder />

export default PlayerToolbar
