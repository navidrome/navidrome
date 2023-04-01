import * as React from 'react'
import { Typography } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import Inbox from '@material-ui/icons/Inbox'
import {
  CreateButton,
  useTranslate,
  useListContext,
  useResourceContext,
  useGetResourceLabel,
} from 'react-admin'

export const Empty = ({ extra, ...props }) => {
  const { basePath, hasCreate } = useListContext(props)
  const resource = useResourceContext(props)
  const classes = useStyles(props)
  const translate = useTranslate()

  const getResourceLabel = useGetResourceLabel()
  const resourceName = translate(`resources.${resource}.forcedCaseName`, {
    smart_count: 0,
    _: getResourceLabel(resource, 0),
  })

  const emptyMessage = translate('ra.page.empty', { name: resourceName })
  const inviteMessage = translate('ra.page.invite')

  return (
    <>
      <div className={classes.message}>
        <Inbox className={classes.icon} />
        <Typography variant="h4" paragraph>
          {translate(`resources.${resource}.empty`, {
            _: emptyMessage,
          })}
        </Typography>
        {hasCreate && (
          <Typography variant="body1">
            {translate(`resources.${resource}.invite`, {
              _: inviteMessage,
            })}
          </Typography>
        )}
      </div>
      {hasCreate && (
        <div className={classes.toolbar}>
          <CreateButton variant="contained" basePath={basePath} />
          {extra}
        </div>
      )}
    </>
  )
}

const useStyles = makeStyles(
  (theme) => ({
    message: {
      textAlign: 'center',
      opacity: theme.palette.type === 'light' ? 0.5 : 0.8,
      margin: '0 1em',
      color:
        theme.palette.type === 'light' ? 'inherit' : theme.palette.text.primary,
    },
    icon: {
      width: '9em',
      height: '9em',
    },
    toolbar: {
      textAlign: 'center',
      marginTop: '2em',
    },
  }),
  { name: 'RaEmpty' }
)
