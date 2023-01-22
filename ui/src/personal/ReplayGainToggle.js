import { NumberInput, SelectInput, useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { changeGain, changePreamp } from '../actions'

export const ReplayGainToggle = (props) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const gainInfo = useSelector((state) => state.replayGain)

  return (
    <>
      <SelectInput
        {...props}
        fullWidth
        source="replayGain"
        label={translate('menu.personal.options.replaygain')}
        choices={[
          { id: 'none', name: 'menu.personal.options.gain.none' },
          { id: 'album', name: 'menu.personal.options.gain.album' },
          { id: 'track', name: 'menu.personal.options.gain.track' },
        ]}
        defaultValue={gainInfo.gainMode}
        onChange={(event) => {
          dispatch(changeGain(event.target.value))
        }}
      />
      <br />
      {gainInfo.gainMode !== 'none' && (
        <NumberInput
          {...props}
          source="preAmp"
          label={translate('menu.personal.options.preAmp')}
          defaultValue={gainInfo.preAmp}
          step={0.5}
          min={-15}
          max={15}
          onChange={(event) => {
            dispatch(changePreamp(event.target.value))
          }}
        />
      )}
    </>
  )
}
