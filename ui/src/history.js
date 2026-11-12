import { createHashHistory } from './utils/hashHistory'

// The app's single router history. Navigate with history.push() rather than
// assigning window.location.href/hash: on insecure origins Safari reloads the
// page for those (see utils/hashHistory.js).
const history = createHashHistory()

export default history
