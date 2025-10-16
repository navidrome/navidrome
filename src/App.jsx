// src/App.jsx
import React, { useState, useEffect, useCallback } from 'react';
import { invoke } from '@tauri-apps/api/tauri';
import { listen } from '@tauri-apps/api/event';
import { 
    callJukebox, 
    coverArtUrl, 
    addRandomSong, 
    getConfig 
} from './jukeboxApi';

// --- INITIAL STATE ---
const initialState = {
    playlist: [],
    currentIndex: 0,
    playing: false,
    gain: 1,
    position: 0,
    // ... other state properties
};

function App() {
    const [state, setState] = useState(initialState);
    const [statusText, setStatusText] = useState('Initializing...');
    const config = getConfig(); // Load current config

    // --- Media Control Native Interop ---

    const updateNativeMediaInfo = useCallback((track) => {
        if (!track) {
            // Clear media info
            invoke('update_media_info', { title: '', artist: '', coverUrl: '' });
            return;
        }
        
        // Call the Rust command to update the macOS Control Center
        invoke('update_media_info', {
            title: track.title,
            artist: track.artist,
            coverUrl: coverArtUrl(track.coverArt, 512)
        }).catch(e => console.error("Failed to update macOS media info:", e));
    }, []);


    // --- Jukebox Core Logic ---

    const refreshState = useCallback(async () => {
        try {
            const { status, playlist } = await callJukebox('get');
            const newState = {
                ...initialState, 
                ...status, 
                ...playlist
            };
            
            // Check if track changed to update native media info
            const currentTrack = newState.playlist[newState.currentIndex];
            if (currentTrack?.id !== state.playlist[state.currentIndex]?.id) {
                updateNativeMediaInfo(currentTrack);
            }

            setState(newState);
            setStatusText(newState.playing ? '▶️ Playing' : '⏸️ Paused');

        } catch (e) {
            setStatusText(`Error: ${e.message}. Configure server.`);
            console.error('Refresh failed:', e);
        }
    }, [state.playlist, state.currentIndex, updateNativeMediaInfo]);


    // --- Event Listeners and Polling ---
    useEffect(() => {
        // Polling loop (less aggressive than original, or use WebSockets if Navidrome supports)
        const pollInterval = setInterval(refreshState, 2000); 

        // Tauri Media Key Event Listener
        const unlisten = listen('media-key-event', async (event) => {
            console.log("Received native media key event:", event.payload);
            
            // Map native event to Jukebox API call
            let action;
            if (event.payload === 'play-pause') {
                action = state.playing ? 'stop' : 'start';
            } else if (event.payload === 'next-track') {
                action = 'skip'; // needs index update, simpler to use next
                await callJukebox('next');
            } else if (event.payload === 'prev-track') {
                await callJukebox('previous');
            }

            if (action) {
                await callJukebox(action);
            }
            refreshState();
        });

        // Cleanup
        return () => {
            clearInterval(pollInterval);
            unlisten.then(f => f());
        };
    }, [refreshState, state.playing]);


    // --- Render UI (Simplified JSX) ---
    const currentTrack = state.playlist[state.currentIndex];

    return (
        <div className="player-shell">
            <div className="cover-card">
                <img src={coverArtUrl(currentTrack?.coverArt, 512)} alt="Album art" />
                <div className="title">{currentTrack?.title || 'Nothing playing'}</div>
                <div className="artist">{currentTrack?.artist || '—'}</div>
            </div>
            <div className="transport-card">
                <div id="statusText">{statusText}</div>
                {/* Control buttons updated to use React state and Jukebox API */}
                <button onClick={() => callJukebox(state.playing ? 'stop' : 'start').then(refreshState)}>
                    {state.playing ? '⏸️ Pause' : '▶️ Play'}
                </button>
                <button onClick={() => callJukebox('next').then(refreshState)}>⏭️ Next</button>
                <button onClick={() => addRandomSong().then(refreshState)}>🎲 Random</button>
                {/* ... other controls */}
            </div>
            {/* ... other sections for Queue, Search, Config */}
        </div>
    );
}

export default App;
