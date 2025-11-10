# V-Mail

![CI](https://github.com/vdavid/vmail/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/badge/go-1.25.1-blue.svg)
![Node.js](https://img.shields.io/badge/node-%3E%3D25.0.0-brightgreen.svg)
![TypeScript](https://img.shields.io/badge/typescript-5.9-blue.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/vdavid/vmail/backend)](https://goreportcard.com/report/github.com/vdavid/vmail/backend)
![License](https://img.shields.io/github/license/vdavid/vmail)

A fast, self-hosted webmail client with a familiar, keyboard-driven UI.

## Overview

V-Mail is a self-hosted, web-based email client designed for personal use.
It uses the layout and keyboard shortcuts of Gmail to make it immediately familiar for ex-Gmail users.
It connects to an IMAP server (tested with mailcow) and provides the web UI to read and send email.

I built V-Mail with the explicit legal constraint to **not** use any of Google's proprietary assets (fonts, icons, logos)
or aesthetic design. The focus is on **functional parity**, not visual imitation, to avoid any brand confusion.

## Running

- Install [Authelia](https://www.authelia.com) and run it. Get its hostname, you'll need it.
- Get your IMAP and SMTP credentials ready.
- Clone this repo by `git clone git@github.com:vdavid/vmail.git && cd vmail`.
- Run `cp .env.example .env` to create a private env file.
- Edit the `.env` file and follow the instructions inside.
- Make sure you have **Docker** and **Docker Compose** installed.
- Run `docker compose up -d --build` to start the services.
- Open `http://localhost:8080` in the browser.
- Log in with your Authelia credentials.

## Non-goals

Compared to Gmail, this project does **not** include:

* Client-side email filters. The user should set these up on the server, typically via [Sieve](http://sieve.info/).
* A visual query builder for the search box. A simple text field is fine.
* A multi-language UI. The UI is English-only.
* 95% of Gmail's settings. V-Mail has some basic settings like emails per page and undo send delay, but that's it.
* Automatic categorization such as primary/social/promotions.
* The ability to collapse the left sidebar.
* Signature management.
* Smiley/emoji reactions to emails. This is Google's proprietary thing.

## Tech stack

V-Mail uses a **Go** back end, a **REST** API, and a **React** front end with **TypeScript**.
It uses a **Postgres** database for caching, drafts, and settings.
V-Mail does **not** handle authentication. A separate, self-hosted [Authelia](https://www.authelia.com) instance is responsible for that.

### IMAP server

V-Mail works with modern IMAP servers, **mailcow** (using Dovecot under the hood) being the primary target.
It has two **hard requirements** for the IMAP server:

1.  **`THREAD` Extension ([RFC 5256](https://datatracker.ietf.org/doc/html/rfc5256)):** Server-side threading is mandatory.
    V-Mail will not implement client-side threading.
2.  **Full-Text Search (FTS):** The server must support fast, server-side `SEARCH` commands.
    Standard IMAP `SEARCH` is part of the core protocol, but V-Mail's performance relies on the server's FTS capabilities,
    like those in Dovecot.

### Authelia

**Authelia** ([authelia.com](https://www.authelia.com/)) is responsible for authentication.
It's an [open-source](https://github.com/authelia/authelia), Go-based single sign-on (SSO) and 2FA server.

**Interaction flow:** The V-Mail front end redirects the user to Authelia for login.
After successful login, Authelia provides a session token, a JWT, which the front end stores in the browser.
After this, all API requests from the front end to the Go back end will include this token.
The back end validates the token before processing requests.

## Security

We designed the project with security in mind.
However, you are responsible for regularly backing up the database to avoid data loss. The emails themselves
live on the IMAP server, but offline drafts and settings are in the database.

## Keyboard shortcuts

We designed the app to be fully usable via a subset of Gmail's shortcuts.

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

