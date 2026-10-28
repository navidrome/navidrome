import RadioCreate from './RadioCreate'
import RadioEdit from './RadioEdit'
import RadioList from './RadioList'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import RadioIcon from '@mui/icons-material/Radio'
import RadioOutlinedIcon from '@mui/icons-material/RadioOutlined'
import React from 'react'

const all = {
  list: RadioList,
  icon: (
    <DynamicMenuIcon
      path={'radio'}
      icon={RadioOutlinedIcon}
      activeIcon={RadioIcon}
    />
  ),
}

const admin = {
  ...all,
  create: RadioCreate,
  edit: RadioEdit,
}

export default { all, admin }
