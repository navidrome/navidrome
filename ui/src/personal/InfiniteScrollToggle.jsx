import { useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { setInfiniteScroll } from '../actions'
import { FormControl, FormControlLabel, Switch } from '@material-ui/core'

export const InfiniteScrollToggle = () => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const currentSetting = useSelector(
    (state) => state.infiniteScroll?.enabled ?? false,
  )

  const toggleInfiniteScroll = (event) => {
    dispatch(setInfiniteScroll(event.target.checked))
  }

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={'infinite-scroll'}
            color="primary"
            checked={currentSetting}
            onChange={toggleInfiniteScroll}
          />
        }
        label={
          <span>{translate('menu.personal.options.infinite_scroll')}</span>
        }
      />
    </FormControl>
  )
}
