// Adapted from Empty.tsx from react-admin

import { Typography } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import Inbox from '@material-ui/icons/Inbox'
import { useTranslate, useResourceContext, useGetResourceLabel } from 'ra-core'

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
  }),
  { name: 'EmptyExpected' },
)

export const EmptyExpected = (props) => {
  const resource = useResourceContext(props)
  const classes = useStyles(props)
  const translate = useTranslate()

  const getResourceLabel = useGetResourceLabel()
  const resourceName = translate(`resources.${resource}.forcedCaseName`, {
    smart_count: 0,
    _: getResourceLabel(resource, 0),
  })

  const emptyMessage = translate('ra.page.emptyExpected', {
    name: resourceName,
  })

  return (
    <>
      <div className={classes.message}>
        <Inbox className={classes.icon} />
        <Typography variant="h4" paragraph>
          {translate(`resources.${resource}.empty`, {
            _: emptyMessage,
          })}
        </Typography>
      </div>
    </>
  )
}
