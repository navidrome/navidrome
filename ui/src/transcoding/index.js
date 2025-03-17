import { MdTransform } from 'react-icons/md'
import TranscodingList from './TranscodingList'
import TranscodingEdit from './TranscodingEdit'
import TranscodingCreate from './TranscodingCreate'
import TranscodingShow from './TranscodingShow'
import config from '../config'

export default {
  list: TranscodingList,
  edit: config.enableTranscodingConfig && TranscodingEdit,
  create: config.enableTranscodingConfig && TranscodingCreate,
  show: !config.enableTranscodingConfig && TranscodingShow,
  icon: MdTransform,
}
