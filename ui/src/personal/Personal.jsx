import { SimpleForm, Title, useTranslate, usePermissions } from 'react-admin'
import { Card, Typography, Box } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { SelectLanguage } from './SelectLanguage'
import { SelectTheme } from './SelectTheme'
import { SelectDefaultView } from './SelectDefaultView'
import { NotificationsToggle } from './NotificationsToggle'
import { LastfmScrobbleToggle } from './LastfmScrobbleToggle'
import { ListenBrainzScrobbleToggle } from './ListenBrainzScrobbleToggle'
import config from '../config'
import { ReplayGainToggle } from './ReplayGainToggle'
import { Link } from 'react-router-dom'

const useStyles = makeStyles({
  root: { marginTop: '1em' },
  adminLinks: { marginTop: '1em' },
})

const Personal = () => {
  const translate = useTranslate()
  const classes = useStyles()
  const { permissions } = usePermissions()
  const isAdmin = permissions === 'admin'

  return (
    <Card className={classes.root}>
      <Title title={'Navidrome - ' + translate('menu.personal.name')} />
      <SimpleForm toolbar={null} variant={'outlined'}>
        <SelectTheme />
        <SelectLanguage />
        <SelectDefaultView />
        {config.enableReplayGain && <ReplayGainToggle />}
        <NotificationsToggle />
        {config.lastFMEnabled && <LastfmScrobbleToggle />}
        {config.listenBrainzEnabled && <ListenBrainzScrobbleToggle />}
      </SimpleForm>
      
      {isAdmin && (
        <Box p={2} className={classes.adminLinks}>
          <Typography variant="h6" gutterBottom>
            {translate('menu.personal.admin')}
          </Typography>
          <Link to="/personal/logs">
            <Typography color="primary">
              {translate('menu.personal.logs')}
            </Typography>
          </Link>
        </Box>
      )}
    </Card>
  )
}

export default Personal
