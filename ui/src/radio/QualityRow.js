import { Box } from '@material-ui/core'
import { NumberInput, TextInput } from 'react-admin'

export const QualityRow = ({ basePath, ...props }) => (
  <Box display="flex" width="100% !important">
    <Box flex={1} mr="0.5em">
      <TextInput {...props} source="codec" fullWidth variant="outlined" />
    </Box>
    <Box flex={1} mr="0.5em">
      <NumberInput
        {...props}
        min={0}
        source="bitrate"
        fullWidth
        variant="outlined"
      />
    </Box>
  </Box>
)
