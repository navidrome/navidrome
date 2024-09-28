window.global = window // fix "global is not defined" error in react-image-lightbox

import ReactDOM from 'react-dom'
import './index.css'
import App from './App'

ReactDOM.render(<App />, document.getElementById('root'))
