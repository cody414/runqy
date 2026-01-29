# runqy Monitoring UI v2

Modern monitoring dashboard for runqy built with SvelteKit + Skeleton.dev.

## Tech Stack

- **SvelteKit** - Full-stack framework with SSR/SPA modes
- **Skeleton.dev** - Tailwind-based UI kit for Svelte
- **TypeScript** - Type safety
- **Chart.js** - Lightweight charting (planned)

## Development

```bash
# Install dependencies
npm install

# Start dev server (with proxy to backend at localhost:3000)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## Project Structure

```
src/
├── lib/
│   ├── api/           # API client and TypeScript types
│   ├── components/    # Reusable UI components
│   ├── stores/        # Svelte stores for state
│   └── utils/         # Formatting and utility functions
├── routes/
│   ├── +layout.svelte # App shell with sidebar
│   ├── +page.svelte   # Dashboard
│   ├── queues/        # Queue list and details
│   ├── workers/       # Worker monitoring
│   ├── system/        # Redis and server info
│   └── settings/      # User preferences
└── app.html
```

## Features

- **Dashboard**: Overview of all queues with stats
- **Queues**: List view with search, detail view with tabbed task states
- **Workers**: Card/table view with status filtering
- **System**: Redis info and Asynq server details
- **Settings**: Theme, poll interval, view density

## Integration

The UI consumes the same REST API endpoints as the existing monitoring:

- `GET /api/queues` - List queues
- `GET /api/queues/{qname}` - Queue details
- `GET /api/queues/{qname}/pending_tasks` etc.
- `GET /api/workers` - Workers list
- `GET /api/redis_info` - Redis stats

Build output goes to `build/` directory which can be embedded by Go.
