# V-Mail

## Overview

V-Mail is a self-hosted, web-based email client designed as a functional alternative to Gmail with familiar keyboard-driven navigation. The application connects to IMAP servers (primarily mailcow/Dovecot) to provide a fast, personal email interface. It explicitly focuses on functional parity with Gmail's core features while avoiding visual imitation and brand confusion.

## User Preferences

Preferred communication style: Simple, everyday language.

## Current Status

**Milestone 1: IMAP Spike - COMPLETE ✓**

A CLI proof-of-concept has been successfully implemented in `backend/cmd/spike/` that demonstrates all core IMAP operations:
- TLS connection to IMAP servers
- Authentication with username/password
- Capability detection (verifies THREAD support)
- Inbox selection and mailbox status retrieval
- THREAD command execution (RFC 5256)
- SEARCH command for finding messages
- FETCH command for retrieving message details

The spike includes comprehensive error handling and unit tests. It requires IMAP credentials to run against a real server.

To run the spike:
1. Copy `backend/.env.example` to `backend/.env`
2. Fill in your IMAP server credentials
3. Run: `cd backend && go run ./cmd/spike`

See `backend/README.md` for detailed documentation.

**Milestone 2 Part 2/3: Frontend Skeleton and Settings Page - COMPLETE ✓**

The frontend foundation has been successfully implemented with:
- React 19 with TypeScript and Vite build system
- Tailwind CSS v4 for styling
- React Router for navigation (routes: /, /thread/:id, /settings)
- TanStack Query for server state management
- Zustand for client state management
- Comprehensive component architecture:
  - Layout component with Sidebar and Header
  - AuthWrapper for onboarding flow
  - Settings page with full IMAP/SMTP configuration
  - Placeholder Inbox and Thread pages
- Complete test coverage (31 tests passing) using Vitest and React Testing Library
- Properly configured for Replit environment (dev server on port 5000)

To run the frontend:
1. The frontend workflow is already configured and running on port 5000
2. Access it via the Webview at port 5000
3. Run tests: `cd frontend && pnpm test`

**Note**: The frontend is configured to proxy `/api/*` requests to the Go backend at `http://localhost:8080`. For the full application to work:
1. The backend server needs to be running on port 8080
2. The backend requires environment variables to be set (see `.env.example`)
3. The Postgres database must be accessible

The frontend connects to the Go backend API at `/api/v1/*` endpoints and includes proper error handling, form validation, and loading states. The Vite dev server proxies these requests to the backend automatically.

## System Architecture

### Authentication Strategy
- **External authentication provider**: Uses Authelia for all user authentication and authorization
- **Rationale**: Separates security concerns from application logic, leveraging a dedicated, battle-tested authentication service
- **Token-based flow**: Frontend receives auth cookie/token from Authelia, passes it with API requests
- **No built-in auth**: V-Mail intentionally does not handle user credentials or session management

### Frontend Architecture
- **Framework**: React with TypeScript
- **Hosting**: Served via Docker container
- **API Communication**: REST API calls to Go backend with Authelia token
- **UI Philosophy**: Keyboard-driven interface mimicking Gmail's functional shortcuts without copying visual design

### Backend Architecture
- **Language**: Go
- **API Style**: REST
- **IMAP Integration**: Direct IMAP client connection to mail servers
  - **Hard requirement**: IMAP server must support THREAD extension (RFC 5256) for server-side email threading
  - **Primary target**: mailcow (Dovecot) servers
  - **Operations**: Connect via TLS, authenticate, search, fetch messages, thread conversations
- **Design principle**: Backend acts as a bridge between IMAP servers and frontend, handling email operations and caching

### Data Storage
- **Database**: Postgres
- **Usage patterns**:
  - Caching email data for performance
  - Storing draft messages
  - Managing user settings (e.g., emails per page, undo send delay)
- **Not used for**: User authentication or credentials

### Deployment Architecture
- **Containerization**: Docker Compose orchestration
- **Services**:
  - Authelia (authentication)
  - React frontend container
  - Go API container
  - Postgres database container
  - Integration with external mailcow server
- **Networking**: Services communicate within Docker network, with exposed ports for browser access

## External Dependencies

### Required Services
- **Authelia**: Self-hosted authentication and authorization service
  - Handles all user login flows
  - Issues auth tokens/cookies for API requests
  - Must be installed and configured separately

- **IMAP Server**: Email server with specific requirements
  - Must support THREAD extension (RFC 5256)
  - Tested with mailcow/Dovecot
  - Requires IMAP and SMTP credentials

- **SMTP Server**: For sending emails
  - Typically, the same server as IMAP
  - Credentials configured via environment variables

### Database
- **Postgres**: Primary data store
  - Connection configurable via `VMAIL_DB_HOST` environment variable
  - Runs in Docker container by default
  - Can be exposed to host machine (port 5432) for local development

### Development Tools
- **Docker**: Container runtime
- **Docker Compose**: Service orchestration
- **Go 1.24+**: Backend development
- **Environment Variables**: Configuration via `.env` file
  - IMAP/SMTP server details
  - Database connection strings
  - Authelia instance hostname