import RecommendationList from './RecommendationList'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import ThumbUpIcon from '@material-ui/icons/ThumbUp'
import ThumbUpOutlinedIcon from '@material-ui/icons/ThumbUpOutlined'

const all = {
  list: RecommendationList,
  icon: (
    <DynamicMenuIcon
      path={'recommendation'}
      icon={ThumbUpOutlinedIcon}
      activeIcon={ThumbUpIcon}
    />
  ),
}

export default { all }
