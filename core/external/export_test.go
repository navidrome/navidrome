package external

func QueuedAlbumRefreshes(p Provider) int  { return len(p.(*provider).albumQueue) }
func QueuedArtistRefreshes(p Provider) int { return len(p.(*provider).artistQueue) }
