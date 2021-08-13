## Nested
Nested is an email based team communication platform. You have shared
mailboxes and comment on your received posts/emails. Implement your workflow
with labels.

## Docker Images
This project will create docker image in `registry.ronaksoft.com`:
docker pull registry.ronaksoft.com/nested/server:1.0

### Configurations
| KEY | DEFAULT | DESC |
| --- | --- | --- |
| DOMAINS | |
| SENDER_DOMAIN | |
| BUNDLE_ID | |
| BIND_PORT | |
| BIND_IP | |
| CYRUS_URL | |
| TLS_KEY_FILE | |
| TLS_CERT_FILE | |
| JOB_ADDRESS | |
| MONGO_TLS | |
| MONGO_DSN | |
| REDIS_DSN | |
| LOG_LEVEL | |
| ADP_MESSAGE_URL | |
| ADP_USERNAME | |
| ADP_PASSWORD | |
| MONITOR_ACCESS_TOKEN | |
| SYSTEM_API_KEY | |
| SMTP_USER | |
| SMTP_PASS | |
| SMTP_HOST | |
| SMTP_PORT | |
| INSTANCE_ID | |
| WEBAPP_BASE_URL | |
| POSTFIX_CHROOT | |
| MAIL_STORE_SOCK | |
| MAIL_UPLOAD_BASE_URL | |
| MAILER_DAEMON | |
| FIREBASE_CRED_PATH | |


## TODOs
[ ] Improve documents
[ ] Handle spam management, delete all, mark as spam, ...
[x] Delete all messages in place
[ ] Change send from address.
[ ] Installing and handling popular dns servers. i.e. Cloudflare, ...
[ ] TLS cert generation using Lets Encrypt
[ ] Workflows. i.e. using labels and users interactions.
[ ] Workflow - create / edit / delete / list / trigger / status / events / hooks 
