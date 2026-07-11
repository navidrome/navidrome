import {
  DateField,
  Edit,
  NumberInput,
  SelectInput,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { CardMedia } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import MicIcon from '@material-ui/icons/Mic'
import { Title, ImageUploadOverlay, useImageLoadingState } from '../common'
import subsonic from '../subsonic'
import config from '../config'

const useStyles = makeStyles({
  coverParent: {
    display: 'inline-flex',
    position: 'relative',
    width: '8rem',
    height: '8rem',
    marginBottom: '1em',
  },
  cover: {
    width: '8rem',
    height: '8rem',
    objectFit: 'cover',
    cursor: 'pointer',
    transition: 'opacity 0.3s ease-in-out',
  },
  coverLoading: {
    opacity: 0.5,
  },
  placeholder: {
    width: '8rem',
    height: '8rem',
  },
})

const PodcastChannelTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.podcastChannel.name', {
    smart_count: 1,
  })
  return <Title subTitle={`${resourceName} ${record ? record.title : ''}`} />
}

const PodcastChannelEdit = (props) => {
  const translate = useTranslate()
  return (
    <Edit title={<PodcastChannelTitle />} {...props}>
      <SimpleForm variant="outlined" {...props}>
        <PodcastChannelCoverArt />
        <TextInput source="url" fullWidth disabled />
        <SelectInput
          source="downloadPolicy"
          choices={[
            {
              id: 'none',
              name: translate('resources.podcastChannel.downloadPolicy.none'),
            },
            {
              id: 'new',
              name: translate('resources.podcastChannel.downloadPolicy.new'),
            },
            {
              id: 'all',
              name: translate('resources.podcastChannel.downloadPolicy.all'),
            },
          ]}
        />
        <NumberInput source="retentionCount" />
        <NumberInput source="retentionDays" />
        <DateField variant="body1" source="lastCheckedAt" showTime />
        <DateField variant="body1" source="updatedAt" showTime />
        <DateField variant="body1" source="createdAt" showTime />
      </SimpleForm>
    </Edit>
  )
}

const PodcastChannelCoverArt = ({ record }) => {
  const classes = useStyles()
  const { imageLoading, handleImageLoad, handleImageError } =
    useImageLoadingState(record?.id)

  if (!record) return null

  return (
    <div className={classes.coverParent}>
      {record.uploadedImage || record.coverArtUrl ? (
        <CardMedia
          component="img"
          src={subsonic.getCoverArtUrl(record, config.uiCoverArtSize, true)}
          className={`${classes.cover} ${imageLoading ? classes.coverLoading : ''}`}
          onLoad={handleImageLoad}
          onError={handleImageError}
          title={record.title}
          alt={record.title}
        />
      ) : (
        <div className={classes.placeholder}>
          <MicIcon fontSize="large" />
        </div>
      )}
      <ImageUploadOverlay
        entityType="podcastChannel"
        entityId={record.id}
        hasUploadedImage={!!record.uploadedImage}
      />
    </div>
  )
}

export default PodcastChannelEdit
