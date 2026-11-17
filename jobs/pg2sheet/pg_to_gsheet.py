#!/usr/bin/env python3
"""
Sync makers from Postgres to a Google Sheet (profile cluster on one row).

Config from environment (see .env.example):
  PG_HOST, PG_PORT, PG_DB, PG_USER, PG_PASSWORD
  SQL_QUERY            optional; defaults to makers + contacts + aggregates
  SHEET_ID             Google Sheet ID from the URL
  SHEET_TAB            worksheet/tab name (default: makers)
  GOOGLE_CREDS_FILE    path to service-account JSON key
"""
from __future__ import annotations

import logging
import os
import sys
import uuid
from datetime import date, datetime, time, timezone
from decimal import Decimal

import gspread
import psycopg2

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
)
log = logging.getLogger("pg2sheet")

DEFAULT_SQL = """
SELECT
  m.id,
  m.name,
  m.city,
  m.publication_status,
  m.verification_date,
  m.moq_quantity,
  m.moq_basis,
  m.moq_basis_note,
  m.lead_time_estimate,
  m.price_min_idr,
  m.price_max_idr,
  m.price_note,
  m.conditions,
  m.is_demo_fixture,
  m.enmasse_vendor_id,
  m.enmasse_user_id,
  m.hours_open,
  m.hours_close,
  array_to_string(m.hours_days, ', ') AS hours_days,
  m.service_area,
  m.jicaf_collaborated_before,
  m.jicaf_collaboration_artist,
  m.jicaf_collaboration_year,
  m.created_at,
  m.updated_at,
  c.channel AS contact_channel,
  c.e164_digits AS contact_e164,
  c.publish_permitted AS contact_publish_permitted,
  c.permission_at AS contact_permission_at,
  cap.category_slugs,
  cap.category_labels,
  pw.portfolio_work_count,
  pw.portfolio_titles,
  media.media_count,
  media.media_urls,
  rec.recommendation_count,
  rec.recommendation_artists,
  rec.recommendation_products
FROM makers m
LEFT JOIN maker_contacts c ON c.maker_id = m.id
LEFT JOIN LATERAL (
  SELECT
    string_agg(cat.slug, ', ' ORDER BY cat.slug) AS category_slugs,
    string_agg(cat.label_id, ', ' ORDER BY cat.slug) AS category_labels
  FROM maker_capabilities mc
  JOIN categories cat ON cat.slug = mc.category_slug
  WHERE mc.maker_id = m.id
) cap ON true
LEFT JOIN LATERAL (
  SELECT
    count(*)::int AS portfolio_work_count,
    string_agg(w.title, ' | ' ORDER BY w.sort_order, w.id) AS portfolio_titles
  FROM maker_portfolio_works w
  WHERE w.maker_id = m.id
) pw ON true
LEFT JOIN LATERAL (
  SELECT
    count(*)::int AS media_count,
    string_agg(mm.url, ' | ' ORDER BY w.sort_order, mm.sort_order, mm.id) AS media_urls
  FROM maker_media mm
  JOIN maker_portfolio_works w ON w.id = mm.work_id
  WHERE mm.maker_id = m.id
) media ON true
LEFT JOIN LATERAL (
  SELECT
    count(*)::int AS recommendation_count,
    string_agg(a.display_name, ' | ' ORDER BY r.created_at, r.id) AS recommendation_artists,
    string_agg(r.product_made, ' | ' ORDER BY r.created_at, r.id) AS recommendation_products
  FROM recommendations r
  JOIN artists a ON a.id = r.artist_id
  WHERE r.maker_id = m.id
) rec ON true
ORDER BY m.created_at
""".strip()


def clean(value):
    """Convert DB values into something Google Sheets accepts."""
    if value is None:
        return ""
    if isinstance(value, uuid.UUID):
        return str(value)
    if isinstance(value, datetime):
        return value.isoformat(sep=" ")
    if isinstance(value, (date, time)):
        return value.isoformat()
    if isinstance(value, Decimal):
        return float(value)
    if isinstance(value, (dict, list, bytes, memoryview)):
        return str(value)
    return value


def fetch_rows():
    conn = psycopg2.connect(
        host=os.environ.get("PG_HOST", "127.0.0.1"),
        port=os.environ.get("PG_PORT", "5432"),
        dbname=os.environ["PG_DB"],
        user=os.environ["PG_USER"],
        password=os.environ["PG_PASSWORD"],
        connect_timeout=15,
    )
    try:
        with conn.cursor() as cur:
            cur.execute(os.environ.get("SQL_QUERY", DEFAULT_SQL))
            headers = [d[0] for d in cur.description]
            rows = [[clean(v) for v in r] for r in cur.fetchall()]
        return headers, rows
    finally:
        conn.close()


def write_sheet(headers, rows):
    sheet_id = os.environ.get("SHEET_ID", "").strip()
    if not sheet_id:
        raise RuntimeError("SHEET_ID / PG2SHEET_SHEET_ID is not set")
    creds = os.environ.get("GOOGLE_CREDS_FILE", "").strip()
    if not creds:
        raise RuntimeError("GOOGLE_CREDS_FILE is not set")

    gc = gspread.service_account(filename=creds)
    sh = gc.open_by_key(sheet_id)
    tab = os.environ.get("SHEET_TAB", "makers")

    try:
        ws = sh.worksheet(tab)
    except gspread.WorksheetNotFound:
        ws = sh.add_worksheet(title=tab, rows=1000, cols=max(len(headers), 10))

    data = [headers] + rows
    ws.clear()

    chunk = 5000
    for i in range(0, len(data), chunk):
        ws.update(
            values=data[i : i + chunk],
            range_name=f"A{i + 1}",
            value_input_option="USER_ENTERED",
        )

    try:
        meta = sh.worksheet("_sync")
    except gspread.WorksheetNotFound:
        meta = sh.add_worksheet(title="_sync", rows=5, cols=3)
    meta.update(
        values=[
            [
                "last_sync_utc",
                datetime.now(timezone.utc).replace(tzinfo=None).isoformat(),
                f"{len(rows)} rows",
            ]
        ],
        range_name="A1",
    )


def main():
    try:
        headers, rows = fetch_rows()
        log.info("Fetched %d rows", len(rows))
        write_sheet(headers, rows)
        log.info("Sheet updated")
    except Exception:
        log.exception("Sync failed")
        sys.exit(1)


if __name__ == "__main__":
    main()
