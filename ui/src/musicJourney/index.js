/**
 * Music Journey — an intelligent listening session builder on top of the
 * existing Navidrome client.
 *
 * Layer map:
 *
 *   JourneyHome / Creator / NowPlaying / History   (components)
 *        ↓
 *   useJourney / useJourneyWatcher / fetchSongPool (hooks + data)
 *        ↓
 *   journeyEngine: scoring, transitions, phases,   (pure logic)
 *   personalization, nameGenerator
 *        ↓
 *   journeyReducer + actions                       (state)
 *        ↓
 *   existing playerReducer queue actions           (existing player reused)
 *
 * See ./README.md for the full design notes.
 */
export { default as JourneyHome } from './JourneyHome'
export { journeyReducer } from './state/journeyReducer'
export { useJourneyWatcher } from './hooks/useJourneyWatcher'
export {
  buildJourney,
  regenerateRemaining,
  JourneyError,
} from './engine/journeyEngine'
