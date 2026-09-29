# Massmaker

Massmaker is the catalogue and activity service. It owns publication status, contact consent, recommendations, search synonyms, events, and current-state legal/purpose consents.

It stores `enmasse_vendor_id` on makers and `enmasse_artist_id` on recommendations. Vendors who save an overlay also get `enmasse_user_id` (JWT `sub`) on that maker row. It never stores emails or password hashes. Consent rows store pepper-hashed IP and user-agent, never the raw values.

Public search is anonymous. Logged-in explore is gated on computed profile completeness. Vendors save a draft overlay (`/v1/me/maker`) before staff publish. Overlay capabilities are leaf category slugs. Vendors propose new groups/leaves with `POST /v1/me/category-proposals`; public `GET /v1/categories` and search stay published-only until staff publish. `GET /v1/me/vendors` re-checks Enmasse over HTTPS with a service JWT. Drafts 404, withdrawn makers 410. Public maker JSON does not include WhatsApp. A complete artist submits an inquiry (`POST /v1/makers/{id}/inquiries`); Massmaker stores it as an `orders` row in `pending_review`. Vendors cannot contact other makers: the inquiry CTA is hidden for a vendor session.

`POST /v1/consents` upserts one current-state row per subject and type (`granted_at` / `withdrawn_at` are server `NOW()` only). Guests may grant `activity_tracking` and `marketing`. Document ticks and role purposes require a JWT. `GET /v1/me/consents` returns the signed-in user's rows. Event ingest refuses unless `activity_tracking` is currently granted for that session or user.

**Ready for review**:
A vendor draft that has PIC/WhatsApp/street address/province/city/district/bio on Enmasse plus category, price, MOQ, lead time, and hours (open/close and weekdays) on Massmaker. Portfolio photos are optional. It is not a public listing.
_Avoid_: Published, live

## Language

**Session**:
A tab-scoped visit identified by `anon_session_id`. It is not a person.
_Avoid_: Unique visitor, unique user

**Activity**:
A `search`, `profile_view`, or `contact_click` stored on Massmaker, bound to a session and optionally to an Enmasse user id from the JWT. Ingest requires a current `activity_tracking` grant.
_Avoid_: Analytics event (jicaf-id / Supabase), identity event

**Consent**:
The current grant or withdrawal for one purpose or legal document (`activity_tracking`, `marketing`, `whatsapp_publication`, `recommendation_publication`, `privacy_policy`, `user_agreement`, `terms`). One row per subject and type; timestamps come from Massmaker Postgres.
_Avoid_: Cookie banner, analytics opt-in

**Counted contact**:
The first `contact_click` for a session and maker in 30 minutes. Repeats are stored with `counted=false`.
_Avoid_: Unique click, unique person

**Order**:
An artist production inquiry stored on Massmaker. Submit creates `pending_review`. The targeted vendor accepts or declines; the artist may cancel while pending. Payment stays off-platform.
_Avoid_: Checkout, WhatsApp message, shop cart
