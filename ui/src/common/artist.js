// Size of an artist's downloadable album-artist content, or undefined when there
// is nothing to download (a missing artist, or no album-artist songs). Download
// and Share only cover album-artist songs, so callers gate on this, not the
// role-inclusive total.
export const artistDownloadSize = (record) =>
  record?.missing ? undefined : record?.stats?.albumartist?.size
