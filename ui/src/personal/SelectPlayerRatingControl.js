import { SelectInput, useTranslate } from 'react-admin'
import { setPlayerRatingControl } from '../actions/settings'
import { useDispatch, useSelector } from 'react-redux'

export const SelectPlayerRatingControl = (props) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const current = useSelector((state) => state.settings.playerRatingControl)
  const choices = [
    {
      id: 'none',
      name: translate(`menu.personal.options.playerRatingControls.none`),
    },
    {
      id: 'love',
      name: translate(`menu.personal.options.playerRatingControls.love`),
    },
    {
      id: 'rating',
      name: translate(`menu.personal.options.playerRatingControls.rating`),
    },
  ]

  return (
    <SelectInput
      {...props}
      source="playerRatingControl"
      label={translate('menu.personal.options.playerRatingControl')}
      defaultValue={current}
      choices={choices}
      translateChoice={false}
      onChange={(event) => {
        dispatch(setPlayerRatingControl(event.target.value))
      }}
    />
  )
}
