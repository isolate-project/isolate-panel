# Phase 10: Notification System

## Overview
Implement comprehensive notification system for Isolate Panel with Webhook and Telegram support.

## Decisions

| Aspect | Decision |
|--------|----------|
| Storage | Last N=100 notifications, no grouping |
| Email (SMTP) | Post-MVP |
| Webhook | ✅ Full implementation with HMAC-SHA256 signature |
| Telegram | ✅ Full implementation (not stub) |
| Retry | Log failures to system_logs only |
| Recipients | All notifications to all admins |
| Integration | NotificationService with direct methods |

## Notification Events

| Event | Trigger | Severity | Metadata |
|-------|---------|----------|----------|
| `quota_exceeded` | User exceeded traffic limit | warning | user_id, username, quota_bytes, used_bytes |
| `expiry_warning` | User expires in 7/3/1 days | warning | user_id, username, expiry_date, days_left |
| `cert_renewed` | TLS certificate renewed | info | cert_id, domain, expires_at |
| `core_error` | Core crashed/failed | critical | core_name, error_message |
| `failed_login` | 5+ failed login attempts from IP | warning | ip_address, username, attempt_count |
| `user_created` | New user created | info | user_id, username |
| `user_deleted` | User deleted | info | user_id, username |

## Database Schema

```sql
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at DATETIME,
    sent_at DATETIME,
    error TEXT,
    metadata TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Webhook Payload

```json
{
  "event_type": "quota_exceeded",
  "severity": "warning",
  "title": "User quota exceeded",
  "message": "User john exceeded 100GB quota",
  "timestamp": "2026-03-24T15:00:00Z",
  "panel_url": "https://panel.example.com",
  "metadata": {
    "user_id": 42,
    "username": "john",
    "quota_bytes": 107374182400,
    "used_bytes": 110000000000
  }
}
```

**HMAC Signature:**
- Header: `X-Isolate-Panel-Signature: sha256=<hmac>`
- Algorithm: HMAC-SHA256(body + secret)
- Timeout: 10 seconds

## Implementation Files

### Backend
- `backend/internal/models/notification.go`
- `backend/internal/services/notification_service.go`
- `backend/internal/services/webhook_notifier.go`
- `backend/internal/services/telegram_notifier.go`
- `backend/internal/api/notifications.go`
- `backend/internal/database/migrations/000026_create_notifications_table.sql`

### Frontend
- `frontend/src/pages/Notifications.tsx`
- `frontend/src/api/endpoints/index.ts` (add notificationApi)
- `frontend/src/i18n/locales/*.json` (add translations)

## API Endpoints

```
GET    /api/notifications              # List notifications
GET    /api/notifications/:id          # Get notification details
POST   /api/notifications/mark-read    # Mark as read
DELETE /api/notifications/:id          # Delete notification
POST   /api/notifications/test         # Send test notification
GET    /api/notifications/settings     # Get notification settings
PUT    /api/notifications/settings     # Update notification settings
```

## Integration Points

| Service | Notification Method |
|---------|---------------------|
| QuotaEnforcer | NotifyQuotaExceeded(user) |
| UserService | NotifyExpiryWarning(user, days) |
| UserService.Create | NotifyUserCreated(user) |
| UserService.Delete | NotifyUserDeleted(user) |
| CertificateService | NotifyCertRenewed(cert) |
| CoreLifecycleManager | NotifyCoreError(name, err) |
| AuthHandler.Login | NotifyFailedLogin(ip, username, count) |

## Tasks

- [ ] 10.1: Notification model + migration
- [ ] 10.2: NotificationService (core)
- [ ] 10.3: WebhookNotifier (HMAC)
- [ ] 10.4: TelegramNotifier (full)
- [ ] 10.5: API endpoints
- [ ] 10.6: Integration with 7 services
- [ ] 10.7: Frontend Notifications page
- [ ] 10.8: i18n translations (en/ru/zh)
- [ ] 10.9: Tests + build + commit

## Deliverables

1. Webhook notifications working
2. Telegram notifications working
3. All 7 event types implemented
4. UI for notification settings
5. Test notification functionality
