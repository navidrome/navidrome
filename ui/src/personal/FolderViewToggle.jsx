import { useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { setShowFolderView } from '../actions'
import { FormControl, FormControlLabel, Switch } from '@material-ui/core'

export const FolderViewToggle = () => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const showFolderView = useSelector((state) => state.settings.showFolderView)

  const toggleFolderView = (event) => {
    dispatch(setShowFolderView(event.target.checked))
  }

  return (
    <FormControl>
      <FormControlLabel
        control={
          <Switch
            id={'showFolderView'}
            color="primary"
            checked={showFolderView !== false}
            onChange={toggleFolderView}
          />
        }
        label={<span>{translate('menu.personal.options.showFolderView')}</span>}
      />
    </FormControl>
  )
}
