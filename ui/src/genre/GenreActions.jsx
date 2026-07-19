import PropTypes from 'prop-types'
import { TopToolbar, sanitizeListRestProps } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import { ShuffleAllButton } from '../common'
import { CreatePlaylistFromGenreButton } from './CreatePlaylistFromGenreButton'

const useStyles = makeStyles({
  toolbar: {
    minHeight: 'auto',
    padding: '0 !important',
    background: 'transparent',
    boxShadow: 'none',
    '& .MuiToolbar-root': {
      minHeight: 'auto',
      padding: '0 !important',
      background: 'transparent',
    },
  },
})

const GenreActions = ({ className, record, ...rest }) => {
  const classes = useStyles()
  if (!record) return null

  return (
    <TopToolbar
      className={`${className} ${classes.toolbar}`}
      {...sanitizeListRestProps(rest)}
    >
      <ShuffleAllButton filters={{ genre_id: record.id }} />
      <CreatePlaylistFromGenreButton record={record} />
    </TopToolbar>
  )
}

GenreActions.propTypes = {
  className: PropTypes.string,
  record: PropTypes.object,
}

GenreActions.defaultProps = {
  className: '',
}

export default GenreActions
