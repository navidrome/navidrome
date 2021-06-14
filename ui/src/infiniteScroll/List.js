import { List } from 'react-admin'

function InfiniteList({ pagination, children, ...rest }) {
  return (
    <List pagination={null} {...rest}>
      {children}
    </List>
  )
}

export default InfiniteList
