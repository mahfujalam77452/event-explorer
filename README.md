# Event Explorer

A Beego (Go) web app where a visitor searches for a city, browses **Music** and **Sports** events,
opens an event's details and continues to the ticket provider.

## Features

- City search with Google Places autocomplete (only the suggestion list updates, no page reload)
- Listing page with up to 6 Music and 6 Sports events, fetched concurrently
- Event details page; direct links work
- Safe ticket redirect with an approved-hostname check
- Shared in-memory cache with hit/miss logging, plus endpoints to clear it (all or per category)
- Server-side rendering with shared header/footer templates; JavaScript is used only for the autocomplete
- Clear empty and error states, mobile-friendly layout
- Unit tests with all external APIs mocked

## Tech stack

Go, Beego v2, HTML/CSS, vanilla JavaScript.
External APIs: Google Places API (New), Ticketmaster Discovery API v2.

## Setup

Requirements: Go 1.21+, a Google API key (Places API (New) enabled) and a Ticketmaster Consumer Key.
Optional: the `bee` tool (`go install github.com/beego/bee/v2@latest`).

```bash
git clone https://github.com/mahfujalam77452/event-explorer.git
cd event-explorer
go mod tidy
cp .env.example .env     # then paste your two keys into .env
```

`.env`:

```env
GOOGLE_API_KEY=your_google_key_here
TICKETMASTER_API_KEY=your_ticketmaster_key_here
```

`.env` is git-ignored. Keys are used on the server only and never reach the browser.
The app refuses to start if either key is missing.

## Run

```bash
bee run        # or: go run main.go
```

Open http://localhost:8080

## Configuration (`conf/app.conf`)

| Key | Meaning |
|---|---|
| `httpport` | Server port (default 8080) |
| `google_base_url` | Google Places base URL |
| `ticketmaster_base_url` | Ticketmaster base URL |
| `http_timeout_seconds` | Timeout for every external call |
| `events_per_category` | Events per category (6) |
| `approved_ticket_hosts` | Domains a ticket redirect may go to (subdomains included) |

## Routes

| Method | Route | Purpose |
|---|---|---|
| GET | `/` | Home page and city search |
| GET | `/events?city=Toronto&countryCode=CA` | Music and Sports lists |
| GET | `/events/:eventId` | Event details |
| GET | `/api/locations/autocomplete?input=&sessionToken=` | JSON: city suggestions |
| GET | `/api/locations/:placeId?sessionToken=` | JSON: selected city and country code |
| GET | `/redirect/:eventId` | Validates the ticket URL, then HTTP 302 |
| GET, POST | `/admin/cache/clear` | Deletes the whole cache |
| GET, POST, DELETE | `/admin/cache/invalidate?city=&countryCode=&category=` | Deletes one cache entry |

### API responses

`GET /api/locations/autocomplete` needs `input` (at least 3 characters) and `sessionToken`, and returns up to 5 suggestions:

```json
{"suggestions":[{"placeId":"ChIJ...","text":"Toronto, Canada"}]}
```

`GET /api/locations/:placeId` returns:

```json
{"placeId":"ChIJ...","city":"Toronto","countryCode":"CA","label":"Toronto, Canada"}
```

Errors return `{"error":"..."}` with status 400 (bad input), 404 (no city for the place), 502 or 504 (Google failed or timed out).

## Cache

- One shared map protected by `sync.RWMutex`, keyed by `city|country|category` (case-insensitive).
- Only successful results are cached; failures never are.
- Every lookup is logged as `[cache] HIT` or `[cache] MISS`.
- As instructed by the course instructor, entries do not expire after five minutes. They are removed with the endpoints below instead.

```bash
# delete everything
curl -X POST http://localhost:8080/admin/cache/clear
# {"cleared":4,"message":"cache cleared"}

# delete one entry (category is Music or Sports, case-insensitive)
curl -X DELETE "http://localhost:8080/admin/cache/invalidate?city=Toronto&countryCode=CA&category=Music"
# {"deleted":true,"key":"toronto|CA|music","message":"cache entry deleted"}
```

`/admin/cache/invalidate` requires all three parameters (otherwise 400). If nothing is cached for the key it still returns 200 with `"deleted": false`. Other cities and the other category stay cached. Each deletion is logged as `[cache] DELETE key=... found=...`.

## How it works

**Concurrency.** `EventService.GetListing` starts one goroutine per category before waiting for any of them.
Each goroutine sends exactly one result (events or error) into a buffered channel. If one category fails,
its section shows an error and the other one is kept. A `recover` in each worker turns a panic into an error.

**Ticket redirect.** `/redirect/:eventId` reads only the event ID, fetches the ticket URL from Ticketmaster
on the server, and requires `https`, no embedded credentials, no unusual port and a hostname that is in
`approved_ticket_hosts` (exact match or a real subdomain, so `evilticketmaster.com` is rejected).
No destination is ever taken from the visitor. Then it answers with a 302.

**Google policies.** Google responses are never cached (`Cache-Control: no-store`), one session token is used
for the autocomplete and place-details calls of a search and a new one for the next search, and
"Powered by Google" is shown next to the search box.

## Project structure

```
conf/          app.conf
controllers/   base (Prepare), page, api, redirect, cache, error
filters/       request logging and security headers
models/        view models, raw API structs, conversion and validation
services/      location service, event service (goroutines), cache
routers/       route definitions
utils/         config, HTTP client, ticket-link validation
views/         home, listing, details, error + partials (header, footer)
static/        css/style.css, js/autocomplete.js
```

Request flow: router -> controller -> service -> (cache | external API) -> model -> template.

## Tests

```bash
go test ./...                                  # run everything
go test ./... -v -cover                        # verbose, with coverage
go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

All external APIs are replaced by local `httptest` servers, so the tests pass without internet access or API keys.
Each `_test.go` file sits next to the file it tests.

## Known limitations

- The cache is in memory: it is lost on restart and not shared between instances.
- The cache endpoints have no authentication (login is out of scope for this assessment).
- If a city has fewer than 6 events in a category, fewer cards are shown.

## Troubleshooting

| Problem | Fix |
|---|---|
| `config error: missing environment variables` | Fill in `.env`, then restart |
| Autocomplete fails (check the `[api]` log line) | Enable **Places API (New)** and billing, and make sure the key has no HTTP-referrer restriction |
| Both sections show an error | Check the Ticketmaster key and its rate limit |
| "Tickets ... not available" | The ticket host is not in `approved_ticket_hosts` |
| `/admin/cache/invalidate` returns 400 | Send `city`, `countryCode` and `category` (`Music` or `Sports`) |