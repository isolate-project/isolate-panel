# Phase 10: Notifications — Детальный план доработок до 100%

**Дата:** 2 апреля 2026  
**Текущий статус:** ~70% (бэкенд-инфраструктура готова, фронтенд не подключён, тесты поверхностные)  
**Цель:** 100% (полностью рабочие Telegram + Webhook нотификации с ретраями, UI, и покрытие тестами)

---

## Текущее состояние

### Что уже реализовано ✅

| Компонент | Файл | Статус |
|-----------|------|--------|
| Notification model + DB schema | `internal/models/notification.go`, `migrations/000026_*` | Готово |
| NotificationService (Send, List, Get, Delete, Settings) | `internal/services/notification_service.go` | Готово |
| WebhookNotifier (HMAC-SHA256 подпись) | `internal/services/webhook_notifier.go` | Готово |
| TelegramNotifier (Bot API, Markdown, TestConnection) | `internal/services/telegram_notifier.go` | Готово |
| API handlers (6 эндпоинтов) | `internal/api/notifications.go` | Готово |
| Trigger: QuotaEnforcer → NotifyQuotaExceeded/Warning | `internal/services/quota_enforcer.go` | Готово |
| Trigger: UserService → NotifyUserCreated/Deleted/ExpiryWarning | `internal/services/user_service.go` | Готово |
| Trigger: CertificateService → NotifyCertRenewed | `internal/services/certificate_service.go` | Готово |
| Trigger: CoreLifecycle → NotifyCoreError | `internal/services/core_lifecycle.go` | Готово |
| Trigger: AuthHandler → NotifyFailedLogin (5+ попыток) | `internal/api/auth.go:444` | Готово |

### Что НЕ работает / отсутствует ❌

| Проблема | Описание |
|----------|----------|
| **Нет фонового retry-воркера** | Retry-логика вычисляет `NextRetryAt`, но ничего не перезапускает failed-нотификации. `sendNotification()` вызывается один раз в goroutine, при ошибке — просто ставит `failed` статус и `NextRetryAt`, но воркера, который бы проверял и пересылал, нет. |
| **Линейный backoff вместо exponential** | Текущая формула: `retryCount * 5min` (5m, 10m, 15m). Нужен exponential: `2^retryCount * baseInterval`. |
| **Frontend: нет `notificationApi`** | В `frontend/src/api/endpoints/index.ts` **нет** notification endpoints. Страница `Notifications.tsx` использует `backupApi.list()` и `backupApi.delete()` — это баг, данные грузятся из бэкапов, а не нотификаций. |
| **Frontend: Settings не подключены** | Инпуты (Webhook URL, Telegram Token, Event toggles) — статический HTML без `value`, `onChange`, и API-вызовов. |
| **Frontend: Test Notification — заглушка** | `handleSendTest()` содержит `alert('Test notification sent!')` без реального вызова API. |
| **Тесты поверхностные** | `notification_service_test.go` — только создание сервиса (3 теста) + модель (5 тестов). Нет тестов отправки, retry, webhook/telegram с mock HTTP server. API тесты — только ошибочные кейсы. |
| **Bug: escapeMarkdown ломает форматирование** | `TelegramNotifier.escapeMarkdown()` экранирует ВСЕ `*` и `_`, включая те, что используются для Markdown-форматирования в `formatMessage()`. Результат: `\*\*emoji title\*\*` вместо жирного текста. |

---

## План работ

### Block 1: Backend — Retry Worker + Exponential Backoff (Задача 10.2)

**Цель:** Неотправленные нотификации автоматически ретраятся по расписанию с exponential backoff.

#### 10.2.1 — Exponential backoff в `sendNotification()`

**Файл:** `internal/services/notification_service.go`

**Изменение:** Заменить линейный backoff `retryCount * 5min` на exponential `2^retryCount * 30s`:

```
Retry 1: 30s
Retry 2: 60s  
Retry 3: 120s (2 мин)
```

Текущий код (строка ~167):
```go
nextRetry := now.Add(time.Duration(notification.RetryCount*5) * time.Minute)
```

Заменить на:
```go
backoff := time.Duration(1<<notification.RetryCount) * 30 * time.Second
nextRetry := now.Add(backoff)
```

#### 10.2.2 — Retry Worker (фоновый горутин)

**Файл:** `internal/services/notification_service.go` — добавить методы:

