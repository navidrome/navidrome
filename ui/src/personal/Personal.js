import { SimpleForm, Title, useTranslate } from 'react-admin'
import { Card } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { SelectLanguage } from './SelectLanguage'
import { SelectTheme } from './SelectTheme'
import { SelectDefaultView } from './SelectDefaultView'
import { NotificationsToggle } from './NotificationsToggle'
import { LastfmScrobbleToggle } from './LastfmScrobbleToggle'
import config from '../config'

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
        <NotificationsToggle />
        {config.devEnableScrobble && <LastfmScrobbleToggle />}
      </SimpleForm>
    </Card>
  )
}

export default Personal
