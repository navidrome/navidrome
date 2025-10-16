// src/App.jsx
import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
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
    
    // Use ref to track if we're currently executing a command to prevent race conditions
    const commandInProgress = useRef(false);
    const stateRef = useRef(state);
    
    // Keep stateRef in sync with state
    useEffect(() => {
        stateRef.current = state;
    }, [state]);

    // --- Core State Refresh Logic (Polls the server) ---
    const refreshState = useCallback(async (forceUpdate = false) => {
        // Don't refresh while a command is in progress unless forced
        if (!forceUpdate && commandInProgress.current) {
            return;
        }
        
        try {
            const result = await callJukebox('get');
            console.log('API Response:', result); // DEBUG
            
            const { status, playlist } = result;
            
            const newPlaylist = Array.isArray(playlist?.entry) 
                ? playlist.entry 
                : (playlist?.entry ? [playlist.entry] : []);
            
            // If status is missing, still update playlist but keep existing state
            if (!status) {
                console.warn('API returned no status object, updating playlist only');
                setState(prevState => ({
                    ...prevState,
                    playlist: newPlaylist.length > 0 ? newPlaylist : prevState.playlist,
                }));
                return;
            }
            
            setState(prevState => {
                // Check if track changed to reset repeat state
                const currentTrack = newPlaylist[status.currentIndex || 0];
                const prevTrack = prevState.playlist[prevState.currentIndex];
                
                return {
                    ...prevState,
                    playing: status.playing ?? prevState.playing,
                    currentIndex: status.currentIndex ?? prevState.currentIndex,
                    position: status.position ?? prevState.position,
                    gain: status.gain ?? prevState.gain,
                    playlist: newPlaylist,
                    lastStatusTs: Date.now(),
                    localTickStart: status.position ?? prevState.position,
                    endHandledForId: currentTrack?.id !== prevTrack?.id ? null : prevState.endHandledForId,
                };
            });
            
            setStatusText(status.playing ? '▶️ Playing' : '⏸️ Paused');
        } catch (e) {
            setStatusText(`Error: ${e.message}. Check server or config.`);
            console.error('Refresh failed:', e);
        }
    }, []);

    // Handles skipping to a specific index/offset
    const skipTo = useCallback(async (index, offsetSec = 0) => {
        const currentState = stateRef.current;
        index = Math.max(0, Math.min(index, currentState.playlist.length - 1));
        
        commandInProgress.current = true;
        try {
            await callJukebox('skip', `&index=${index}&offset=${Math.max(0, Math.floor(offsetSec))}`);
            await refreshState(true); // Force refresh after command
        } catch (e) {
            console.error(e);
        } finally {
            commandInProgress.current = false;
        }
    }, [refreshState]);

    // Handles all transport button clicks
    const handleTransport = useCallback(async (action) => {
        const currentState = stateRef.current;
        
        commandInProgress.current = true;
        try {
            if (action === 'play-pause') {
                const cmd = currentState.playing ? 'stop' : 'start';
                await callJukebox(cmd);
                // Immediately update local state for better UX
                setState(prev => ({ ...prev, playing: !prev.playing }));
            } else if (action === 'next') {
                const nextIndex = Math.min(currentState.currentIndex + 1, currentState.playlist.length - 1);
                await callJukebox('skip', `&index=${nextIndex}&offset=0`);
            } else if (action === 'previous') {
                // Restart song if pos > 3s, otherwise skip back
                const restart = (currentState.position || 0) > 3;
                const target = restart ? currentState.currentIndex : Math.max(0, currentState.currentIndex - 1);
                await callJukebox('skip', `&index=${target}&offset=0`);
            } else if (action === 'clear') {
                if (!confirm('Clear the whole queue?')) {
                    commandInProgress.current = false;
                    return;
                }
                await callJukebox('clear');
            } else if (action === 'shuffle') {
                await callJukebox('shuffle');
            } else if (action === 'stop') {
                await callJukebox('stop');
                setState(prev => ({ ...prev, playing: false }));
            } else if (action === 'addRandom') {
                setStatusText('Adding random song…');
                const { randomSong } = await addRandomSong();
                if (!currentState.playing && currentState.playlist.length === 0) {
                    await callJukebox('start');
                }
                setStatusText(`Random song added: ${randomSong.title}!`);
            }
            
            // Force refresh after command completes
            await refreshState(true);
        } catch (e) {
            setStatusText(`Action failed: ${e.message}`);
            console.error('Transport action failed:', e);
        } finally {
            commandInProgress.current = false;
        }
    }, [refreshState]);

    // Handles actions from a queue item row
    const handleQueueAction = useCallback(async (action, index) => {
        commandInProgress.current = true;
        try {
            if (action === 'play') {
                await skipTo(index, 0);
            } else if (action === 'remove') {
                await callJukebox('remove', `&index=${index}`);
                await refreshState(true);
            }
        } catch(e) {
            console.error(e);
        } finally {
            commandInProgress.current = false;
        }
    }, [skipTo, refreshState]);
    
    // --- Effects & Listeners ---

    // Initialization - runs ONCE on mount
    useEffect(() => {
        let mounted = true;
        
        (async function init() {
            try {
                const saved = localStorage.getItem('jukeboxConfig');
                if (saved) {
                    setConfigForm(getConfig());
                    setStatusText('Reconnecting…');
                    await refreshState(true);
                    
                    if (!mounted) return;
                    
                    // Check current state after refresh
                    const { playlist } = await callJukebox('get');
                    const currentPlaylist = Array.isArray(playlist.entry) ? playlist.entry : (playlist.entry ? [playlist.entry] : []);
                    
                    if (mounted && currentPlaylist.length === 0) {
                        for (let i = 0; i < 3; i++) {
                            if (!mounted) break;
                            await addRandomSong();
                        }
                        if (mounted) {
                            await refreshState(true);
                        }
                    }
                    
                    if (mounted) {
                        setStatusText('Ready');
                    }
                }
            } catch (e) {
                console.error(e);
                if (mounted) {
                    setStatusText('Error connecting. Configure server.');
                }
            }
        })();
        
        return () => { mounted = false; };
    }, []); // Empty deps - run once on mount

    // Polling loop - runs ONCE on mount
    useEffect(() => {
        const pollInterval = setInterval(() => {
            refreshState(false); // Don't force during polling
        }, 2000);
        
        return () => clearInterval(pollInterval);
    }, [refreshState]);

    // Position ticker and auto-repeat
    useEffect(() => {
        const tickInterval = setInterval(() => {
            setState(prevState => {
                if (!prevState.playing || prevState.seeking) {
                    return prevState;
                }
                
                const tr = prevState.playlist[prevState.currentIndex];
                const dur = Math.max(0, tr?.duration || 0);
                const dt = (Date.now() - prevState.lastStatusTs) / 1000;
                let pos = Math.min(dur, prevState.localTickStart + dt);
                
                // End-of-song/Auto-Repeat Logic
                if (dur > 3 && (dur - pos) <= 0.8 && prevState.endHandledForId !== tr?.id) {
                    const currentId = tr?.id;
                    if (prevState.repeatMode === 'one') {
                        // Schedule skip on next tick
                        setTimeout(() => skipTo(prevState.currentIndex, 0), 0);
                        return { ...prevState, position: pos, endHandledForId: currentId };
                    } else if (prevState.repeatMode === 'all' && prevState.currentIndex === prevState.playlist.length - 1) {
                        setTimeout(() => skipTo(0, 0), 0);
                        return { ...prevState, position: pos, endHandledForId: currentId };
                    }
                }
                
                return { ...prevState, position: pos };
            });
        }, 500);

        return () => clearInterval(tickInterval);
    }, [skipTo]);

    // Volume Change Handler
    const handleVolumeChange = useCallback(async (e) => {
        const volumeValue = Number(e.target.value);
        const gain = Math.max(0, Math.min(1, volumeValue / 100));
        
        setState(prev => ({ ...prev, gain }));
        
        try {
            await callJukebox('setGain', `&gain=${gain}`);
        } catch (e) {
            console.error(e);
        }
    }, []);

    // Seek Logic
    const handleSeekInput = useCallback((e) => {
        setState(prevState => {
            const tr = prevState.playlist[prevState.currentIndex];
            const dur = Math.max(0, tr?.duration || 0);
            const fill = Number(e.target.value);
            const pos = (fill / 1000) * dur;
            
            return { 
                ...prevState, 
                seeking: true, 
                position: pos,
                localTickStart: pos,
                lastStatusTs: Date.now(),
            };
        });
    }, []);
    
    const handleSeekChange = useCallback(async (e) => {
        const currentState = stateRef.current;
        const tr = currentState.playlist[currentState.currentIndex];
        const dur = Math.max(0, tr?.duration || 0);
        const pos = (Number(e.target.value) / 1000) * dur;
        
        setState(prev => ({ ...prev, seeking: false }));
        await skipTo(currentState.currentIndex, pos);
    }, [skipTo]);
    
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

    const addSongFromSearch = useCallback(async (id) => {
        const currentState = stateRef.current;
        commandInProgress.current = true;
        
        try {
            await callJukebox('add', `&id=${encodeURIComponent(id)}`);
            if (!currentState.playing && currentState.playlist.length === 0) {
                await callJukebox('start');
            }
            setSearchQuery('');
            await refreshState(true);
        } catch (e) {
            console.error(e);
        } finally {
            commandInProgress.current = false;
        }
    }, [refreshState]);
    
    // Config Logic
    const handleConfigChange = useCallback((e) => {
        setConfigForm(f => ({ ...f, [e.target.id]: e.target.value }));
    }, []);
    
    const handleConnect = useCallback(async () => {
        try {
            saveConfig({ ...configForm });
            
            setStatusText('Configuration Saved. Reconnecting…');
            await refreshState(true);
            
            const currentState = stateRef.current;
            // Add initial song if queue is empty
            if (currentState.playlist.length === 0) {
              await handleTransport('addRandom'); 
            }
            setStatusText(currentState.playing ? '▶️ Playing' : 'Ready');
            
        } catch (e) {
            setStatusText(`Login failed: ${e.message}. Check URL/credentials.`);
            console.error(e);
        }
    }, [configForm, refreshState, handleTransport]);

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
                            onClick={() => handleTransport('shuffle')}
                            disabled={commandInProgress.current}>🔀</button>
                        <button 
                            id="btnPrev" 
                            className="btn" 
                            title="Previous" 
                            aria-label="Previous"
                            onClick={() => handleTransport('previous')}
                            disabled={commandInProgress.current}>⏮️</button>
                        <button 
                            id="btnPlay" 
                            className={`btn primary ${state.playing ? 'paused' : ''}`}
                            title="Play/Pause" 
                            aria-label="Play/Pause"
                            onClick={() => handleTransport('play-pause')}
                            disabled={commandInProgress.current}>
                            {state.playing ? '⏸️' : '▶️'}
                        </button>
                        <button 
                            id="btnNext" 
                            className="btn" 
                            title="Next" 
                            aria-label="Next"
                            onClick={() => handleTransport('next')}
                            disabled={commandInProgress.current}>⏭️</button>
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
                            onClick={() => handleTransport('stop')}
                            disabled={commandInProgress.current}>⏹️</button>
                        <button 
                            id="btnClear" 
                            className="btn danger" 
                            title="Clear queue" 
                            aria-label="Clear"
                            onClick={() => handleTransport('clear')}
                            disabled={commandInProgress.current}>🗑️</button>
                        <button 
                            id="btnAddRandom" 
                            className="btn" 
                            title="Add Random Song to Queue" 
                            aria-label="Add Random"
                            onClick={() => handleTransport('addRandom')}
                            disabled={commandInProgress.current}>🎲</button>
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
