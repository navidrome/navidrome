import { AddToPlaylistDialog } from './AddToPlaylistDialog'
import DownloadMenuDialog from './DownloadMenuDialog'
import { HelpDialog } from './HelpDialog'
import { ShareDialog } from './ShareDialog'
import { SaveQueueDialog } from './SaveQueueDialog'
import { TagSongDialog } from './TagSongDialog'

export const Dialogs = (props) => (
  <>
    <AddToPlaylistDialog />
    <TagSongDialog />
    <SaveQueueDialog />
    <DownloadMenuDialog />
    <HelpDialog />
    <ShareDialog />
  </>
)
