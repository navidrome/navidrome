import { createDecisionService } from './decisionService'
import subsonic from '../subsonic'
export { detectBrowserProfile } from './browserProfile'

export const decisionService = createDecisionService(
  subsonic.getTranscodeDecision,
)
