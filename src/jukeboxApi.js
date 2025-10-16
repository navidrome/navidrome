// src/jukeboxApi.js

const API_VERSION = '1.16.1';

// Global configuration state, initialized from localStorage
let config = JSON.parse(localStorage.getItem('jukeboxConfig')) || {
    serverUrl: 'http://localhost:4533',
    username: '',
    token: '',
    salt: ''
};

// --- API Utilities ---

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

export async function callJukebox(action, extra = '') {
    const url = buildJukeboxUrl(action, extra);

    const res = await fetch(url);
    if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`);
    const data = await res.json();

    const resp = data?.['subsonic-response'];
    if (resp?.status !== 'ok') {
        const errorMsg = resp?.error?.message || 'Unknown API error';
        throw new Error(`API failed: ${errorMsg}`);
    }

    // Return structured data for React component to process
    return {
        status: resp.jukeboxStatus,
        playlist: resp.jukeboxPlaylist
    };
}

// --- Jukebox Control Functions ---

export async function addRandomSong() {
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

    await callJukebox('add', `&id=${encodeURIComponent(song.id)}`);

    return song;
}

export async function searchSongs(query) {
    if (query.length < 2) return [];

    const url = `${config.serverUrl}/rest/search3?u=${encodeURIComponent(config.username)}&t=${config.token}&s=${config.salt}&v=${API_VERSION}&c=ModernJukebox&f=json&query=${encodeURIComponent(query)}`;
    const res = await fetch(url);
    const data = await res.json();

    return data?.['subsonic-response']?.searchResult3?.song || [];
}

export function getConfig() {
    return config;
}

export function saveConfig(newConfig) {
    config = { ...config, ...newConfig };
    localStorage.setItem('jukeboxConfig', JSON.stringify(config));
    return config;
}
