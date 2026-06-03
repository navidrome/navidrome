# Navidrome Physical Folder Browsing - Roadmap

This document outlines the planned enhancements for the Physical Folder Browsing feature, moving from core functionality to a polished, high-end user experience.

---

## Phase 2: Visual Polish & UX Improvements

### 1. Folder Thumbnails & Grid View
*   **Composite Icons**: Automatically generate a folder thumbnail using the artwork from the first 4 albums found within that hierarchy.
*   **Grid View Toggle**: Implement a "Grid View" for folders (similar to the Album list) with large cards and thumbnails.
*   **Navigation & Pagination**: Fixed breadcrumb routing and enabled 500+ item pagination across all folder levels.
*   **Status**: ✅ Completed

### 2. "Show in Folder" Integration
*   **Context Action**: Add a "Show in Folder" option to the "More" menu for any Song or Album.
*   **Navigation**: Clicking the action should jump the user directly to the exact location of that item in the physical folder browser.
*   **Status**: ✅ Completed

---

## Phase 3: Advanced Data & Actions

### 3. Metadata & Folder Statistics
*   **Counts**: Display the number of Subfolders and Songs directly in the Folder list.
*   **Storage Info**: Show the total physical disk size (e.g., "1.4 GB") of the folder.
*   **Duration**: Calculate and show the total play time for the entire folder hierarchy.
*   **Status**: ✅ Completed

### 4. Folder ZIP Downloads
*   **Bulk Export**: Add a "Download Folder (ZIP)" action to the toolbar and context menus.
*   **Implementation**: Leverage Navidrome's existing archiving logic to package folders on-the-fly.
*   **Status**: ✅ Completed

---

## Phase 4: Search & Automation

### 5. Scoped Search (Search within Folder)
*   **Local Filter**: Add a search bar inside the `FolderShow` view that only returns results from that specific folder and its children.
*   **Status**: ✅ Completed

### 6. "Folder as Playlist" Sync
*   **Dynamic Playlists**: Allow users to "pin" a physical folder as a Navidrome playlist.
*   **Auto-Update**: Any files added to the physical folder on disk should automatically appear in the virtual playlist.
*   **Status**: ✅ Completed

---

## Completed Features (Phase 1)
*   ✅ Hierarchical Folder Navigation
*   ✅ Automatic Path Normalization (Hash Matching)
*   ✅ Breadcrumb Navigation
*   ✅ Recursive Play All / Shuffle
*   ✅ Recursive Add to Playlist
*   ✅ Full Song Toolbar inside Folders
*   ✅ Folder Context Menus (More Actions)
*   ✅ Visual Refinement: Hidden empty sections and standard Album-style thumbnails
