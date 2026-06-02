import React from 'react'
import {
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
  Button,
} from 'react-admin'
import {
  ButtonGroup,
  makeStyles,
} from '@material-ui/core'
import ViewHeadlineIcon from '@material-ui/icons/ViewHeadline'
import ViewModuleIcon from '@material-ui/icons/ViewModule'
import { useDispatch, useSelector } from 'react-redux'
import { folderViewGrid, folderViewTable } from '../actions'

const useStyles = makeStyles({
  buttonGroup: { marginLeft: '1rem' },
})

const FolderListActions = (props) => {
  const {
    className,
    ...rest
  } = props
  const dispatch = useDispatch()
  const folderView = useSelector((state) => state.folderView)
  const classes = useStyles()
  const translate = useTranslate()

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <ButtonGroup
        variant="text"
        color="primary"
        className={classes.buttonGroup}
      >
        <Button
          size="small"
          label={translate('ra.toggleFieldsMenu.grid')}
          color={folderView.grid ? 'primary' : 'secondary'}
          onClick={() => dispatch(folderViewGrid())}
        >
          <ViewModuleIcon />
        </Button>
        <Button
          size="small"
          label={translate('ra.toggleFieldsMenu.table')}
          color={folderView.grid ? 'secondary' : 'primary'}
          onClick={() => dispatch(folderViewTable())}
        >
          <ViewHeadlineIcon />
        </Button>
      </ButtonGroup>
    </TopToolbar>
  )
}

export default FolderListActions
