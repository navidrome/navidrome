// src/jukeboxApi.js

const API_VERSION = '1.16.1';

// Set to true in browser console to enable debug logging: window.JUKEBOX_DEBUG = true
const DEBUG = () => typeof window !== 'undefined' && window.JUKEBOX_DEBUG === true;

// Global configuration state, initialized from localStorage
let config = JSON.parse(localStorage.getItem('jukeboxConfig')) || {
    serverUrl: 'http://localhost:4533',
    username: '',
    token: '',
    salt: ''
};

// --- Utilities ---
export function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', '\'': '&#39;' }[c]));
}

function buildJukeboxUrl(action, extra = '') {
    if (!config.token || !config.salt) {
        throw new Error('Authentication missing. Please connect to the server.');
    }
    const base = `${config.serverUrl}/rest/jukeboxControl?u=${encodeURIComponent(config.username)}&t=${config.token}&s=${config.salt}&v=${API_VERSION}&c=ModernJukebox&f=json`;
    return `${base}&action=${action}${extra}`;
}

export function coverArtUrl(id, size = 512) {
    if (!id || !config.token || !config.salt) return '';
    return `${config.serverUrl}/rest/getCoverArt?id=${encodeURIComponent(id)}&size=${size}&u=${encodeURIComponent(config.username)}&t=${config.token}&s=${config.salt}&v=${API_VERSION}&c=ModernJukebox`;
}

// --- Jukebox API Call ---
export async function callJukebox(action, extra = '') {
    const url = buildJukeboxUrl(action, extra);
    
    const res = await fetch(url);
    if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`);
    const data = await res.json();
    
    if (DEBUG()) console.log(`API ${action} response:`, data);
    
    const resp = data?.['subsonic-response'];
    if (resp?.status !== 'ok') {
        const errorMsg = resp?.error?.message || 'Unknown API error';
        throw new Error(`API failed: ${errorMsg}`);
    }
    
    // Navidrome API is inconsistent:
    // - 'get' returns status fields in jukeboxPlaylist
    // - 'add'/'skip'/etc return status in jukeboxStatus
    const playlistObj = resp.jukeboxPlaylist || {};
    const statusObj = resp.jukeboxStatus || playlistObj;
    
    // Extract status fields from the correct object
    const status = {
        currentIndex: statusObj.currentIndex ?? 0,
        playing: statusObj.playing ?? false,
        gain: statusObj.gain ?? 1,
        position: statusObj.position ?? 0,
    };
    
    // Extract playlist entries (only in jukeboxPlaylist)
    const playlist = {
        entry: playlistObj.entry || []
    };
    
    if (DEBUG()) {
        console.log('Parsed status:', status);
        console.log('Parsed playlist:', playlist.entry?.length || 0, 'tracks');
    }
    
    // Return structured data for React component to process
    return { status, playlist };
}

// --- Random Song & Search ---
async function getRandomSongFromServer() {
    const url = `${config.serverUrl}/rest/getRandomSongs?u=${encodeURIComponent(config.username)}&t=${config.token}&s=${config.salt}&v=${API_VERSION}&c=ModernJukebox&f=json&size=1`;
    
    const res = await fetch(url);
    const data = await res.json();
    const resp = data?.['subsonic-response'];
    
    if (resp?.status !== 'ok') {
        throw new Error(`API failed: ${resp?.error?.message || 'Unknown API error'}`);
    }
    
    const song = Array.isArray(resp.randomSongs?.song) 
        ? resp.randomSongs.song[0] 
        : resp.randomSongs?.song;
        
    if (!song || !song.id) {
        throw new Error('Server returned no songs.');
    }
    
    return song;
}

export async function addRandomSong() {
    const randomSong = await getRandomSongFromServer();
    const resp = await callJukebox('add', `&id=${encodeURIComponent(randomSong.id)}`);
    
    return { randomSong, resp };
}

export async function searchSongs(query) {
    if (query.length < 2) return [];
    
    const url = `${config.serverUrl}/rest/search3?u=${encodeURIComponent(config.username)}&t=${config.token}&s=${config.salt}&v=${API_VERSION}&c=ModernJukebox&f=json&query=${encodeURIComponent(query)}`;
    const res = await fetch(url);
    const data = await res.json();
    
    return data?.['subsonic-response']?.searchResult3?.song || [];
}

// --- Config Management ---
export function getConfig() {
    return config;
}

export function saveConfig(newConfig) {
    config = { ...config, ...newConfig };
    localStorage.setItem('jukeboxConfig', JSON.stringify(config));
    return config;
}
