import { Search, SearchOutlined } from '@material-ui/icons'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import RadioInfoList from './RadioInfoList'

export default {
  list: RadioInfoList,
  icon: (
    <DynamicMenuIcon
      path="radioInfo"
      icon={SearchOutlined}
      activeIcon={Search}
    />
  ),
}
