/**
 * APlayer integration for Navidrome Share Links
 * Works with public share links without authentication
 */
(function() {
  'use strict';

  // Wait for DOM and APlayer to be ready
  function initAPlayer() {
    console.log('APlayer initialization started');
    
    if (typeof APlayer === 'undefined') {
      console.error('APlayer library not loaded');
      return;
    }
    console.log('APlayer library loaded');

    // Get share info from the page (injected by server)
    const shareInfoElement = document.getElementById('share-info');
    if (!shareInfoElement) {
      console.error('Share info not found');
      return;
    }
    console.log('Share info element found:', shareInfoElement.textContent);

    let shareInfo;
    try {
      shareInfo = JSON.parse(shareInfoElement.textContent);
    } catch (e) {
      console.error('Failed to parse share info:', e);
      return;
    }

    if (!shareInfo || !shareInfo.tracks || shareInfo.tracks.length === 0) {
      console.error('No tracks found in share');
      return;
    }

    // Get base URL from the page
    const baseURL = window.NavidromeConfig?.baseURL || '';

    // Convert share tracks to APlayer format
    const playlist = shareInfo.tracks.map(function(track) {
      // Stream URL uses the encoded track ID (contains JWT token)
      const streamUrl = baseURL + '/share/s/' + track.id;

      // Cover art URL - we'll construct it from the share's image
      const coverUrl = shareInfo.imageUrl || baseURL + '/android-chrome-192x192.png';

      return {
        name: track.title || 'Unknown Title',
        artist: track.artist || 'Unknown Artist',
        url: streamUrl,
        cover: coverUrl,
        theme: '#b7daff'
      };
    });

    // Initialize APlayer
    const container = document.getElementById('aplayer');
    if (!container) {
      console.error('APlayer container not found');
      return;
    }
    console.log('APlayer container found:', container);

    console.log('Creating APlayer with playlist:', playlist);
    
    const ap = new APlayer({
      container: container,
      lrcType: 0,
      audio: playlist,
      autoplay: false,
      theme: '#b7daff',
      loop: 'all',
      order: 'list',
      preload: 'auto',
      volume: 0.7,
      mutex: true,
      listFolded: false,
      listMaxHeight: 90,
      fixed: false,
      mini: false,
    });

    // Log initialization
    console.log('APlayer initialized with', playlist.length, 'tracks');

    // Optional: Add event listeners
    ap.on('play', function() {
      console.log('Playing:', ap.list.audios[ap.list.index].name);
    });

    ap.on('error', function() {
      console.error('Playback error');
    });
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initAPlayer);
  } else {
    initAPlayer();
  }
})();