- `StartRetryWorker(ctx context.Context)` — горутин, каждые 30 секунд проверяет `notifications` таблицу на записи со `status = 'failed' AND retry_count < max_retries AND next_retry_at <= NOW()`, и пересылает их через `sendNotification()`.
- `StopRetryWorker()` — через `context.Cancel()`.

**Файл:** `internal/app/background.go` — добавить запуск/остановку retry-воркера в `StartWorkers()` / `StopWorkers()`.

**Файл:** `internal/app/providers.go` — нет изменений (воркер стартует через background.go).

#### 10.2.3 — Метод `RetryFailed()` 

**Файл:** `internal/services/notification_service.go`

Добавить публичный метод `RetryFailed()` для использования воркером:
```go
func (s *NotificationService) RetryFailed() error {
    var pending []models.Notification
    s.db.Where("status = ? AND retry_count < max_retries AND next_retry_at <= ?",
        models.NotificationStatusFailed, time.Now()).
        Find(&pending)
    
    for i := range pending {
        s.sendNotification(&pending[i])
    }
    return nil
}
```

**Тесты (Block 4):** Тест retry-воркера, тест exponential backoff интервалов, тест что `max_retries` соблюдается.

---

### Block 2: Backend — Bugfix Telegram Markdown (Задача 10.1)

**Цель:** Исправить форматирование Telegram-сообщений, валидировать интеграцию.

#### 10.1.1 — Fix `escapeMarkdown()` 

**Файл:** `internal/services/telegram_notifier.go`

**Проблема:** `escapeMarkdown()` вызывается ПОСЛЕ `formatMessage()`, который добавляет markdown-разметку (`*bold*`, `_italic_`). Escaping ломает всю разметку.

**Решение:** Экранировать только пользовательские данные (title, message, metadata values), а не итоговую строку. Перенести escaping внутрь `formatMessage()` — экранировать каждое поле отдельно перед вставкой в шаблон:

```go
func (t *TelegramNotifier) formatMessage(notification *models.Notification, emoji string) string {
    title := t.escapeMarkdown(notification.Title)
    message := t.escapeMarkdown(notification.Message)
    // ... build markdown using safe title/message ...
}
```

И убрать вызов `escapeMarkdown()` из `Send()`.

#### 10.1.2 — Валидация Telegram отправки через mock HTTP server

**Файл:** `internal/services/telegram_notifier_test.go` (новый)

Создать `httptest.NewServer` который эмулирует Telegram Bot API (`/botTOKEN/sendMessage`, `/botTOKEN/getMe`). Проверить:
- Формат сообщения (markdown разметка не сломана)
- `TestConnection()` возвращает nil при валидном ответе
- `SendTestMessage()` отправляет корректный payload
- Обработка ошибок (невалидный token, сетевая ошибка, non-OK response)

**Тесты:** ~8 тестов.

---

### Block 3: Frontend — Notification API + UI Wiring (Задача 10.3)

**Цель:** Страница Notifications полностью работает: загружает историю, управляет настройками, отправляет тест.

#### 10.3.1 — `notificationApi` endpoints

**Файл:** `frontend/src/api/endpoints/index.ts`

Добавить:
```ts
export const notificationApi = {
  list: (params?: { limit?: number; offset?: number }) =>
    apiClient.get('/notifications', { params }),
  get: (id: number) => apiClient.get(`/notifications/${id}`),
  delete: (id: number) => apiClient.delete(`/notifications/${id}`),
  getSettings: () => apiClient.get('/notifications/settings'),
  updateSettings: (data: Record<string, unknown>) =>
    apiClient.put('/notifications/settings', data),
  sendTest: (channel: string) =>
    apiClient.post('/notifications/test', { channel }),
}
```

#### 10.3.2 — Переписать `Notifications.tsx`

**Файл:** `frontend/src/pages/Notifications.tsx`

**Изменения:**
1. Заменить `import { backupApi }` на `import { notificationApi }`
2. Добавить state для settings (`useState<NotificationSettings>`)
3. `loadData()` → вызывать `notificationApi.list()` и `notificationApi.getSettings()`
4. Привязать value/onChange к инпутам Settings (webhook URL, secret, telegram token, chat ID)
5. Event toggle checkboxes → привязать к `settings.notify_*` полям
6. Кнопка «Save Settings» → `notificationApi.updateSettings(settings)`
7. Кнопка «Send Test» → `notificationApi.sendTest(testChannel)`, показать results из ответа
8. Добавить `handleDeleteNotification()` → `notificationApi.delete(id)`
9. Пагинация (кнопки prev/next) для истории уведомлений

