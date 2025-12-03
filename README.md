# V-Mail

![CI](https://github.com/vdavid/vmail/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/badge/go-1.25.5-blue.svg)
![Node.js](https://img.shields.io/badge/node-%3E%3D25.0.0-brightgreen.svg)
![TypeScript](https://img.shields.io/badge/typescript-5.9-blue.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/vdavid/vmail/backend)](https://goreportcard.com/report/github.com/vdavid/vmail/backend)
![License](https://img.shields.io/github/license/vdavid/vmail)

A fast, self-hosted webmail client for those who love Gmail's layout and its keyboard shortcuts.

## Overview

V-Mail is a self-hosted, web-based email client designed for personal use.
It uses the layout and keyboard shortcuts of Gmail to make it immediately familiar for ex-Gmail users.
It connects to an IMAP server and provides the web UI to read and send email.

We built V-Mail with the explicit legal constraint to **not** use any of Google's proprietary assets (fonts, icons,
logos) or aesthetic design. The focus is on **functional parity** while avoiding visual imitation, to avoid any brand
confusion.

![vmail-ui-draft-scrshot](https://github.com/user-attachments/assets/d003da28-ce02-4307-ba74-6fb0fac86dc6)

## Running

### Prerequisites

- **Docker and Docker Compose** installed
- **A reverse proxy** (Caddy, Nginx, or Traefik) for HTTPS
- Your **IMAP and SMTP credentials** ready

### Option A: Bundled Authelia (easiest)

V-Mail can run its own Authelia instance. This is the simplest setup if you don't already have Authelia.

```bash
git clone git@github.com:vdavid/vmail.git && cd vmail

# Set up V-Mail config
cp .env.example .env
# Edit .env - see comments inside

# Set up Authelia config
cp config/authelia/configuration.yml.example config/authelia/configuration.yml
cp config/authelia/users.yml.example config/authelia/users.yml
# Edit both files - generate secrets and add your user (see comments inside)

# Start everything
docker compose -f docker-compose.yml -f docker-compose.with-authelia.yml up -d --build
```

Then configure your reverse proxy (example for Caddy):

```
# V-Mail (protected by Authelia)
mail.example.com {
    forward_auth localhost:9091 {
        uri /api/verify?rd=https://auth.example.com
        copy_headers Remote-User Remote-Email
    }
    reverse_proxy localhost:40941
}

# Authelia login portal
auth.example.com {
    reverse_proxy localhost:9091
}
```

### Option B: Bring your own Authelia

If you already have Authelia (or another forward_auth-compatible solution like Authentik), use the base compose file:

```bash
git clone git@github.com:vdavid/vmail.git && cd vmail
cp .env.example .env
# Edit .env - set VMAIL_AUTH_MODE=header and AUTHELIA_URL
docker compose up -d --build
```

Configure your reverse proxy to do `forward_auth` to your existing Authelia instance.

### Reverse proxy setup

V-Mail uses Authelia for authentication via your reverse proxy's `forward_auth` feature.
The reverse proxy intercepts requests, checks with Authelia, and passes auth headers to V-Mail.

**Caddy example:**

```
mail.example.com {
    forward_auth authelia:9091 {
        uri /api/verify?rd=https://auth.example.com
        copy_headers Remote-User Remote-Email
    }
    reverse_proxy localhost:40941
}
```

For Nginx, see [Authelia's Nginx docs](https://www.authelia.com/integration/proxies/nginx/).

**Important notes:**
- HTTPS is required (Authelia cookies need secure context)
- Never expose V-Mail directly to the internet without the reverse proxy
- The header-based auth trusts `Remote-User`/`Remote-Email` headers from your proxy

## Tech stack

V-Mail uses a **Postgres** database, a **Go** back end, a **REST** API, and a **React** front end with **TypeScript**.
V-Mail needs a separate, self-hosted [Authelia](https://www.authelia.com) (an
[open-source](https://github.com/authelia/authelia), Go-based SSO and 2FA server) instance for authentication.

### IMAP server

V-Mail works with modern IMAP servers, **[mailcow](https://mailcow.email/)** (using Dovecot under the hood) being the
primary target.
It has two **hard requirements** for the IMAP server:

1. **`THREAD` Extension ([RFC 5256](https://datatracker.ietf.org/doc/html/rfc5256)):** Server-side threading is
   mandatory. V-Mail will not implement client-side threading.
2. **Full-Text Search (FTS):** The server must support fast, server-side `SEARCH` commands.
   Standard IMAP `SEARCH` is part of the core protocol, but V-Mail's performance relies on the server's FTS
   capabilities, like those in Dovecot.

## Security

We designed the project with security in mind.
However, when self-hosting the project, you are responsible for regularly backing up the database to avoid data loss.
The emails themselves live on the IMAP server, but offline drafts and settings are stored in V-Mail.

## Keyboard shortcuts

The app provides a subset of Gmail's shortcuts:

* **Navigation:**
    * `j` / `↓`: Move cursor to next email in list / next message in thread.
    * `k` / `↑`: Move cursor to previous email in list / previous message in thread.
    * `o` / `Enter`: Open the selected thread.
    * `u`: Go back to the list view (from a thread).
    * `g` then `i`: Go to inbox.
    * `g` then `s`: Go to starred.
    * `g` then `t`: Go to sent.
    * `g` then `d`: Go to drafts.
* **Actions:**
    * `c`: Compose new email.
    * `r`: Reply (to sender).
    * `a`: Reply all.
    * `f`: Forward.
    * `e`: Archive selected.
    * `s`: Star/unstar selected.
    * `#` (Shift+3): Move to trash (delete).
    * `z`: Undo last action.
    * `/`: Focus search bar.
* **Selection (in list view):**
    * `x`: Toggle selection on the focused email.
    * `*` then `a`: Select all.
    * `*` then `n`: Select none.
    * `*` then `r`: Select read.
    * `*` then `u`: Select unread.
    * `*` then `s`: Select starred.
    * `*` then `t`: Select unstarred.

## Contributing

Contributions are welcome!
See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

Report issues and feature requests in the [issue tracker](https://github.com/vdavid/vmail/issues).

Happy emailing!

David
