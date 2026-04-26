# Phase 14: Optimization & Polish — Детальный план реализации

**Дата создания:** 1 апреля 2026  
**Цель:** Довести проект до 100% готовности к релизу v1.0.0  
**Текущая готовность:** 35%

---

## Текущее состояние (аудит)

| Задача | Статус | Комментарий |
|--------|--------|-------------|
| **14.1** subscription_token индекс | ✅ Уже есть | Индекс `idx_users_subscription_token` в миграции 000002 |
| **14.2** CacheManager в DI | ⚠️ Частично | Прокинут в 4 сервиса, не везде активен |
| **14.3** WebSockets на Dashboard | ❌ Нет | Ни пакета, ни кода |
| **14.4** Рефакторинг main.go | ❌ Нет | 533 строки монолитной инициализации |
| **14.5** Version через ldflags | ❌ Нет | Хардкод `v0.1.0` в 2 местах |
| **14.6** Frontend code splitting | ❌ Нет | Все 17 страниц импортируются статически |
| **14.7** Rate Limiting для auth | ⚠️ Частично | Только login и subscription |
| **14.8** Audit Log | ❌ Нет | Полностью отсутствует |
| **14.9** Request Validation Middleware | ⚠️ Частично | validate-теги есть, движок не подключён |
| **14.10** CSP headers + govulncheck | ❌ Нет | Никаких security headers |
| **14.11** 2FA / TOTP | ⚠️ Модель есть | Поля в Admin, но логики нет |
| **14.12–14.15** Документация | ❌ Нет | Только ручной docs/API.md |
| **14.16** Auto traffic reset | ⚠️ Частично | Cron-инфраструктура и метод есть, шедулинга нет |

---

## Блок 1: Performance & Refactoring

### 14.1 — subscription_token индекс ✅ ВЫПОЛНЕНО
Индекс `idx_users_subscription_token` уже создан в миграции `000002_create_users_table.up.sql`.  
Задача закрыта без дополнительных действий.

---

### 14.5 — Version через ldflags

**Файлы:**
- создать `backend/internal/version/version.go`
- обновить `backend/Makefile`
- обновить `backend/cmd/server/main.go` (убрать хардкод)

**Реализация:**

```go
// backend/internal/version/version.go
package version

var (
    Version   = "dev"
    BuildDate = "unknown"
    GitCommit = "unknown"
)
```

**Makefile (фрагмент):**
```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X github.com/isolate-project/isolate-panel/internal/version.Version=$(VERSION) \
           -X github.com/isolate-project/isolate-panel/internal/version.BuildDate=$(BUILD_DATE) \
           -X github.com/isolate-project/isolate-panel/internal/version.GitCommit=$(GIT_COMMIT)

build:
    go build -ldflags "$(LDFLAGS)" -o bin/server cmd/server/main.go
```

**В main.go:** заменить `"v0.1.0"` и `"0.1.0"` на `version.Version`.

---

### 14.4 — Рефакторинг main.go

**Цель:** Разбить 533-строчный `main.go` на модули. `main()` должна стать тонкой.

**Новая структура:**

```
backend/internal/app/
├── providers.go    # DI: инициализация всех сервисов
├── routes.go       # регистрация всех маршрутов
└── background.go   # запуск/остановка фоновых воркеров
```

**providers.go** — функция `InitServices(cfg, db) (*Services, error)`, где `Services` — struct со всеми сервисами и хендлерами.

**routes.go** — функция `SetupRoutes(app *fiber.App, s *Services, tokenSvc *auth.TokenService)`.

**background.go** — функции `StartBackgroundWorkers(s *Services)` и `StopBackgroundWorkers(s *Services)`.

**Итоговый main.go:** ~60-80 строк:
1. Load config
2. Init logger
3. Init DB
4. `svc := app.InitServices(cfg, db)`
5. `app.SetupRoutes(fiber, svc)`
6. `app.StartBackgroundWorkers(svc)`
7. Listen + graceful shutdown с вызовом `StopBackgroundWorkers`

---

### 14.2 — CacheManager DI (мёртвый код)

**Проблема:** `CacheManager` прокинут в сервисы, но вызовы кэша обёрнуты в `if s.cache != nil` — нужно убедиться что:
1. Кэш реально инвалидируется при мутациях (update/delete)
2. Нет сервисов, которые принимают cacheManager но никогда не используют его методы

