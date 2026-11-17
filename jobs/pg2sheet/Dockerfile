FROM python:3.12-slim

ARG SUPERCRONIC_VERSION=v0.2.29
ARG TARGETARCH=amd64

RUN apt-get update \
 && apt-get install -y --no-install-recommends tzdata curl ca-certificates \
 && curl -fsSL -o /usr/local/bin/supercronic \
    "https://github.com/aptible/supercronic/releases/download/${SUPERCRONIC_VERSION}/supercronic-linux-${TARGETARCH}" \
 && chmod +x /usr/local/bin/supercronic \
 && apt-get purge -y curl && apt-get autoremove -y \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY pg_to_gsheet.py crontab ./

CMD ["supercronic", "/app/crontab"]
