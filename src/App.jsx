// src/App.jsx
import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { 
    callJukebox, 
    coverArtUrl, 
    addRandomSong, 
    searchSongs,
    getConfig,
    saveConfig,
    escapeHtml 
} from './jukeboxApi';
import './App.css'; 

// --- UTILITY FUNCTIONS ---
function fmtTime(sec) {
    sec = Math.max(0, Math.floor(sec));
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return `${m}:${String(s).padStart(2, '0')}`;
}

// --- INITIAL STATE ---
const initialState = {
    playlist: [],
    currentIndex: 0,
    playing: false,
    gain: 1,
    position: 0,
    lastStatusTs: 0,
    localTickStart: 0,
    repeatMode: 'off', // 'off' | 'all' | 'one'
    seeking: false,
    endHandledForId: null,
};

// Component for a single queue item
function JukeboxQueueItem({ song, index, currentIndex, onAction }) {
    const isCurrent = index === currentIndex;
    
    return (
        <div 
            className={`qitem${isCurrent ? ' current' : ''}`}
            data-index={index}
        >
            <div className="idx">{index + 1}</div>
            <div>
                <div className="qi-title">{escapeHtml(song.title || 'Unknown')}</div>
                <div className="qi-meta">{escapeHtml(song.artist || 'Unknown')} • {escapeHtml(song.album || '')}</div>
            </div>
            <div className="qi-actions">
                <button title="Play here" className="btn" onClick={() => onAction('play', index)}>▶️</button>
                <button title="Remove" className="btn" onClick={() => onAction('remove', index)}>✖️</button>
            </div>
        </div>
    );
}