#### 10.3.3 — TypeScript интерфейсы

**Файл:** `frontend/src/pages/Notifications.tsx` (или `types/`)

```ts
interface NotificationSettings {
  id: number
  webhook_enabled: boolean
  webhook_url: string
  webhook_secret: string
  telegram_enabled: boolean
  telegram_bot_token: string
  telegram_chat_id: string
  notify_quota_exceeded: boolean
  notify_expiry_warning: boolean
  notify_cert_renewed: boolean
  notify_core_error: boolean
  notify_failed_login: boolean
  notify_user_created: boolean
  notify_user_deleted: boolean
}
```

---

### Block 4: Тесты (Задача 10.4)

**Цель:** Полное покрытие Telegram notifier, Webhook notifier, NotificationService (Send, retry, event filtering), API handlers.

#### 10.4.1 — Webhook Notifier тесты

**Файл:** `internal/services/webhook_notifier_test.go` (новый)

Тесты с `httptest.NewServer`:

| # | Тест | Что проверяем |
|---|------|---------------|
| 1 | `TestWebhookNotifier_Send_Success` | Отправка, статус 200, JSON payload корректен |
| 2 | `TestWebhookNotifier_Send_HMACSignature` | Заголовок `X-Isolate-Panel-Signature` содержит валидный HMAC-SHA256 |
| 3 | `TestWebhookNotifier_Send_NoSecret` | Без secret → заголовок подписи отсутствует |
| 4 | `TestWebhookNotifier_Send_ServerError` | Сервер отвечает 500 → ошибка |
| 5 | `TestWebhookNotifier_Send_Timeout` | Сервер задерживает ответ > timeout → ошибка |
| 6 | `TestWebhookNotifier_Send_Disabled` | `enabled=false` → nil (не отправляет) |
| 7 | `TestWebhookNotifier_Send_EmptyURL` | Пустой URL → nil |
| 8 | `TestWebhookNotifier_Send_WithMetadata` | Metadata корректно сериализуется в payload |

#### 10.4.2 — Telegram Notifier тесты

**Файл:** `internal/services/telegram_notifier_test.go` (новый)

| # | Тест | Что проверяем |
|---|------|---------------|
| 1 | `TestTelegramNotifier_Send_Success` | Mock сервер, `ok: true` → nil |
| 2 | `TestTelegramNotifier_Send_APIError` | `ok: false, description: "..."` → error |
| 3 | `TestTelegramNotifier_Send_Disabled` | `enabled=false` → nil |
| 4 | `TestTelegramNotifier_Send_MessageFormat` | Проверить что body содержит chat_id, text с emoji и markdown |
| 5 | `TestTelegramNotifier_TestConnection_Success` | Mock `/getMe` → nil |
| 6 | `TestTelegramNotifier_TestConnection_Error` | Mock `/getMe` ошибка → error |
| 7 | `TestTelegramNotifier_SendTestMessage` | Вызывает Send с test notification |
| 8 | `TestTelegramNotifier_EscapeMarkdown` | Пользовательский текст с `_*[]` экранирован, markdown-разметка сохранена |
| 9 | `TestTelegramNotifier_SeverityEmoji` | Critical→🚨, Error→❌, Warning→⚠️, Info→ℹ️ |

#### 10.4.3 — NotificationService тесты (расширение)

**Файл:** `internal/services/notification_service_test.go` (новый, заменяет `tests/unit/services/notification_service_test.go`)

