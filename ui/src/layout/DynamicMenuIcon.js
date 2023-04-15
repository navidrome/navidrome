import PropTypes from 'prop-types'
import { useLocation } from 'react-router-dom'
import { createElement } from 'react'

const DynamicMenuIcon = ({ icon, activeIcon, path }) => {
  const location = useLocation()

  if (!activeIcon) {
    return createElement(icon, { 'data-testid': 'icon' })
  }

  return location.pathname.startsWith('/' + path)
    ? createElement(activeIcon, { 'data-testid': 'activeIcon' })
    : createElement(icon, { 'data-testid': 'icon' })
}

DynamicMenuIcon.propTypes = {
  path: PropTypes.string.isRequired,
  icon: PropTypes.object.isRequired,
  activeIcon: PropTypes.object,
}

export default DynamicMenuIcon
