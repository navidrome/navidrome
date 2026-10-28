import { SimpleForm, Title, useTranslate } from 'react-admin'
import { Card } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { SelectLanguage } from './SelectLanguage'
import { SelectDateFormat } from './SelectDateFormat'
import { SelectTheme } from './SelectTheme'
import { SelectDefaultView } from './SelectDefaultView'
import { NotificationsToggle } from './NotificationsToggle'
import { LastfmScrobbleToggle } from './LastfmScrobbleToggle'
import { ListenBrainzScrobbleToggle } from './ListenBrainzScrobbleToggle'
import config from '../config'
import { ReplayGainToggle } from './ReplayGainToggle'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles({
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
        <SelectDateFormat />
        <SelectDefaultView />
        {config.enableReplayGain && <ReplayGainToggle />}
        <NotificationsToggle />
        {config.lastFMEnabled && <LastfmScrobbleToggle />}
        {config.listenBrainzEnabled && <ListenBrainzScrobbleToggle />}
      </SimpleForm>
    </Card>
  )
}

export default Personal
