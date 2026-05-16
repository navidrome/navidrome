# Testing Flows

## Add to Favorites

Stars a song (or album/artist) using the Subsonic `star` endpoint.

**Request**

```
GET /rest/star
```

Query parameters:

| Parameter | Value | Description |
|-----------|-------|-------------|
| `u` | `admin` | Username |
| `t` | `47b1fbf1aaa6b36d0dec0c8ae0b1a6ef` | Auth token (MD5 of password + salt) |
| `s` | `701805` | Salt used to generate the token |
| `f` | `json` | Response format |
| `v` | `1.8.0` | Subsonic API version |
| `c` | `NavidromeUI` | Client name |
| `id` | `qORBlifrm7cjK4yDgfy5Zy` | ID of the song/album/artist to star |

**Example request (dev)**

```bash
curl "http://localhost:4533/rest/star?u=admin&t=47b1fbf1aaa6b36d0dec0c8ae0b1a6ef&s=701805&f=json&v=1.8.0&c=NavidromeUI&id=qORBlifrm7cjK4yDgfy5Zy"
```

**Response**

```json
{
    "subsonic-response": {
        "status": "ok",
        "version": "1.16.1",
        "type": "navidrome",
        "serverVersion": "dev",
        "openSubsonic": true
    }
}
```

A `status: "ok"` with no error field means the item was successfully starred.

## Running Star/Unstar Tests

### Backend

```bash
go test ./server/subsonic/ --ginkgo.focus="Star/Unstar songs"
```

### Frontend

```bash
npm --prefix ui test -- useToggleLove.test.js --run
```

### Database

```bash
docker compose -f docker-compose.dev.yml exec backend \
go test ./tests/db -run TestStarEndpoint -
```
