

# brew install ffmpeg
# pip install yt-dlp

import yt_dlp

def download_audio(artist, track, output_dir="./downloads"):
    query = f"{artist} - {track} official audio"
    
    ydl_opts = {
        "format": "bestaudio/best",
        "postprocessors": [{
            "key": "FFmpegExtractAudio",
            "preferredcodec": "mp3",
            "preferredquality": "192",
        }],
        "outtmpl": f"{output_dir}/{artist} - {track}.%(ext)s",
        "quiet": False,
        "noplaylist": True,
        "default_search": "ytsearch1",  # search YouTube, take first result
    }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        ydl.download([query])

# Example
download_audio("Taylor Swift", "Blank Space")