**Действия:**
- Аудит использования в `UserService`, `SubscriptionService`, `ConfigService`, `SettingsService`
- Убрать неиспользуемые поля/параметры или активировать кэш там, где он нужен
- Убедиться что при `UpdateUser` / `DeleteUser` происходит инвалидация кэша

---

## Блок 2: Security

### 14.7 — Rate Limiting для auth-эндпоинтов

**Текущее состояние:** Rate limiting только для `/auth/login` (5 req/min) и `/sub/:token` (10 req/hour).

**Новый middleware `AuthRateLimiter`:**
- Стандартный: 60 req/min per user (ключ = userID из JWT)
- Тяжёлые операции: 10 req/min (backup create, cert request, core restart)

**Применение в routes.go:**
```go
// Стандартный лимит на весь protected group
protectedGroup.Use(middleware.AuthRateLimiter(standardLimiter))

// Повышенный лимит на конкретные роуты
backupGroup.Post("/", middleware.AuthRateLimiter(heavyLimiter), backupHandler.CreateBackup)
```

---

### 14.9 — Request Validation Middleware

**Проблема:** `validate`-теги есть в struct-ах, но `go-playground/validator` не подключён.

**Реализация:**
1. Добавить `github.com/go-playground/validator/v10` в `go.mod`
2. Создать `middleware/validator.go`:

```go
func BindAndValidate[T any](c fiber.Ctx) (T, error) {
    var req T
    if err := c.Bind().JSON(&req); err != nil {
        return req, fiber.NewError(fiber.StatusBadRequest, "Invalid JSON: "+err.Error())
    }
    if err := validate.Struct(req); err != nil {
        return req, fiber.NewError(fiber.StatusBadRequest, formatValidationErrors(err))
    }
    return req, nil
}
```

3. Рефакторить хендлеры: заменить ручную валидацию на `BindAndValidate`.

---

### 14.10 — Security Headers + govulncheck

**Создать `middleware/security.go`:**

```go
func SecurityHeaders() fiber.Handler {
    return func(c fiber.Ctx) error {
        c.Set("X-Content-Type-Options", "nosniff")
        c.Set("X-Frame-Options", "DENY")
        c.Set("X-XSS-Protection", "1; mode=block")
        c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Set("Content-Security-Policy",
            "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")
        return c.Next()
    }
}
```

**govulncheck:**
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```
Зафиксировать результат, обновить уязвимые зависимости.

---

### 14.8 — Audit Log

**Модель:**
```go
// internal/models/audit_log.go
type AuditLog struct {
    ID         uint      `gorm:"primaryKey"`
    AdminID    uint      `gorm:"index;not null"`
    Action     string    `gorm:"not null"` // user.create, user.delete, core.start, ...
    Resource   string    `gorm:"not null"` // user, core, inbound, backup, ...
    ResourceID *uint
    Details    string    // JSON строка с деталями
    IPAddress  string
    CreatedAt  time.Time `gorm:"index"`
}
```

**Миграция:** `000XXX_create_audit_logs.up.sql`

**AuditService:**
- `Log(ctx, adminID, action, resource, resourceID, details) error`
- Retention: удаление записей старше 90 дней через `DataRetentionService`

**Middleware:** `AuditMiddleware` — вешается на критические маршруты.

**API:** `GET /api/audit-logs?page=1&limit=50&action=user.delete`

---

### 14.11 — 2FA / TOTP

**Зависимость:** `github.com/pquerna/otp/totp`

**Поля в Admin уже есть:** `TOTPSecret string`, `TOTPEnabled bool`

**Новые эндпоинты:**
- `POST /api/auth/totp/setup` — генерирует TOTP secret + QR URI
- `POST /api/auth/totp/verify` — подтверждает код, включает 2FA
- `POST /api/auth/totp/disable` — отключает 2FA (требует пароль)

**Изменение Login flow:**
```
POST /auth/login { username, password, totp_code? }
→ если totp_enabled: проверить totp_code
→ если totp_code отсутствует: вернуть { requires_totp: true }
→ фронтенд показывает поле ввода TOTP
```

**Frontend:** поле ввода TOTP на странице Login, страница настройки в Settings с QR-кодом.

---

## Блок 3: Frontend Optimization

### 14.6 — Code Splitting (React lazy loading)

**Текущее состояние:** Все 17 страниц импортируются статически в `app.tsx`.

**Реализация:**

```tsx
// До:
import { Dashboard } from './pages/Dashboard'

