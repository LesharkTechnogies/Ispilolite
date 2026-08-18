# Talksasa SMS

The application uses the Talksasa v3 JSON API for transactional and administrative SMS.

Configure the server environment:

```text
TALKSASA_API_TOKEN=replace-with-a-rotated-token
TALKSASA_SENDER_ID=TALKSASA
TALKSASA_BASE_URL=https://bulksms.talksasa.com/api/v3
```

The token is never returned to clients or stored in PostgreSQL. The provider sends plain SMS through `/sms/send` and supports hashed transactional SMS through `/sms/send-hashed` for callers that have the Safaricom SHA256 MSISDN value.

Admin messaging is available at `POST /api/v1/admin/sms` with an admin JWT:

```json
{"role":"customer","message":"Scheduled maintenance tonight."}
```

Use `{"all":true,"message":"..."}` for all active users or `{"user_ids":["..."],"message":"..."}` for selected users.
