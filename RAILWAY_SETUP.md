# Railway Setup Guide — School Management Platform

> Step-by-step instructions for provisioning the production (and staging) environment on Railway.
> Follow every section in order. Do not skip steps.

---

## Prerequisites

- [Railway account](https://railway.app) (Hobby plan minimum — $5/month)
- [Railway CLI](https://docs.railway.app/develop/cli) installed: `npm install -g @railway/cli`
- Cloudinary account (free tier is fine to start)
- Resend account for email (free tier: 3,000 emails/month)
- Sentry account (free tier: 5,000 errors/month)
- GitHub repository with this codebase pushed to it

---

## 1. Create the Railway Project

```bash
# Log in to Railway CLI
railway login

# Create a new project
railway init

# Name it: school-platform-production (or school-platform-staging for staging)
```

Alternatively, create the project from the [Railway dashboard](https://railway.app/new).

---

## 2. Provision PostgreSQL

In the Railway dashboard → your project → **+ New Service** → **Database** → **PostgreSQL**

- Railway auto-provisions PostgreSQL 16
- Railway automatically injects `DATABASE_URL` into services that reference it (or copy it manually)
- Note the **internal** `DATABASE_URL` — this is what the Go service uses (private network, faster, no egress cost)

---

## 3. Provision Redis

In the Railway dashboard → your project → **+ New Service** → **Database** → **Redis**

- Railway auto-provisions Redis 7
- Note the **internal** `REDIS_URL`
- **Important:** Use the Hobby paid Redis plan in production — the free tier has memory limits that cause silent job drops in asynq

---

## 4. Deploy the Go Service

```bash
# In the repository root
railway link         # Link local repo to the Railway project
railway up           # Deploy (Railway detects Dockerfile automatically)
```

Or connect your GitHub repo in the Railway dashboard for auto-deploys on push.

The `railway.toml` file is pre-configured for Dockerfile builds and health checks:
```toml
[build]
builder = "DOCKERFILE"
dockerfilePath = "Dockerfile"

[deploy]
healthcheckPath = "/health"
healthcheckTimeout = 60
restartPolicyType = "ON_FAILURE"
restartPolicyMaxRetries = 3
```

---

## 5. Set Environment Variables

In the Railway dashboard → Go service → **Variables** tab, add every variable below.

### Required (system will not start without these)

| Variable | Value / Notes |
|---|---|
| `DATABASE_URL` | Copy from Railway PostgreSQL service → **Variables** → `DATABASE_URL = postgresql://postgres:yrvWbxVRgoELHLtEUWfjUtyEbYpPXdMM@postgres.railway.internal:5432/railway` (use the **internal** URL) |
| `REDIS_URL = redis://default:OMqGYlMxPrBFvttZmSrpFQChHuTxHwCn@redis.railway.internal:6379` | Copy from Railway Redis service → **Variables** → `REDIS_URL` (use the **internal** URL, format: `host:port` without `redis://` prefix if asynq needs it bare) |
| `JWT_SECRET = f3890931a822e964d49bb949a79ec162d11437bad251f33a31a916c466edccef` | Random 256-bit secret. Generate: `openssl rand -hex 32` |
| `JWT_REFRESH_SECRET = 6b4b5a57c9fd3e210f3d2603d02008937f0dda3fb27342775bd0ef550c37d60b` | A different random 256-bit secret. Generate: `openssl rand -hex 32` |
| `CSRF_SECRET = 914d2c2b132e51b6a12ad26e32ef5271c25ade832742071811d3efb535a5009a` | Another random secret. Generate: `openssl rand -hex 32` |
| `APP_BASE_URL = leaps-edu.up.railway.app` | Your production URL, e.g. `https://yourschool.com` — used in QR codes. Use the Railway-generated URL until you set up a custom domain. |
| `ENVIRONMENT` | `production` |
| `CHROMIUM_PATH` | `/usr/bin/chromium` (set by the Dockerfile — Chromium is installed at this path in the container) |

### Cloudinary (file storage — all uploaded files and generated PDFs)

| Variable | Value / Notes |
|---|---|
| `CLOUDINARY_CLOUD_NAME = df4natxfq` | Your Cloudinary cloud name (from dashboard) |
| `CLOUDINARY_API_KEY = 513643525777182` | Your Cloudinary API key |
| `CLOUDINARY_API_SECRET = jmtq4DdyBXdhrXAJhkhjGRn_UFE` | Your Cloudinary API secret |
| `CLOUDINARY_UPLOAD_PRESET = schoolmgt` | Name of your signed upload preset (create in Cloudinary dashboard → Settings → Upload → Upload presets) |

### Email (OTP and verification)

| Variable | Value / Notes |
|---|---|
| `RESEND_API_KEY = re_QPW2KnQQ_CHZzvvAheeVAqdknivEkfpZY` | API key from [Resend dashboard](https://resend.com) — `re_xxxx...` |

### Error Tracking

| Variable | Value / Notes |
|---|---|
| `SENTRY_DSN = https://0834e3e2b6e3079346ee1369287b61c6@o4511639963959296.ingest.de.sentry.io/4511640226365520` | DSN from [Sentry dashboard](https://sentry.io) — create a Go project |

### Rate Limiting (optional — defaults are set in config)

| Variable | Default | Notes |
|---|---|---|
| `RATE_LIMIT_REQUESTS` | `60` | Requests per window for public routes |
| `RATE_LIMIT_WINDOW` | `1m` | Window duration (Go duration format: `1m`, `15m`, `1h`) |

---

## 6. Configure Cloudinary Upload Preset

In Cloudinary dashboard → Settings → Upload → **Add upload preset**:

1. Set **Signing mode** to `Signed`
2. Set **Folder** to `school-platform`
3. Under **Media analysis and AI**, disable unnecessary features
4. Under **Upload control**, set allowed formats:
   - Images preset: `jpg,jpeg,png,webp,gif`
   - Documents preset: `pdf`
   - Videos preset: `mp4,mov,webm`
5. Copy the preset name and set it as `CLOUDINARY_UPLOAD_PRESET`

---

## 7. Set Up GitHub Actions Secrets

In your GitHub repo → **Settings** → **Secrets and variables** → **Actions**, add:

| Secret | Value |
|---|---|
| `RAILWAY_TOKEN` | Railway API token — from [Railway dashboard](https://railway.app/account/tokens) → **Tokens** → **New Token** |
| `PRODUCTION_APP_URL` | Your production URL (same as `APP_BASE_URL`) |

The CI/CD pipeline (`.github/workflows/deploy.yml`) uses these to auto-deploy on merge to `main`.

---

## 8. Custom Domain (optional but recommended)

1. Purchase a domain (Namecheap, Cloudflare Registrar, etc.)
2. Add the domain to **Cloudflare** — set Cloudflare as the nameserver for the domain
3. In Railway dashboard → Go service → **Settings** → **Domains** → **Add Custom Domain**
4. Railway gives you a CNAME target (e.g. `some-name.up.railway.app`)
5. In Cloudflare DNS → **Add record**:
   - Type: `CNAME`
   - Name: `@` (or `www`)
   - Target: the Railway CNAME
   - Proxy status: **Proxied** (orange cloud) ← essential for DDoS protection and caching
6. In Cloudflare SSL/TLS → set mode to **Full (Strict)**
7. Update `APP_BASE_URL` on Railway to the new domain (e.g. `https://yourschool.com`)

---

## 9. Add Cloudflare Cache Rules

In Cloudflare dashboard → your domain → **Rules** → **Cache Rules** → **Create rule**:

**Rule 1 — Never cache API responses:**
- Match: URI path starts with `/api`
- Cache: Bypass cache

**Rule 2 — Cache static assets forever:**
- Match: URI path starts with `/static`
- Cache: Override cache — Edge TTL: 1 year; Browser TTL: 1 year

---

## 10. Verify the Deployment

```bash
# Check the health endpoint
curl https://yourschool.com/health
# Expected: {"status":"ok","env":"production"}

# Check the default owner account works
# Email: owner@graceacademy.test
# Password: ChangeMe!2026
# IMPORTANT: Change this password immediately after first login
```

---

## 11. First Login — Change Default Credentials

The database seed creates a default Owner account:
- **Email:** `owner@graceacademy.test`
- **Password:** `ChangeMe!2026`

**Immediately after deployment:**
1. Log in with the default credentials
2. Go to Settings → Profile → Change password
3. Update the school name, logo, and motto in Settings → School

---

## Staging Environment

Repeat steps 1–9 in a **separate Railway project** named `school-platform-staging`.

In GitHub Actions, PRs auto-deploy to staging (see `.github/workflows/deploy.yml`).

Staging secrets to add to GitHub (under a `staging` environment in GitHub Settings → Environments):
- `STAGING_RAILWAY_TOKEN` — separate Railway token scoped to the staging project
- `STAGING_APP_URL` — the Railway-generated staging URL

---

## Asynq Queue Monitoring (optional)

[asynqmon](https://github.com/hibiken/asynqmon) provides a web dashboard for monitoring queue health, failed jobs, and retry status.

To add it as a Railway service:
```bash
# In the Railway dashboard → + New Service → Docker Image
# Image: hibiken/asynqmon
# Environment variable: REDIS_URL=<internal redis URL>
# Set an auth header or restrict to internal access only
```

---

## Environment Variable Reference (complete)

```env
# ─── Core ─────────────────────────────────────────────────────────────────────
PORT=8080                          # Set automatically by Railway — do not set manually
DATABASE_URL=                      # Railway PostgreSQL internal URL
REDIS_URL=                         # Railway Redis internal URL (host:port format)
ENVIRONMENT = production             # production | staging | development
APP_BASE_URL=https://yourschool.com

# ─── Auth & Security ──────────────────────────────────────────────────────────
JWT_SECRET=                        # openssl rand -hex 32
JWT_REFRESH_SECRET=                # openssl rand -hex 32 (must differ from JWT_SECRET)
CSRF_SECRET=                       # openssl rand -hex 32

# ─── PDF Generation ───────────────────────────────────────────────────────────
CHROMIUM_PATH=/usr/bin/chromium    # Pre-installed in Docker image

# ─── Cloudinary ───────────────────────────────────────────────────────────────
CLOUDINARY_CLOUD_NAME=
CLOUDINARY_API_KEY=
CLOUDINARY_API_SECRET=
CLOUDINARY_UPLOAD_PRESET=

# ─── Email ────────────────────────────────────────────────────────────────────
RESEND_API_KEY=

# ─── Error Tracking ───────────────────────────────────────────────────────────
SENTRY_DSN=

# ─── Rate Limiting (optional — defaults are fine) ─────────────────────────────
RATE_LIMIT_REQUESTS=60
RATE_LIMIT_WINDOW=1m
```

---

## Troubleshooting

**Deployment fails at health check:**
- Check Railway logs: `railway logs`
- Common cause: `DATABASE_URL` or `REDIS_URL` not set, or using public URL instead of internal
- Migrations run on startup — if they fail, the server exits before the health check passes

**PDF generation fails in production:**
- Verify `CHROMIUM_PATH=/usr/bin/chromium` is set
- Check Sentry for the error — chromedp logs context timeouts if Chromium is not found
- Chromium is installed in the Dockerfile — rebuild the Railway image if this is a fresh deploy

**Emails not sending:**
- Verify `RESEND_API_KEY` is set and valid
- Check the `email:send` asynq queue for failed jobs (use asynqmon)
- Resend free tier only sends from `onboarding@resend.dev` — add and verify a sending domain in the Resend dashboard for production use

**CSRF errors on form submissions:**
- Verify `CSRF_SECRET` is set and consistent (it must not change between deploys)
- Verify `APP_BASE_URL` matches the actual domain (used in cookie `Domain` attributes)

---

*RAILWAY_SETUP.md — School Management Platform*
*Update this document whenever infrastructure changes are made.*


here is the school logo, the name is Leadership Preparatory Academy - LEAPS, 

location Makurdi, Benue state

motto: building tomorrow's world now