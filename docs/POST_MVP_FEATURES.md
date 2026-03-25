# Post-MVP Features

Этот документ описывает функции, которые были отложены для реализации после версии MVP (v1.0.0).

---

## Email Notifications

**Статус:** ⏸️ Post-MVP (v1.1.0)

**Причина откладки:** Для MVP достаточно Telegram и Webhook уведомлений. Email требует дополнительной инфраструктуры (SMTP сервер или сторонний сервис).

### Планируемая реализация

#### 1. Email Provider (`internal/services/email_notifier.go`)

```go
package services

import (
    "crypto/tls"
    "fmt"
    "net/smtp"
    "net/textproto"
)

type EmailNotifier struct {
    smtpHost     string
    smtpPort     int
    username     string
    password     string
    fromEmail    string
    useTLS       bool
}

type EmailNotification struct {
    To      []string
    Subject string
    Body    string
    HTML    bool
}

func (e *EmailNotifier) Send(notification EmailNotification) error {
    // SMTP отправка
}
```

#### 2. Конфигурация

**Environment variables:**
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=admin@example.com
SMTP_PASSWORD=app-password
SMTP_FROM=noreply@example.com
SMTP_USE_TLS=true
```

**Database (settings table):**
```sql
INSERT INTO settings (key, value) VALUES 
    ('email_notifications_enabled', 'false'),
    ('email_smtp_host', ''),
    ('email_smtp_port', '587'),
    ('email_from_address', '');
```

#### 3. NotificationSettings Model Update

Добавить в `models/notification.go`:

```go
type NotificationSettings struct {
    // ... existing fields ...
    
    // Email settings
    EmailEnabled      bool   `json:"email_enabled"`
    EmailSMTPHost     string `json:"email_smtp_host"`
    EmailSMTPPort     int    `json:"email_smtp_port"`
    EmailUsername     string `json:"email_username"`
    EmailFromAddress  string `json:"email_from_address"`
}
```

#### 4. API Endpoints

```
POST /api/notifications/settings/email - Configure email settings
POST /api/notifications/test/email - Send test email
```

#### 5. Поддерживаемые сервисы

- ✅ Gmail (через App Password)
- ✅ SendGrid
- ✅ Mailgun
- ✅ Amazon SES
- ✅ Custom SMTP

---

## WARP Full Integration

**Статус:** ✅ Частично реализовано в Phase 8 (MVP)

**Что реализовано в MVP:**
- ✅ Регистрация через Cloudflare API
- ✅ Генерация WireGuard ключей (curve25519)
- ✅ Сохранение аккаунта
- ✅ Генерация WireGuard конфига

**Что осталось для Post-MVP:**
- ⏸️ Автоматическое продление токена (token refresh)
- ⏸️ Интеграция с ядрами (Xray/Sing-box/Mihomo) для маршрутизации через WARP
- ⏸️ Мониторинг статуса подключения
- ⏸️ Переключение между WARP аккаунтами

---

## GeoIP/GeoSite Core Integration

**Статус:** ⏸️ Post-MVP (v1.1.0)

**Что реализовано:**
- ✅ API для загрузки GeoIP/GeoSite баз
- ✅ CRUD для Geo правил
- ✅ Frontend для управления правилами

**Что осталось:**
- ⏸️ Интеграция с конфигурациями ядер
- ⏸️ Автоматическое обновление баз (раз в неделю)
- ⏸️ Кэширование Geo данных

---

## Remote Backup Destinations

**Статус:** ⏸️ Post-MVP (v1.2.0)

**Планируемые провайдеры:**
- Amazon S3
- Google Cloud Storage
- SFTP
- FTP
- WebDAV

---

## SSH Protocol Support

**Статус:** ⏸️ Post-MVP (v1.2.0)

SSH как прокси-протокол (не путать с SSH доступом к VPS).

**Причина откладки:** Требует дополнительной настройки аутентификации и менее востребован.

---

## WireGuard Protocol Support

**Статус:** ⏸️ Post-MVP (v1.2.0)

WireGuard как прокси-протокол для inbound/outbound.

**Причина откладки:** Требует NET_ADMIN capability и создания сетевых интерфейсов.

---

## TProxy/TUN Support

**Статус:** ⏸️ Post-MVP (v1.5.0)

Прозрачный прокси на уровне ядра.

**Причина откладки:** Требует повышенных привилегий и сложной настройки.

---

## HAProxy Integration

**Статус:** ⏸️ Post-MVP (v2.0.0)

**Преимущества:**
- SNI-based routing (множественные inbound на одном порту)
- Path-based routing для HTTP/WebSocket
- Централизованный rate limiting

**Причина откладки:** 
- Дополнительные 40-50MB RAM
- Усложнение архитектуры
- Для MVP достаточно прямого подключения к ядрам

---

## Multi-Admin с ролями

**Статус:** ⏸️ Post-MVP (v1.5.0)

**Планируемые роли:**
- Super Admin (полный доступ)
- Manager (управление пользователями, без настроек системы)
- Support (только просмотр, без редактирования)

---

## User Portal

**Статус:** ⏸️ Post-MVP (v2.0.0)

Личный кабинет для обычных пользователей (не администраторов).

**Функционал:**
- Просмотр статистики трафика
- Информация о подписке
- QR коды для подключения
- Смена пароля

**Причина откладки:** Усложнение архитектуры аутентификации. Для MVP достаточно панели только для администраторов.

---

## Приоритеты реализации

### v1.1.0 (Следующий релиз)
1. ✅ Email notifications
2. ⏸️ GeoIP/GeoSite core integration
3. ⏸️ WARP token refresh

### v1.2.0
1. ⏸️ Remote backup destinations
2. ⏸️ SSH protocol support
3. ⏸️ WireGuard protocol support

### v1.5.0
1. ⏸️ Multi-admin с ролями
2. ⏸️ TProxy/TUN support

### v2.0.0
1. ⏸️ HAProxy integration
2. ⏸️ User portal

---

## Исключено из roadmap

Следующие функции были рассмотрены, но исключены из плана разработки:

- **Webhook для каждого события** — заменено на универсальный webhook notifier
- **Интеграция с Discord/Slack** — можно реализовать через webhooks
- **Push уведомления** — низкий приоритет для серверного приложения