// После:
const Dashboard = lazy(() => import('./pages/Dashboard').then(m => ({ default: m.Dashboard })))
```

Все маршруты обернуть в `<Suspense fallback={<PageLoader />}>`.

**Приоритет lazy loading:**
1. Dashboard (Chart.js — самый тяжёлый)
2. Inbounds / InboundCreate / InboundEdit
3. Certificates
4. Backups
5. Остальные страницы

**Vite** автоматически создаст отдельные чанки при dynamic import.

---

## Блок 4: Real-time & Scheduling

### 14.3 — WebSockets на Dashboard

**Зависимость:** `github.com/gofiber/contrib/websocket`

**Backend:**
- Endpoint: `GET /api/ws/dashboard` (auth через `?token=` query param)
- Hub-паттерн: горутина читает stats каждые 5 сек, рассылает всем клиентам
- Payload:
```json
{
  "active_connections": 42,
  "total_traffic_today": 1073741824,
  "cores": [{"name": "xray", "status": "running"}],
  "top_users": [...]
}
```

**Frontend:**
- `useWebSocket(url)` хук в `src/hooks/useWebSocket.ts`
- Dashboard.tsx подписывается на WS, обновляет state
- Fallback: если WS недоступен — polling каждые 30 сек (существующий механизм)

---

### 14.16 — Auto Traffic Reset по расписанию

**Инфраструктура:** `robfig/cron/v3` уже есть, `ResetUserTraffic()` в QuotaEnforcer уже есть.

**Settings:** добавить поле `traffic_reset_schedule` (enum: `disabled` | `weekly` | `monthly` | `first_of_month`)

**TrafficResetScheduler:**
- При старте читает настройку из Settings
- Регистрирует cron-задачу (`0 0 1 * *` для monthly, `0 0 * * 1` для weekly)
- При срабатывании: вызывает `QuotaEnforcer.ResetUserTraffic()` + отправляет уведомление

**UI:** dropdown в Settings → раздел "Traffic Management".

---

## Блок 5: Documentation

### 14.12 — Обновить README.md
- Актуальная архитектура, список фич, badges (Go version, license, CI status)
- Инструкции: Docker quick start, manual install, SSH tunnel access

### 14.13 — Swagger/OpenAPI
- Добавить `swaggo/swag` + `gofiber/swagger`
- Аннотировать все хендлеры
- `swag init -g cmd/server/main.go`
- Endpoint: `GET /api/docs` → Swagger UI

### 14.14 — Architecture & Contributing docs
- `docs/ARCHITECTURE.md`: диаграмма слоёв, описание пакетов, flow charts
- `docs/CONTRIBUTING.md`: dev setup, code style, PR process, commit convention

### 14.15 — CHANGELOG.md
- Формат: Keep a Changelog (https://keepachangelog.com)
- Версия v1.0.0: собрать изменения из Phases 0–14
- Подготовить к `git tag v1.0.0`

---

## Рекомендуемый порядок реализации

```
Блок 1 (Refactoring)
  └── 14.1 ✅ → 14.5 → 14.4 → 14.2

Блок 2 (Security)
  └── 14.10 → 14.9 → 14.7 → 14.8 → 14.11

Блок 3 (Frontend)
  └── 14.6

Блок 4 (Real-time)
  └── 14.5 уже done → 14.16 → 14.3

Блок 5 (Docs)
  └── 14.12 → 14.13 → 14.14 → 14.15
```

## Оценка сложности

| Блок | Задачи | Сложность |
|------|--------|-----------|
| Блок 1 | 14.1✅, 14.2, 14.4, 14.5 | Средняя |
| Блок 2 | 14.7, 14.8, 14.9, 14.10, 14.11 | Высокая (14.11 — самая трудоёмкая) |
| Блок 3 | 14.6 | Низкая |
| Блок 4 | 14.3, 14.16 | Средняя |
| Блок 5 | 14.12–14.15 | Средняя (14.13 Swagger — рутина) |