export default function App() {
    const [state, setState] = useState(initialState);
    const [statusText, setStatusText] = useState('Initializing...');
    const [searchQuery, setSearchQuery] = useState('');
    const [searchResults, setSearchResults] = useState([]);
    const [configForm, setConfigForm] = useState(getConfig()); 

    // --- Core State Refresh Logic (Polls the server) ---
    const refreshState = useCallback(async () => {
        try {
            const { status, playlist } = await callJukebox('get');
            
            const newPlaylist = Array.isArray(playlist.entry) ? playlist.entry : (playlist.entry ? [playlist.entry] : []);
            
            const newState = {
                ...state, 
                ...status, 
                playlist: newPlaylist,
            };
            
            // Check if track changed to reset repeat state
            const currentTrack = newState.playlist[newState.currentIndex];
            if (currentTrack?.id !== state.playlist[state.currentIndex]?.id) {
                newState.endHandledForId = null; 
            }
            
            setState(newState);
            setStatusText(newState.playing ? '▶️ Playing' : '⏸️ Paused');
        } catch (e) {
            setStatusText(`Error: ${e.message}. Check server or config.`);
            console.error('Refresh failed:', e);
        }
    }, [state]);

    // Handles skipping to a specific index/offset
    const skipTo = useCallback(async (index, offsetSec = 0) => {
        index = Math.max(0, Math.min(index, state.playlist.length - 1));
        try {
            await callJukebox('skip', `&index=${index}&offset=${Math.max(0, Math.floor(offsetSec))}`);
            await refreshState();
        } catch (e) {
            console.error(e);
        }
    }, [state.playlist.length, refreshState]);

    // Handles all transport button clicks
    const handleTransport = useCallback(async (action) => {
        try {
            if (action === 'play-pause') {
                await callJukebox(state.playing ? 'stop' : 'start');
            } else if (action === 'next') {
                await callJukebox('skip', `&index=${state.currentIndex + 1}`);
            } else if (action === 'previous') {
                // Implements the original logic: restart song if pos > 3s, otherwise skip back
                const restart = (state.position || 0) > 3;
                const target = restart ? state.currentIndex : Math.max(0, state.currentIndex - 1);
                await callJukebox('skip', `&index=${target}&offset=0`);
            } else if (action === 'clear') {
                if (!confirm('Clear the whole queue?')) return;
                await callJukebox('clear');
            } else if (action === 'shuffle') {
                await callJukebox('shuffle');
            } else if (action === 'stop') {
                await callJukebox('stop');
            } else if (action === 'addRandom') {
                setStatusText('Adding random song…');
                const { randomSong } = await addRandomSong();
                if (!state.playing && state.playlist.length === 0) await callJukebox('start');
                setStatusText(`Random song added: ${randomSong.title}!`);
            }
            await refreshState();
        } catch (e) {
            setStatusText(`Action failed: ${e.message}`);
            console.error('Transport action failed:', e);
            setTimeout(() => refreshState(), 2000);
        }
    }, [state.playing, state.currentIndex, state.playlist, state.position, refreshState]);

    // Handles actions from a queue item row
    const handleQueueAction = async (action, index) => {
        try {
            if (action === 'play') {
                await skipTo(index, 0);
            } else if (action === 'remove') {
                await callJukebox('remove', `&index=${index}`);
            }
            await refreshState();
        } catch(e) {
            console.error(e);
        }
    };
    
    // --- Effects & Listeners ---

    // Initialization - runs ONCE on mount
    useEffect(() => {
        (async function init() {
            try {
                const saved = localStorage.getItem('jukeboxConfig');
                if (saved) {
                    setConfigForm(getConfig());
                    setStatusText('Reconnecting…');
                    await refreshState();
                    // Check current state after refresh
                    const { playlist } = await callJukebox('get');
                    const currentPlaylist = Array.isArray(playlist.entry) ? playlist.entry : (playlist.entry ? [playlist.entry] : []);
                    
                    if (currentPlaylist.length === 0) {
                        for (let i = 0; i < 3; i++) {
                            await addRandomSong();
                        }
                        await refreshState();
                    }
                    setStatusText('Ready');
                }
            } catch (e) {
                console.error(e);
                setStatusText('Error connecting. Configure server.');
            }
        })();
    }, []); // Empty deps - run once on mount

    // Polling loop - runs ONCE on mount
    useEffect(() => {
        const pollInterval = setInterval(refreshState, 2000);
        return () => clearInterval(pollInterval);
    }, [refreshState]);

    // Position ticker and auto-repeat - runs on state changes for logic
    useEffect(() => {
        const tickInterval = setInterval(() => {
            if (state.playing && !state.seeking) {
                const tr = state.playlist[state.currentIndex];
                const dur = Math.max(0, tr?.duration || 0);
                const dt = (Date.now() - state.lastStatusTs) / 1000;
                let pos = Math.min(dur, state.localTickStart + dt);
                
                // End-of-song/Auto-Repeat Logic
                if (dur > 3 && (dur - pos) <= 0.8 && state.endHandledForId !== tr?.id) {
                    const currentId = tr?.id;
                    if (state.repeatMode === 'one') {
                        setState(s => ({ ...s, endHandledForId: currentId }));
                        skipTo(state.currentIndex, 0); 
                    } else if (state.repeatMode === 'all' && state.currentIndex === state.playlist.length - 1) {
                        setState(s => ({ ...s, endHandledForId: currentId }));
                        skipTo(0, 0); 
                    }
                }
                setState(s => ({ ...s, position: pos }));
            }
        }, 500);

        return () => clearInterval(tickInterval);
    }, [state.playing, state.seeking, state.currentIndex, state.playlist, state.lastStatusTs, 
        state.localTickStart, state.repeatMode, state.endHandledForId, skipTo]);

    // Volume Change Handler
    const handleVolumeChange = async (e) => {
        const volumeValue = Number(e.target.value);
        const gain = Math.max(0, Math.min(1, volumeValue / 100));
        setState(s => ({ ...s, gain }));
        try {
            await callJukebox('setGain', `&gain=${gain}`);
        } catch (e) {
            console.error(e);
        }
    };

    // Seek Logic
    const handleSeekInput = (e) => {
        const tr = state.playlist[state.currentIndex];
        const dur = Math.max(0, tr?.duration || 0);
        const fill = Number(e.target.value);
        const pos = (fill / 1000) * dur;
        
        setState(s => ({ 
            ...s, 
            seeking: true, 
            position: pos,
            localTickStart: pos,
            lastStatusTs: Date.now(),
        }));
    };
    
    const handleSeekChange = async (e) => {
        const tr = state.playlist[state.currentIndex];
        const dur = Math.max(0, tr?.duration || 0);
        const pos = (Number(e.target.value) / 1000) * dur;
        
        setState(s => ({ ...s, seeking: false }));
        await skipTo(state.currentIndex, pos);
    };
    
    // Search Logic (Uses useEffect debounce logic)
    useEffect(() => {
        let searchTimer;
        const q = searchQuery.trim();
        if (q.length >= 2) {
            searchTimer = setTimeout(async () => {
                try {
                    const results = await searchSongs(q);
                    setSearchResults(results);
                } catch (e) {
                    console.error('Search failed:', e);
                    setSearchResults([]);
                }
            }, 250);
        } else {
            setSearchResults([]);
        }
        return () => clearTimeout(searchTimer);
    }, [searchQuery]);

    const addSongFromSearch = async (id) => {
        try {
            await callJukebox('add', `&id=${encodeURIComponent(id)}`);
            if (!state.playing && state.playlist.length === 0) await callJukebox('start');
            setSearchQuery('');
            await refreshState();
        } catch (e) {
            console.error(e);
        }
    };
    
    // Config Logic
    const handleConfigChange = (e) => {
        setConfigForm(f => ({ ...f, [e.target.id]: e.target.value }));
    };
    
    const handleConnect = async () => {
        try {
            saveConfig({ ...configForm });
            
            setStatusText('Configuration Saved. Reconnecting…');
            await refreshState();
            
            // Add initial song if queue is empty
            if (state.playlist.length === 0) {
              await handleTransport('addRandom'); 
            }
            setStatusText(state.playing ? '▶️ Playing' : 'Ready');
            
        } catch (e) {
            setStatusText(`Login failed: ${e.message}. Check URL/credentials.`);
            console.error(e);
        }
    };

    // --- Derived State (Memoization) ---
    const currentTrack = state.playlist[state.currentIndex];
    
    const seekValue = useMemo(() => {
        const tr = state.playlist[state.currentIndex];
        const dur = Math.max(0, tr?.duration || 0);
        const pos = Math.max(0, Math.min(dur, state.position));
        return dur ? Math.round((pos / dur) * 1000) : 0;
    }, [state.position, state.currentIndex, state.playlist]);
    
    const seekFillStyle = useMemo(() => ({
        '--seek-fill': `${(seekValue / 10).toFixed(1)}%`
    }), [seekValue]);
    
    const volFillStyle = useMemo(() => ({
        '--vol-fill': `${Math.round(state.gain * 100)}%`
    }), [state.gain]);


    // --- RENDER ---
    return (
        <div className="player-shell">
            <aside className="cover-card">
                <img id="cover" className="cover" alt="Album art" src={coverArtUrl(currentTrack?.coverArt, 900)} />
                <div className="meta">
                    <div id="title" className="title">{currentTrack?.title || 'Nothing playing'}</div>
                    <div id="artist" className="artist">{currentTrack?.artist || '—'}</div>
                    <div id="album" className="album">{currentTrack?.album || ''}&nbsp;</div>
                </div>
            </aside>

            <main className="transport-card">
                <div className="status-row">
                    <div id="statusText">{statusText}</div>
                    <div className="small">Queue: <span id="queueCount">{state.playlist.length}</span> tracks</div>
                </div>
                
                <div className="progress">
                    <div id="currentTime" className="time">{fmtTime(state.position)}</div>
                    <input 
                        id="seek" 
                        className="seek" 
                        type="range" 
                        min="0" 
                        max="1000" 
                        value={seekValue} 
                        style={seekFillStyle}
                        onChange={handleSeekChange}
                        onInput={handleSeekInput}
                    />
                    <div id="totalTime" className="time">{fmtTime(currentTrack?.duration || 0)}</div>
                </div>
                
                <div>
                    <div className="controls">
                        <button 
                            id="btnShuffle" 
                            className="btn" 
                            title="Shuffle queue" 
                            aria-label="Shuffle"
                            onClick={() => handleTransport('shuffle')}>🔀</button>
                        <button 
                            id="btnPrev" 
                            className="btn" 
                            title="Previous" 
                            aria-label="Previous"
                            onClick={() => handleTransport('previous')}>⏮️</button>
                        <button 
                            id="btnPlay" 
                            className={`btn primary ${state.playing ? 'paused' : ''}`}
                            title="Play/Pause" 
                            aria-label="Play/Pause"
                            onClick={() => handleTransport('play-pause')}>
                            {state.playing ? '⏸️' : '▶️'}
                        </button>
                        <button 
                            id="btnNext" 
                            className="btn" 
                            title="Next" 
                            aria-label="Next"
                            onClick={() => handleTransport('next')}>⏭️</button>
                        <button 
                            id="btnRepeat" 
                            className={`btn ${state.repeatMode !== 'off' ? 'active' : ''}`}
                            title="Repeat" 
                            aria-label="Repeat"
                            onClick={() => setState(s => ({ 
                                ...s, 
                                repeatMode: s.repeatMode === 'off' ? 'all' : s.repeatMode === 'all' ? 'one' : 'off'
                            }))}>
                            {state.repeatMode === 'one' ? '🔂1' : state.repeatMode === 'all' ? '🔂' : '🔁'}
                        </button>
                        <button 
                            id="btnStop" 
                            className="btn warn" 
                            title="Stop" 
                            aria-label="Stop"
                            onClick={() => handleTransport('stop')}>⏹️</button>
                        <button 
                            id="btnClear" 
                            className="btn danger" 
                            title="Clear queue" 
                            aria-label="Clear"
                            onClick={() => handleTransport('clear')}>🗑️</button>
                        <button 
                            id="btnAddRandom" 
                            className="btn" 
                            title="Add Random Song to Queue" 
                            aria-label="Add Random"
                            onClick={() => handleTransport('addRandom')}>🎲</button>
                    </div>
                    
                    <div className="vol">
                        <div className="vol-row">
                            <div title="Volume">🔊</div>
                            <input 
                                id="volume" 
                                type="range" 
                                min="0" 
                                max="100" 
                                value={Math.round(state.gain * 100)}
                                style={volFillStyle}
                                onChange={handleVolumeChange}
                                onInput={handleVolumeChange} 
                            />
                            <div id="volPct" className="volpct">{Math.round(state.gain * 100)}%</div>
                        </div>
                    </div>
                </div>
            </main>

            <aside className="side-card">
                <h3>Queue</h3>
                <div id="queue" className="queue" aria-label="Playlist queue">
                    {state.playlist.map((song, index) => (
                        <JukeboxQueueItem 
                            key={song.id || index} 
                            song={song}
                            index={index}
                            currentIndex={state.currentIndex}
                            onAction={handleQueueAction}
                        />
                    ))}
                </div>
                
                <div className="search-box">
                    <input 
                        id="search" 
                        placeholder="Search songs to add…"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                    />
                    <div id="sresults" className="search-results" hidden={searchResults.length === 0}>
                        {searchResults.map((song) => (
                            <div key={song.id} className="srow" onClick={() => addSongFromSearch(song.id)}>
                                <img 
                                    src={coverArtUrl(song.coverArt, 80)} 
                                    alt="Cover" 
                                    onError={(e) => e.target.style.visibility='hidden'}
                                />
                                <div className="s-meta">
                                    <div className="qi-title">{escapeHtml(song.title || 'Unknown')}</div>
                                    <div className="s-artist">{escapeHtml(song.artist || '')} • {escapeHtml(song.album || '')}</div>
                                </div>
                                <div style={{textAlign: 'right'}}>➕</div>
                            </div>
                        ))}
                    </div>
                </div>
                
                <div className="config">
                    <div className="row">
                        <input 
                            id="serverUrl" 
                            placeholder="Server URL (e.g., http://localhost:4533)"
                            value={configForm.serverUrl || ''}
                            onChange={handleConfigChange}
                        />
                        <input 
                            id="username" 
                            placeholder="Username"
                            value={configForm.username || ''}
                            onChange={handleConfigChange}
                        />
                        <input 
                            id="token" 
                            placeholder="Token (Advanced)" 
                            type="password"
                            value={configForm.token || ''}
                            onChange={handleConfigChange}
                        />
                         <input 
                            id="salt" 
                            placeholder="Salt (Advanced)" 
                            type="password"
                            value={configForm.salt || ''}
                            onChange={handleConfigChange}
                        />
                        <button id="btnConnect" onClick={handleConnect}>Save & Connect</button>
                    </div>
                    <div className="small">Auth token/salt should be manually generated/entered for full functionality. Saved in <code>localStorage</code>.</div>
                </div>
            </aside>
        </div>
    );
}
