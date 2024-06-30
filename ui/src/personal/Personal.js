import { SimpleForm, Title, useTranslate } from 'react-admin'
import { Card } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { SelectLanguage } from './SelectLanguage'
import { SelectTheme } from './SelectTheme'
import { SelectDefaultView } from './SelectDefaultView'
import { NotificationsToggle } from './NotificationsToggle'
import { LastfmScrobbleToggle } from './LastfmScrobbleToggle'
import { ListenBrainzScrobbleToggle } from './ListenBrainzScrobbleToggle'
import config from '../config'
import { ReplayGainToggle } from './ReplayGainToggle'
import { SelectPlayerRatingControl } from './SelectPlayerRatingControl'

const useStyles = makeStyles({
  root: { marginTop: '1em' },
})

const Personal = () => {
  const translate = useTranslate()
  const classes = useStyles()

  return (
    <Card className={classes.root}>
      <Title title={'Navidrome - ' + translate('menu.personal.name')} />
      <SimpleForm toolbar={null} variant={'outlined'}>
        <SelectTheme />
        <SelectLanguage />
        <SelectDefaultView />
        {config.enableReplayGain && <ReplayGainToggle />}
        <SelectPlayerRatingControl />
        <NotificationsToggle />
        {config.lastFMEnabled && localStorage.getItem('lastfm-apikey') && (
          <LastfmScrobbleToggle />
        )}
        {config.listenBrainzEnabled && <ListenBrainzScrobbleToggle />}
      </SimpleForm>
    </Card>
  )
}

export default Personal
