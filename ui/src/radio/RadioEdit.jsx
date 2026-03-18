import {
  DateField,
  Edit,
  required,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { CardMedia } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { urlValidate } from '../utils/validations'
import { Title, ImageUploadOverlay, useImageLoadingState } from '../common'
import subsonic from '../subsonic'

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
})

const RadioTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.radio.name', {
    smart_count: 1,
  })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const RadioEdit = (props) => {
  return (
    <Edit title={<RadioTitle />} {...props}>
      <SimpleForm variant="outlined" {...props}>
        <RadioCoverArt />
        <TextInput source="name" validate={[required()]} />
        <TextInput
          type="url"
          source="streamUrl"
          fullWidth
          validate={[required(), urlValidate]}
        />
        <TextInput
          type="url"
          source="homePageUrl"
          fullWidth
          validate={[urlValidate]}
        />
        <DateField variant="body1" source="updatedAt" showTime />
        <DateField variant="body1" source="createdAt" showTime />
      </SimpleForm>
    </Edit>
  )
}

const RadioCoverArt = ({ record }) => {
  const classes = useStyles()
  const { imageLoading, handleImageLoad, handleImageError } =
    useImageLoadingState(record?.id)

  if (!record) return null

  const imageUrl = subsonic.getCoverArtUrl(record, 300, true)

  return (
    <div className={classes.coverParent}>
      <CardMedia
        component="img"
        src={imageUrl}
        className={`${classes.cover} ${imageLoading ? classes.coverLoading : ''}`}
        onLoad={handleImageLoad}
        onError={handleImageError}
        title={record.name}
      />
      <ImageUploadOverlay
        entityType="radio"
        entityId={record.id}
        hasUploadedImage={!!record.uploadedImage}
      />
    </div>
  )
}

export default RadioEdit
