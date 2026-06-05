import { createDecisionService } from './decisionService'
import { fetchTranscodeDecision } from './fetchDecision'
export { detectBrowserProfile } from './browserProfile'

export const decisionService = createDecisionService(fetchTranscodeDecision)
