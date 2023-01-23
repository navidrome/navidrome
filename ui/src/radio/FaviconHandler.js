import { debounce } from '@material-ui/core'
import PropTypes from 'prop-types'
import { ImageField, Loading, TextInput } from 'react-admin'

export const FaviconHandler = ({
  favicon,
  loading,
  setFavicon,
  setLoading,
  ...props
}) => {
  const iconExists = (url) => {
    if (!url) return

    return new Promise((resolve) => {
      const img = new Image()
      img.onload = function () {
        setFavicon(url)
        setLoading(false)
        resolve()
      }
      img.onerror = function () {
        setFavicon(undefined)
        setLoading(false)
        resolve('ra.page.not_found')
      }
      img.src = url
    })
  }

  const throttledIconExists = debounce(iconExists, 300)
  const markLoading = () => setLoading(true)

  return (
    <>
      <TextInput
        {...props}
        type="url"
        source="favicon"
        fullWidth
        validate={[throttledIconExists]}
        onChange={markLoading}
      />
      {favicon &&
        (loading ? (
          <Loading />
        ) : (
          <ImageField
            {...props}
            record={{ favicon }}
            source="favicon"
            title="favicon"
          />
        ))}
    </>
  )
}

FaviconHandler.propTypes = {
  record: PropTypes.object.isRequired,
  fullWidth: PropTypes.bool,
}

FaviconHandler.defaultProps = {
  record: {},
  fullWidth: true,
}
