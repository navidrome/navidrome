import GetAppIcon from '@material-ui/icons/GetApp'
import { CreateButton } from 'react-admin'

export const ImportButton = (props) => (
  <CreateButton
    {...props}
    icon={<GetAppIcon />}
    basePath="externalPlaylist"
    label={'ra.action.import'}
  />
)
