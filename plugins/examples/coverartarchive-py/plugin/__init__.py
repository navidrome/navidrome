# Cover Art Archive Plugin for Navidrome
#
# This plugin fetches album cover art from the Cover Art Archive (https://coverartarchive.org/)
# using the MusicBrainz album MBID.
#
# Build with:
#   extism-py plugin/__init__.py -o coverartarchive-py.wasm

import base64
import extism
import json


@extism.import_fn("extism:host/user", "http_send")
def http_send(req: dict) -> dict: ...


def http_get(url):
    """GET url via Navidrome's HTTP host service. Returns (status_code, body_bytes)."""
    resp = http_send({"request": {"method": "GET", "url": url}})
    if resp.get("error"):
        raise Exception(f"HTTP request failed: {resp['error']}")
    result = resp.get("result") or {}
    return result.get("statusCode", 0), base64.b64decode(result.get("body") or "")


@extism.plugin_fn
def nd_get_album_images():
    """Retrieve album cover images from Cover Art Archive."""
    input_data = extism.input_json()
    mbid = input_data.get("mbid", "")
    
    if not mbid:
        raise Exception("not found: MBID required")
    
    # Query Cover Art Archive API
    url = f"https://coverartarchive.org/release/{mbid}"
    status, body = http_get(url)

    if status != 200:
        raise Exception(f"not found: CAA returned status {status}")

    try:
        data = json.loads(body)
    except json.JSONDecodeError:
        raise Exception("not found: invalid JSON response")
    
    caa_images = data.get("images", [])
    if not caa_images:
        raise Exception("not found: no images in response")
    
    # Find the front cover image
    front_image = find_front_image(caa_images)
    if not front_image:
        raise Exception("not found: no front cover image")
    
    # Build the response with available image sizes
    images = build_image_list(front_image)
    if not images:
        raise Exception("not found: no usable image URLs")
    
    extism.output_str(json.dumps({"images": images}))


def find_front_image(images):
    """Find the front cover image from CAA response."""
    # First, look for an image explicitly marked as front
    for img in images:
        if img.get("front", False):
            return img
    
    # Second, look for an image with "Front" in types
    for img in images:
        types = img.get("types", [])
        if "Front" in types:
            return img
    
    # Fallback to first image
    if images:
        return images[0]
    
    return None


def build_image_list(img):
    """Build list of images with URLs and sizes from CAA image data."""
    images = []
    thumbnails = img.get("thumbnails", {})
    
    # First, try numeric sizes (250, 500, 1200, etc.)
    for size_str, url in thumbnails.items():
        if not url:
            continue
        try:
            size = int(size_str)
            images.append({"url": url, "size": size})
        except ValueError:
            pass  # Not a numeric size
    
    # If no numeric sizes, fallback to named sizes
    if not images:
        size_map = {"large": 500, "small": 250}
        for size_name, size in size_map.items():
            url = thumbnails.get(size_name)
            if url:
                images.append({"url": url, "size": size})
    
    # If still no images, use the main image URL
    if not images:
        main_url = img.get("image")
        if main_url:
            images.append({"url": main_url, "size": 0})
    
    return images
