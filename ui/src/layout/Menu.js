import React, { useState, createElement } from 'react'
import { useSelector } from 'react-redux'
import { useMediaQuery } from '@material-ui/core'
import { useTranslate, MenuItemLink, getResources } from 'react-admin'
import { withRouter } from 'react-router-dom'
import LibraryMusicIcon from '@material-ui/icons/LibraryMusic'
import ViewListIcon from '@material-ui/icons/ViewList'
import SubMenu from './SubMenu'
import inflection from 'inflection'

const translatedResourceName = (resource, translate) =>
  translate(`resources.${resource.name}.name`, {
    smart_count: 2,
    _:
      resource.options && resource.options.label
        ? translate(resource.options.label, {
            smart_count: 2,
            _: resource.options.label
          })
        : inflection.humanize(inflection.pluralize(resource.name))
  })

const Menu = ({ onMenuClick, dense, logout }) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const open = useSelector((state) => state.admin.ui.sidebarOpen)
  const translate = useTranslate()
  const resources = useSelector(getResources)

  const [state, setState] = useState({
    menuLibrary: true
  })

  const handleToggle = (menu) => {
    setState((state) => ({ ...state, [menu]: !state[menu] }))
  }

  const renderMenuItemLink = (resource) => (
    <MenuItemLink
      key={resource.name}
      to={`/${resource.name}`}
      primaryText={translatedResourceName(resource, translate)}
      leftIcon={
        (resource.icon && createElement(resource.icon)) || <ViewListIcon />
      }
      onClick={onMenuClick}
      sidebarIsOpen={open}
      dense={dense}
    />
  )

  const subItems = (subMenu) => (resource) =>
    resource.hasList && resource.options && resource.options.subMenu === subMenu

  return (
    <div>
      <SubMenu
        handleToggle={() => handleToggle('menuLibrary')}
        isOpen={state.menuLibrary}
        sidebarIsOpen={open}
        name="Library"
        icon={<LibraryMusicIcon />}
        dense={dense}
      >
        {resources.filter(subItems('library')).map(renderMenuItemLink)}
      </SubMenu>
      {resources.filter(subItems(undefined)).map(renderMenuItemLink)}
      {isXsmall && logout}
    </div>
  )
}

export default withRouter(Menu)
