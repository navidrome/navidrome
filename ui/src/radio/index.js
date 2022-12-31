import RadioIcon from '@material-ui/icons/Radio'
import RadioCreate from './RadioCreate'
import RadioEdit from './RadioEdit'
import RadioList from './RadioList'
import RadioShow from './RadioShow'

const all = {
  list: RadioList,
  icon: <RadioIcon />,
  show: RadioShow,
}

const admin = {
  ...all,
  create: RadioCreate,
  edit: RadioEdit,
}

export default { all, admin }
