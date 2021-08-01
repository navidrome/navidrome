import { makeStyles } from '@material-ui/core'
import { List } from '../common/List'

const useStyles = makeStyles((theme) => ({
  root: {
    display: 'flex',
    flexDirection: 'column',
    height: '100%',
  },
  main: {
    flexGrow: 1,
  },
}))

function InfiniteList({ pagination, children, ...rest }) {
  const classes = useStyles()

  return (
    <List
      pagination={null}
      classes={{
        root: classes.root,
        main: classes.main,
      }}
      {...rest}
    >
      {children}
    </List>
  )
}

export default InfiniteList
