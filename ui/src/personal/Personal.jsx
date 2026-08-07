import { SimpleForm, Title, useTranslate } from 'react-admin'
import { Card } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { SelectLanguage } from './SelectLanguage'
import { SelectTheme } from './SelectTheme'
import { SelectDefaultView } from './SelectDefaultView'
import { NotificationsToggle } from './NotificationsToggle'
import { FolderViewToggle } from './FolderViewToggle'
import { PodcastsToggle } from './PodcastsToggle'
import { ViewToggle } from './ViewToggle'
import { LastfmScrobbleToggle } from './LastfmScrobbleToggle'
import { ListenBrainzScrobbleToggle } from './ListenBrainzScrobbleToggle'
import config from '../config'
import { ReplayGainToggle } from './ReplayGainToggle'
import { getFeaturePermissions } from '../authProvider'

const useStyles = makeStyles({
  root: { marginTop: '1em' },
})

const Personal = () => {
  const translate = useTranslate()
  const classes = useStyles()
  // Admin-gated fork feature access - a toggle for a feature the admin has revoked shouldn't be
  // offered at all, not just default off (there'd be nothing for it to control). Genre and
  // Podcasts have no admin gate yet, so their toggles are unconditional.
  const featurePermissions = getFeaturePermissions()

  return (
    <Card className={classes.root}>
      <Title title={'Navidrome - ' + translate('menu.personal.name')} />
      <SimpleForm toolbar={null} variant={'outlined'}>
        <SelectTheme />
        <SelectLanguage />
        <SelectDefaultView />
        {config.enableReplayGain && <ReplayGainToggle />}
        <NotificationsToggle />
        {featurePermissions.folders !== false && <FolderViewToggle />}
        <PodcastsToggle />
        <ViewToggle
          settingsKey="showGenreView"
          labelKey="menu.personal.options.showGenreView"
        />
        {featurePermissions.ai_tags !== false && (
          <ViewToggle
            settingsKey="showAiGenreView"
            labelKey="menu.personal.options.showAiGenreView"
          />
        )}
        {featurePermissions.ai_tags !== false && (
          <ViewToggle
            settingsKey="showAiMoodView"
            labelKey="menu.personal.options.showAiMoodView"
          />
        )}
        {featurePermissions.my_tags !== false && (
          <ViewToggle
            settingsKey="showMyTagsView"
            labelKey="menu.personal.options.showMyTagsView"
          />
        )}
        {config.lastFMEnabled && <LastfmScrobbleToggle />}
        {config.listenBrainzEnabled && <ListenBrainzScrobbleToggle />}
      </SimpleForm>
    </Card>
  )
}

export default Personal
