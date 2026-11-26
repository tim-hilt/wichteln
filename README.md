# Wichteln (Secret Santa)

A small server-rendered Go app for self-hosting. There are no accounts: everyone gets access through secret links.

## Running

Images for `linux/amd64` and `linux/arm64` are published to `ghcr.io/tim-hilt/wichteln` on every push to `main` (tags `latest` and `sha-<commit>`).

```yaml
services:
  wichteln:
    image: ghcr.io/tim-hilt/wichteln:latest
    restart: unless-stopped
    volumes:
      - ./wichteln:/data
    environment:
      BASE_URL: https://wichteln.example.com
      SMTP_HOST: smtp.resend.com
      SMTP_USER: resend
      SMTP_PASSWORD: ${WICHTELN_SMTP_PASSWORD}
      MAIL_FROM: Wichteln <wichteln@example.com>
```

### Configuration

| Variable        | Default                 | Meaning                                                                 |
|-----------------|-------------------------|-------------------------------------------------------------------------|
| `LISTEN_ADDR`   | `:8080`                 | Address the HTTP server listens on                                      |
| `DATA_DIR`      | `/data` (image), `data` | Where Sessions are stored                                               |
| `BASE_URL`      | `http://localhost:8080` | Public URL of the app, used for links in emails                         |
| `SMTP_HOST`     | –                       | SMTP server. Unset: emails are printed to stdout instead of sent        |
| `SMTP_PORT`     | `465`                   | `465` uses implicit TLS; other ports use STARTTLS if offered            |
| `SMTP_USER`     | –                       | SMTP username (no authentication if empty)                              |
| `SMTP_PASSWORD` | –                       | SMTP password                                                           |
| `MAIL_FROM`     | –                       | Sender, e.g. `Wichteln <wichteln@example.com>`. Required with SMTP_HOST |

`BASE_URL` must be set explicitly: the app does not trust the request's `Host` header for links it sends by email.

### Email via Resend

Any SMTP provider works; with [Resend](https://resend.com) (free tier: 3,000 mails/month, 100/day):

1. Add your domain under *Domains* and create the DNS records Resend shows (DKIM, SPF and a bounce MX on the `send` subdomain). With Cloudflare DNS, Resend can add them automatically. Wait until the domain is *verified*.
2. Create an API key with *Sending access*, restricted to that domain.
3. Configure `SMTP_HOST=smtp.resend.com`, `SMTP_USER=resend`, `SMTP_PASSWORD=<API key>` and `MAIL_FROM` with an address on the verified domain. The address needs no mailbox.
