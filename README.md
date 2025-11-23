# V-Mail

![CI](https://github.com/vdavid/vmail/actions/workflows/ci.yml/badge.svg)
![Go Version](https://img.shields.io/badge/go-1.25.3-blue.svg)
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

- Install [Authelia](https://www.authelia.com) and run it. Get its hostname, you'll need it.
- Get your IMAP and SMTP credentials ready.
- Clone this repo by `git clone git@github.com:vdavid/vmail.git && cd vmail`.
- Run `cp .env.example .env` to create a private env file.
- Edit the `.env` file and follow the instructions inside.
- Make sure you have **Docker** and **Docker Compose** installed.
- Run `docker compose up -d --build` to start the services.
- Open `http://localhost:11764` in the browser (or configure port mapping if using Docker).
- Log in with your Authelia credentials.

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
