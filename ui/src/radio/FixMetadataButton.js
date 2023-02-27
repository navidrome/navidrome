import { Button, makeStyles } from '@material-ui/core'
import ErrorOutline from '@material-ui/icons/ErrorOutline'
import { useTranslate } from 'react-admin'

const useStyles = makeStyles(() => ({
  button: {
    textTransform: 'none',
  },
}))

const FixMetadataButton = ({ onFix }) => {
  const styles = useStyles()
  const translate = useTranslate()

  return (
    <Button
      variant="outlined"
      startIcon={<ErrorOutline />}
      size="small"
      className={styles.button}
      onClick={(evt) => {
        evt.stopPropagation()
        evt.preventDefault()
        onFix()
      }}
    >
      {translate('resources.radio.message.noArtist')}
    </Button>
  )
}

export default FixMetadataButton