| # | Тест | Что проверяем |
|---|------|---------------|
| 1 | `TestNotificationService_Send_CreatesRecord` | Send() создаёт запись в БД |
| 2 | `TestNotificationService_Send_DisabledEvent` | Отключенный event type → запись не создаётся |
| 3 | `TestNotificationService_Send_MetadataSerialized` | Metadata корректно сохранена как JSON |
| 4 | `TestNotificationService_Initialize_CreatesDefaults` | На пустой БД создаётся default settings |
| 5 | `TestNotificationService_Initialize_LoadsExisting` | Загружает существующие settings |
| 6 | `TestNotificationService_UpdateSettings_Persists` | UpdateSettings сохраняет в БД и обновляет runtime |
| 7 | `TestNotificationService_ListNotifications_Pagination` | limit/offset работают корректно |
| 8 | `TestNotificationService_DeleteNotification` | Удаление по ID |
| 9 | `TestNotificationService_CleanupOldNotifications` | Оставляет только maxNotifications записей |
| 10 | `TestNotificationService_RetryFailed` | Метод находит failed уведомления с истекшим next_retry_at |
| 11 | `TestNotificationService_RetryFailed_RespectsMaxRetries` | Не ретраит записи с retry_count >= max_retries |
| 12 | `TestNotificationService_ExponentialBackoff` | Проверяет интервалы: 30s, 60s, 120s |
| 13 | `TestNotificationService_NotifyQuotaExceeded` | Helper отправляет правильный event type |
| 14 | `TestNotificationService_NotifyCoreError` | Helper формирует правильный title/message/metadata |
| 15 | `TestNotificationService_IsEventTypeEnabled` | Все 7 типов + default=true |

#### 10.4.4 — API Handler тесты (дополнение)

**Файл:** `internal/api/notifications_test.go` (дополнить)

| # | Тест | Что проверяем |
|---|------|---------------|
| 1 | `TestNotificationsHandler_GetSettings_OK` | Возвращает settings из БД |
| 2 | `TestNotificationsHandler_UpdateSettings_OK` | Обновляет и возвращает новые settings |
| 3 | `TestNotificationsHandler_Get_OK` | Существующая нотификация возвращается |
| 4 | `TestNotificationsHandler_List_WithData` | Создать N записей, проверить пагинацию |
| 5 | `TestNotificationsHandler_SendTest_AllChannels` | channel=all → обе результата в массиве |

---

### Block 5: Обновление GLOBAL_changes (финализация)

После завершения всех блоков:

1. Обновить `GLOBAL_changes_for_100%_complete.md`:
   - Phase 10 table: `⚠️ 70%` → `✅ 100%`
   - Задачи 10.1–10.4: `[ ]` → `[x]`
   - 10.5 оставить `[ ]` с пометкой `(отложено → v1.1.0)`

---

## Порядок выполнения

```
Block 1 (10.2) — Retry Worker + Exponential Backoff
    ↓
Block 2 (10.1) — Bugfix Telegram Markdown
    ↓
Block 3 (10.3) — Frontend Notification API + UI
    ↓
Block 4 (10.4) — Тесты (все блоки)
    ↓
Block 5 — GLOBAL_changes update
```

**Зависимости:**
- Block 2 зависит от Block 1 (retry-логика влияет на тесты отправки)
- Block 4 зависит от Blocks 1-3 (тестируем финальный код)
- Block 3 независим от 1-2 (фронтенд работает через API, бэкенд уже отдаёт данные)

---

## Оценка объёма

| Block | Файлы | Новый код | Тесты |
|-------|-------|-----------|-------|
| 1. Retry Worker | 3 файла (notification_service.go, background.go, providers.go) | ~60 строк | 3 теста |
| 2. Telegram Fix | 1 файл (telegram_notifier.go) | ~15 строк (рефакторинг) | — |
| 3. Frontend | 2 файла (endpoints/index.ts, Notifications.tsx) | ~200 строк (переписать страницу) | — |
| 4. Тесты | 4 файла (webhook_test, telegram_test, service_test, notifications_test) | ~600 строк тестов | ~40 тестов |
| **Итого** | **~10 файлов** | **~875 строк** | **~43 теста** |

---

## Файлы, которые будут затронуты

### Новые файлы
- `backend/internal/services/webhook_notifier_test.go`
- `backend/internal/services/telegram_notifier_test.go`
- `backend/internal/services/notification_service_test.go`

### Изменяемые файлы
- `backend/internal/services/notification_service.go` — retry worker, exponential backoff, RetryFailed()
- `backend/internal/services/telegram_notifier.go` — fix escapeMarkdown
- `backend/internal/app/background.go` — start/stop retry worker
- `backend/internal/api/notifications_test.go` — дополнительные тесты
- `frontend/src/api/endpoints/index.ts` — notificationApi
- `frontend/src/pages/Notifications.tsx` — полная переработка
- `GLOBAL_changes_for_100%_complete.md` — статус Phase 10

### Удаляемые файлы
- `backend/tests/unit/services/notification_service_test.go` — перенесён и расширен в `internal/services/notification_service_test.go`
