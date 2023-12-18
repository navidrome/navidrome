import React, { cloneElement } from 'react'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import {
  ButtonGroup,
  useMediaQuery,
  Typography,
  makeStyles,
} from '@material-ui/core'
import ViewHeadlineIcon from '@material-ui/icons/ViewHeadline'
import ViewModuleIcon from '@material-ui/icons/ViewModule'
import { useDispatch, useSelector } from 'react-redux'
import { albumViewGrid, albumViewTable } from '../actions'
import { ToggleFieldsMenu } from '../common'

const useStyles = makeStyles({
  title: { margin: '1rem' },
  buttonGroup: { width: '100%', justifyContent: 'center' },
  leftButton: { paddingRight: '0.5rem' },
  rightButton: { paddingLeft: '0.5rem' },
})

const AlbumViewToggler = React.forwardRef(
  ({ showTitle = true, disableElevation, fullWidth }, ref) => {
    const dispatch = useDispatch()
    const albumView = useSelector((state) => state.albumView)
    const classes = useStyles()
    const translate = useTranslate()
    return (
      <div ref={ref}>
        {showTitle && (
          <Typography className={classes.title}>
            {translate('ra.toggleFieldsMenu.layout')}
          </Typography>
        )}
        <ButtonGroup
          variant="text"
          color="primary"
          aria-label="text primary button group"
          className={classes.buttonGroup}
        >
          <Button
            size="small"
            className={classes.leftButton}
            label={translate('ra.toggleFieldsMenu.grid')}
            color={albumView.grid ? 'primary' : 'secondary'}
            onClick={() => dispatch(albumViewGrid())}
          >
            <ViewModuleIcon fontSize="inherit" />
          </Button>
          <Button
            size="small"
            className={classes.rightButton}
            label={translate('ra.toggleFieldsMenu.table')}
            color={albumView.grid ? 'secondary' : 'primary'}
            onClick={() => dispatch(albumViewTable())}
          >
            <ViewHeadlineIcon fontSize="inherit" />
          </Button>
        </ButtonGroup>
      </div>
    )
  },
)

const AlbumListActions = ({
  currentSort,
  className,
  resource,
  filters,
  displayedFilters,
  filterValues,
  permanentFilter,
  exporter,
  basePath,
  selectedIds,
  onUnselectItems,
  showFilter,
  maxResults,
  total,
  fullWidth,
  ...rest
}) => {
  const isNotSmall = useMediaQuery((theme) => theme.breakpoints.up('sm'))
  const albumView = useSelector((state) => state.albumView)
  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {filters &&
        cloneElement(filters, {
          resource,
          showFilter,
          displayedFilters,
          filterValues,
          context: 'button',
        })}
      {isNotSmall ? (
        <ToggleFieldsMenu
          resource="album"
          topbarComponent={AlbumViewToggler}
          hideColumns={albumView.grid}
        />
      ) : (
        <AlbumViewToggler showTitle={false} />
      )}
    </TopToolbar>
  )
}

AlbumListActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null,
}

export default AlbumListActions
