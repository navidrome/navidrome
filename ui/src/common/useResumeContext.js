import { useSelector } from 'react-redux'
import config from '../config'

const useResumeContext = () => {
  const { context } = useSelector((state) => state.replayGain || {})

  if (config.enableReplayGain) {
    return () => {
      if (context && context.state !== 'running') {
        context.resume()
      }
    }
  } else {
    return () => {}
  }
}

export default useResumeContext
