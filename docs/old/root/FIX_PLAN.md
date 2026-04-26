# План исправлений Isolate Panel

На основе верифицированного анализа: 42 подтверждённых проблемы.
Каждый пункт перепроверен по исходному коду. Указаны статус верификации и варианты решения.

---

## P0 — Критические баги (ломается функционал / утечка данных)

### Fix 1. Сломанный кэш подписок — `string(rune(userID))`
**Файл:** `backend/internal/cache/manager.go:129`
**Статус:** ПОДТВЕРЖДЕНО — баг существует.
**Проблема:** `string(rune(userID))` → Unicode-символ вместо строки с цифрами. Кэш никогда не попадает.
**Решение:**
```go
// Было:
return "subscription:" + string(rune(userID)) + ":" + format
// Стало:
return "subscription:" + strconv.FormatUint(uint64(userID), 10) + ":" + format
```
Добавить `strconv` в импорты.

---

### Fix 2. Content-Length = мусорный символ
**Файл:** `backend/internal/api/backup.go:233`
**Статус:** ПОДТВЕРЖДЕНО — баг существует.
**Проблема:** `string(rune(len(data)))` → тот же баг. `string(rune(1000))` = символ `Ϩ`, а не `"1000"`.
**Решение:**
```go
// Было:
c.Set("Content-Length", string(rune(len(data))))
// Стало:
c.Set("Content-Length", strconv.Itoa(len(data)))
```
Добавить `strconv` в импорты.

---

### Fix 3. Пароли пользователей в ответах API
**Файлы:** `backend/internal/services/user_service.go:57`, `backend/internal/api/users.go:257-279`
**Статус:** ПОДТВЕРЖДЕНО — пароль прокси-пользователя возвращается в JSON.
**Контекст:** User — прокси-пользователь, не пользователь панели. Пароль нужен в plaintext для генерации конфигов ядер (VLESS, Trojan и т.д.). Хеширование здесь невозможно. Но отдавать пароль в каждом API-ответе — лишнее.
**Решение — вариант А (минимальный):** Убрать `Password` из `UserResponse`, добавить отдельный endpoint `/users/:id/credentials` для получения пароля.
**Решение — вариант Б (быстрый):** Добавить тег `json:"-"` на поле `Password` в модели `User`, отдавать пароль только через `UserResponse` при явном запросе.
```go
// В модели User:
Password string `gorm:"not null" json:"-"`  // не сериализуется автоматически

// В UserResponse — оставить поле, заполнять только при необходимости
```

---

### Fix 4. Паники из-за unchecked type assertions
**Файлы:** `backend/internal/api/auth.go:270,299,339,379,419`, `backend/internal/api/users.go:37`
**Статус:** ПОДТВЕРЖДЕНО — 6 мест с `c.Locals("admin_id").(uint)` без comma-ok.
**Проблема:** Если middleware не установил `admin_id` → паника всего сервера.
**Решение:**
```go
// Было:
adminID := c.Locals("admin_id").(uint)
// Стало:
adminID, ok := c.Locals("admin_id").(uint)
if !ok {
    return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
}
```

---

### Fix 5. Notifications.tsx — неправильные API-эндпоинты
**Файл:** `frontend/src/pages/Notifications.tsx`
**Статус:** ПОДТВЕРЖДЕНО — вызывает `backupApi.list()` и `backupApi.delete()`.
**Проблема:** Страница нотификаций полностью сломана: загружает бэкапы вместо нотификаций, удаляет бэкапы вместо нотификаций.
**Решение:**
1. Добавить `notificationApi` в `frontend/src/api/endpoints/index.ts`:
```typescript
export const notificationApi = {
  list: (params?: { limit?: number; offset?: number }) =>
    apiClient.get('/notifications', { params }),
  delete: (id: number) => apiClient.delete(`/notifications/${id}`),
  getSettings: () => apiClient.get('/notifications/settings'),
  updateSettings: (data: Record<string, unknown>) =>
    apiClient.put('/notifications/settings', data),
  sendTest: (channel: string) =>
    apiClient.post('/notifications/test', { channel }),
}
```
2. В `Notifications.tsx` заменить все `backupApi` → `notificationApi`.

---

### Fix 6. Certificates.tsx — нерабочая форма загрузки
**Файл:** `frontend/src/pages/Certificates.tsx:280-323`
**Статус:** ПОДТВЕРЖДЕНО — textarea без value/onChange, кнопка без onClick.
**Проблема:** Модал загрузки сертификата полностью нефункционален.
**Решение:**
1. Добавить state:
```typescript
const [uploadForm, setUploadForm] = useState({
  domain: '', certificate: '', private_key: '', issuer: '', is_wildcard: false,
})
```
2. Привязать `value` и `onChange` ко всем input/textarea.
3. Реализовать `handleUpload` с вызовом `certificateApi.upload(uploadForm)`.

---

### Fix 7. Inbounds.tsx — пустой onClick у кнопки Assign
**Файл:** `frontend/src/pages/Inbounds.tsx:283`
**Статус:** ПОДТВЕРЖДЕНО — `onClick={() => {}}`.
**Решение — вариант А:** Реализовать функционал назначения пользователей к inbound.
**Решение — вариант Б:** Убрать кнопку до реализации, чтобы не вводить пользователя в заблуждение.

---

### Fix 8. Path Traversal в backup restore (Zip Slip)
**Файл:** `backend/internal/services/backup_service.go:789`
**Статус:** ПОДТВЕРЖДЕНО — классическая zip-slip уязвимость.
**Проблема:** `filepath.Join(dstDir, header.Name)` без проверки `..` — злонамеренный tar может записать файлы за пределы целевой директории.
**Решение:**
```go
targetPath := filepath.Join(dstDir, header.Name)
// Защита от path traversal
if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(dstDir)+string(os.PathSeparator)) {
    return fmt.Errorf("illegal file path in archive: %s", header.Name)
}
```

---

### Fix 9. main.tsx — JSON.parse без try/catch
**Файл:** `frontend/src/main.tsx:7`
**Статус:** ПОДТВЕРЖДЕНО — краш приложения при невалидном JSON в localStorage.
**Проблема:** `JSON.parse(savedTheme)` → uncaught exception → белый экран.
**Решение:**
```typescript
// Было:
const savedTheme = localStorage.getItem('theme-storage')
const theme = savedTheme ? JSON.parse(savedTheme)?.state?.theme : 'dark'

// Стало:
let theme = 'dark'
try {
  const savedTheme = localStorage.getItem('theme-storage')
  if (savedTheme) {
    theme = JSON.parse(savedTheme)?.state?.theme || 'dark'
  }
} catch {
  // Невалидный JSON — используем тему по умолчанию
}
```

---

### Fix 10. SetSchedule — UPDATE без WHERE
**Файл:** `backend/internal/services/backup_service.go:968`
**Статус:** ПОДТВЕРЖДЕНО — затрагивает ВСЕ записи в таблице backups.
**Проблема:** `s.db.Model(&models.Backup{}).Update("schedule_cron", "")` — нет условия WHERE.
**Решение — вариант А:** Хранить расписание в таблице settings вместо backups.
**Решение — вариант Б (минимальный):** Добавить WHERE:
```go
s.db.Model(&models.Backup{}).Where("schedule_cron != ''").Update("schedule_cron", "")
```

---

## P1 — Серьёзные проблемы (безопасность / архитектура / производительность)

### Fix 11. Порт 8080 на всех интерфейсах в Docker
**Файл:** `docker/docker-compose.yml`
**Статус:** ПОДТВЕРЖДЕНО — `"8080:8080"` привязывает к 0.0.0.0.
**Проблема:** Панель доступна из сети, хотя задумана только для SSH-туннеля.
**Решение:**
```yaml
# Было:
ports:
  - "8080:8080"
# Стало:
ports:
  - "127.0.0.1:8080:8080"
```

---

### Fix 12. OOM при скачивании бэкапа
**Файл:** `backend/internal/services/backup_service.go:997-1009`
**Статус:** ПОДТВЕРЖДЕНО — `os.ReadFile` загружает весь файл в память.
**Проблема:** Большие бэкапы (сотни МБ) → OOM на VPS с 1GB RAM.
**Решение:** Заменить на стриминг через `c.SendFile()` в хендлере:
```go
// В backup.go handler:
func (h *BackupHandler) DownloadBackup(c fiber.Ctx) error {
    id, err := parseID(c)
    if err != nil { ... }

    filePath, filename, err := h.backupService.GetBackupFilePath(id)
    if err != nil { ... }

    c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
    return c.SendFile(filePath)
}
```
Добавить метод `GetBackupFilePath(id) (string, string, error)` в сервис.

---

### Fix 13. Коллизия config.json в бэкапе конфигов
**Файл:** `backend/internal/services/backup_service.go:291-299`
**Статус:** ПОДТВЕРЖДЕНО — `filepath.Base("cores/xray/config.json")` и `filepath.Base("cores/singbox/config.json")` оба дают `config.json`.
**Проблема:** Конфиг singbox перезаписывает конфиг xray при бэкапе.
**Решение:**
```go
coreConfigs := []struct{ src, name string }{
    {"cores/xray/config.json", "xray_config.json"},
    {"cores/singbox/config.json", "singbox_config.json"},
    {"cores/mihomo/config.yaml", "mihomo_config.yaml"},
}
```

---

### Fix 14. Backup rotation считает failed бэкапы
**Файл:** `backend/internal/services/backup_service.go:632`
**Статус:** ПОДТВЕРЖДЕНО — `s.db.Order("created_at DESC").Find(&backups)` без фильтра по статусу.
**Проблема:** Failed бэкапы занимают слоты retention. Успешные удаляются, failed остаются.
**Решение:**
```go
// Было:
s.db.Order("created_at DESC").Find(&backups)
// Стало:
s.db.Where("status = ?", models.BackupStatusCompleted).Order("created_at DESC").Find(&backups)
```

---

### Fix 15. Параметр force в RestoreBackup игнорируется
**Файл:** `backend/internal/services/backup_service.go:681`
**Статус:** ПОДТВЕРЖДЕНО — `force bool` принимается, но нигде не используется.
**Решение — вариант А:** Удалить параметр из сигнатуры, если не нужен.
**Решение — вариант Б:** Реализовать логику: если `force=false` — проверять, что нет запущенных ядер / активных соединений перед restore.

---

### Fix 16. Зависимость от sqlite3 CLI в бэкапах
**Файлы:** `backend/internal/services/backup_service.go:272,841`
**Статус:** ПОДТВЕРЖДЕНО — `exec.Command("sqlite3", dbPath, ".dump")`.
**Проблема:** Внешний бинарник может отсутствовать в среде.
**Решение:** Заменить на Go-код:
```go
func (s *BackupService) dumpDatabase(tmpDir string) error {
    dbPath := filepath.Join(s.dataDir, "isolate-panel.db")
    dstPath := filepath.Join(tmpDir, "database.db")

    sqlDB, err := s.db.DB()
    if err != nil { return err }

    // WAL checkpoint перед копированием
    if _, err := sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
        return fmt.Errorf("WAL checkpoint failed: %w", err)
    }

    src, err := os.Open(dbPath)
    if err != nil { return err }
    defer src.Close()

    dst, err := os.Create(dstPath)
    if err != nil { return err }
    defer dst.Close()

    _, err = io.Copy(dst, src)
    return err
}
```

---

### Fix 17. Spoofing rate limiting через X-Forwarded-For
**Файл:** `backend/internal/middleware/ratelimit.go:91`
**Статус:** ПОДТВЕРЖДЕНО — `c.Get("X-Forwarded-For")` проверяется первым.
**Проблема:** Клиент может подделать заголовок и обойти rate limiting.
**Решение:**
```go
// Было:
ip := c.Get("X-Forwarded-For")
if ip == "" { ip = c.IP() }
// Стало:
ip := c.IP()  // Fiber сам разбирает X-Forwarded-For с учётом TrustedProxies
```

---

### Fix 18. XML injection в Supervisor
**Файл:** `backend/internal/cores/supervisor.go:129-136`
**Статус:** ПОДТВЕРЖДЕНО — конкатенация строк в XML без экранирования.
**Проблема:** Имя ядра или параметры попадают в XML-RPC без экранирования.
**Решение:**
```go
import "encoding/xml"

func escapeXML(s string) string {
    var buf bytes.Buffer
    xml.EscapeText(&buf, []byte(s))
    return buf.String()
}

// В call():
case string:
    buf.WriteString(`<value><string>` + escapeXML(v) + `</string></value>`)
```

---

### Fix 19. Токен в URL WebSocket
**Файл:** `frontend/src/pages/Dashboard.tsx:45`
**Статус:** ПОДТВЕРЖДЕНО — `?token=${accessToken}` в URL.
**Проблема:** Токен попадает в access-логи, историю браузера, referrer-headers.
**Решение — вариант А:** Передавать через `Sec-WebSocket-Protocol`:
```typescript
const ws = new WebSocket(wsUrl, ['Bearer', accessToken])
```
На бэкенде — читать из `Sec-WebSocket-Protocol`.
**Решение — вариант Б:** Первым сообщением после подключения отправлять токен, а на сервере проверять его до начала трансляции данных.

---

### Fix 20. CheckOrigin всегда true в WebSocket
**Файл:** `backend/internal/api/websocket.go:28-30`
**Статус:** ПОДТВЕРЖДЕНО — `return true` без проверок.
**Решение:**
```go
CheckOrigin: func(r *fasthttp.RequestCtx) bool {
    origin := string(r.Request.Header.Peek("Origin"))
    if origin == "" {
        return true // same-origin
    }
    allowed := []string{"http://localhost:5173", "http://localhost:8080"}
    for _, a := range allowed {
        if origin == a { return true }
    }
    return false
}
```

---

### Fix 21. UpdateUser не в транзакции
**Файл:** `backend/internal/services/user_service.go:219-231`
**Статус:** ПОДТВЕРЖДЕНО — Delete + Create маппингов без транзакции.
**Проблема:** Если один INSERT падает — данные в несогласованном состоянии.
**Решение:**
```go
if req.InboundIDs != nil {
    err := us.db.Transaction(func(tx *gorm.DB) error {
        if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserInboundMapping{}).Error; err != nil {
            return err
        }
        for _, inboundID := range req.InboundIDs {
            // Проверить существование inbound
            var count int64
            tx.Model(&models.Inbound{}).Where("id = ?", inboundID).Count(&count)
            if count == 0 {
                return fmt.Errorf("inbound %d not found", inboundID)
            }
            if err := tx.Create(&models.UserInboundMapping{
                UserID: user.ID, InboundID: inboundID,
            }).Error; err != nil {
                return err
            }
        }
        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("failed to update inbound mappings: %w", err)
    }
}
```

---

### Fix 22. CreateUser — ошибка маппинга проглатывается
**Файл:** `backend/internal/services/user_service.go:145`
**Статус:** ПОДТВЕРЖДЕНО — `fmt.Printf("Warning: ...")`, ошибка не возвращается.
**Проблема:** Юзер создан, но привязка к inbound молча не создалась.
**Решение — вариант А (строгий):** Обернуть создание юзера + маппингов в транзакцию, откатывать при ошибке.
**Решение — вариант Б (мягкий):** Возвращать предупреждение в ответе:
```go
var warnings []string
for _, inboundID := range req.InboundIDs {
    mapping := &models.UserInboundMapping{UserID: user.ID, InboundID: inboundID}
    if err := us.db.Create(mapping).Error; err != nil {
        warnings = append(warnings, fmt.Sprintf("failed to map inbound %d: %v", inboundID, err))
    }
}
// Вернуть warnings вместе с user
```

---

### Fix 23. Expiry-нотификации дублируются
**Файл:** `backend/internal/services/user_service.go:304-330`
**Статус:** ПОДТВЕРЖДЕНО — нет трекинга «уже уведомлён».
**Проблема:** `CheckExpiringUsers()` вызывается периодически. При `daysLeft == 7` нотификация отправится каждый вызов в течение ~1 часа.
**Решение — вариант А:** Добавить поле `last_expiry_notified_days` в модель User.
**Решение — вариант Б (без миграции):** Хранить трекинг в памяти сервиса:
```go
type UserService struct {
    db                  *gorm.DB
    notificationService *NotificationService
    expiryNotified      map[uint]int // userID -> last notified daysLeft
}
```

---

### Fix 24. Часовая гранулярность в статистике сломана
**Файл:** `backend/internal/api/stats.go:76-81`
**Статус:** ПОДТВЕРЖДЕНО — всегда группирует по `DATE()`, даже для `hourly`.
**Решение:**
```go
var groupBy string
switch granularity {
case "hourly":
    groupBy = "STRFTIME('%Y-%m-%d %H:00', recorded_at)"
case "daily":
    groupBy = "DATE(recorded_at)"
default: // "raw"
    groupBy = "recorded_at"
}
query := h.db.Table("traffic_stats").
    Select(groupBy+" as date, SUM(upload) as upload, SUM(download) as download, SUM(total) as total").
    Where("user_id = ? AND granularity = ? AND recorded_at >= ?", userID, granularity, startDate).
    Group(groupBy).Order("date ASC")
```

---

### Fix 25. Хендлеры возвращают err.Error() клиенту
**Файлы:** `users.go:47`, `outbounds.go:110,147,175`, `stats.go:85`, `backup.go:54,126`, `certificates.go:49`, `warp.go:96` и др.
**Статус:** ПОДТВЕРЖДЕНО — 15+ мест.
**Проблема:** SQL-ошибки, пути к файлам → утечка внутренней информации.
**Решение:** Возвращать generic сообщение, логировать детали:
```go
// Было:
return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
// Стало:
log.Error().Err(err).Str("handler", "CreateUser").Msg("operation failed")
return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
```

---

### Fix 26. Дублирование валидации с несовпадающими значениями
**Файл:** `backend/internal/services/user_service.go:72-85`
**Статус:** ПОДТВЕРЖДЕНО — ручная проверка `len(req.Password) < 6`, struct tag `min=8`.
**Проблема:** Две системы валидации с разными порогами. Пользователь запутается.
**Решение:** Убрать ручную валидацию, оставить только struct tags + middleware валидатор. Привести `min=` к единому значению.

---

## P2 — Качество кода и UX

### Fix 27. N+1 запросы в сборщике трафика
**Файл:** `backend/internal/services/traffic_collector.go:155-176`
**Статус:** ПОДТВЕРЖДЕНО — отдельный INSERT на каждый sample + SELECT+UPDATE для user traffic.
**Решение:**
```go
// Batch INSERT
var statsToInsert []models.TrafficStats
for _, sample := range samples {
    statsToInsert = append(statsToInsert, models.TrafficStats{...})
}
if len(statsToInsert) > 0 {
    tc.db.CreateInBatches(statsToInsert, 100)
}

// Atomic UPDATE без предварительного SELECT
tc.db.Model(&models.User{}).Where("id = ?", userID).
    Update("traffic_used_bytes", gorm.Expr("traffic_used_bytes + ?", int64(bytes)))
```

---

### Fix 28. N+1 запросы в connection tracker
**Файл:** `backend/internal/services/connection_tracker.go:130-156`
**Статус:** ПОДТВЕРЖДЕНО — SELECT + INSERT/UPDATE на каждое соединение.
**Решение:** Использовать upsert:
```go
tc.db.Where("core_id = ? AND user_id = ? AND source_ip = ? AND source_port = ?",
    conn.CoreID, conn.UserID, conn.SourceIP, conn.SourcePort).
    Assign(models.ActiveConnection{LastActivity: now, Upload: conn.Upload, Download: conn.Download}).
    FirstOrCreate(conn)
```

---

### Fix 29. Race condition при старте ядер
**Файл:** `backend/internal/cores/manager.go:57,135`
**Статус:** ПОДТВЕРЖДЕНО — `time.Sleep(1 * time.Second)` без проверки.
**Решение:** Retry loop с backoff:
```go
func (cm *CoreManager) waitForProcess(name string, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
        running, err := cm.supervisor.IsProcessRunning(name)
        if err == nil && running { return nil }
    }
    return fmt.Errorf("process %s did not start within expected time", name)
}
```

---

### Fix 30. Компоненты внутри компонентов
**Файлы:** `frontend/src/pages/Dashboard.tsx:81` — `StatCard`, `Users.tsx:98,154` — `UserActionMenu`, `TrafficDisplay`
**Статус:** ПОДТВЕРЖДЕНО — пересоздаются при каждом рендере.
**Решение:** Вынести на модульный уровень (перед функцией-компонентом).

---

### Fix 31. 15 идентичных wrapper-компонентов в app.tsx
**Файл:** `frontend/src/app.tsx:29-147`
**Статус:** ПОДТВЕРЖДЕНО — ProtectedDashboard, ProtectedUsers и т.д. — все одинаковые.
**Решение:** Один generic wrapper:
```typescript
function Protected<P>({ Component, ...props }: { Component: ComponentType<P> } & P) {
  return <ProtectedRoute><Component {...props} /></ProtectedRoute>
}
// Использование:
<Route path="/dashboard" component={() => <Protected Component={Dashboard} />} />
```

---

### Fix 32. Sidebar использует `<a>` вместо роутерных ссылок
**Файл:** `frontend/src/components/layout/Sidebar.tsx:69-87`
**Статус:** ПОДТВЕРЖДЕНО — вызывает полную перезагрузку страницы.
**Решение:** Заменить `<a href={item.href}>` на preact-router `<Link href={item.href}>` (import from `preact-router/match` или использовать `route()` из `preact-router`).

---

### Fix 33. WarpRoutes.tsx и GeoRules.tsx — class вместо className
**Файлы:** `frontend/src/pages/WarpRoutes.tsx`, `frontend/src/pages/GeoRules.tsx`
**Статус:** ПОДТВЕРЖДЕНО — десятки мест с `class` вместо `className`.
**Контекст:** В Preact `class` технически работает, но при JSX-трансформации может вызывать warnings. Нет `<PageLayout>`, нет единого дизайна.
**Решение:**
1. Заменить `class` → `className` по всему файлу.
2. Обернуть в `<PageLayout>` для единообразия с остальными страницами.
3. Заменить hardcoded Tailwind-классы (`bg-blue-600`) на CSS-переменные дизайн-системы.

---

### Fix 34. alert()/confirm() вместо дизайн-системы
**Файлы:** `Backups.tsx`, `WarpRoutes.tsx`, `GeoRules.tsx`, `Users.tsx`, `Notifications.tsx`
**Статус:** ПОДТВЕРЖДЕНО.
**Решение:** Заменить `alert()` → `toastStore.error()` / `toastStore.success()`, `confirm()` → модальное окно подтверждения из UI-библиотеки.

---

### Fix 35. useForm — ошибки проглатываются
**Файл:** `frontend/src/hooks/useForm.ts:119-125`
**Статус:** ПОДТВЕРЖДЕНО — catch блок делает только `console.error`.
**Решение:**
```typescript
} catch (error) {
  const message = error instanceof Error ? error.message : 'Submission failed'
  setErrors(prev => ({ ...prev, _form: message }))
  // или использовать toastStore
}
```

---

### Fix 36. Chart.js объекты без useMemo
**Файл:** `frontend/src/components/features/DashboardCharts.tsx:102-177`
**Статус:** ПОДТВЕРЖДЕНО — chartData и options пересоздаются каждый рендер.
**Решение:** Обернуть в `useMemo`:
```typescript
const chartData = useMemo(() => ({ labels: [...], datasets: [...] }), [data])
const options = useMemo(() => ({ ... }), [])
```

---

### Fix 37. formatBytes дублирован 4 раза
**Файлы:** `Users.tsx:85`, `Backups.tsx:215`, `ActiveConnections.tsx:62`, `DashboardCharts.tsx:59`
**Статус:** ПОДТВЕРЖДЕНО — 4 отдельных определения.
**Решение:** Вынести в `frontend/src/utils/format.ts`:
```typescript
export function formatBytes(bytes: number, decimals = 2): string { ... }
```
Импортировать во всех 4 файлах.

---

### Fix 38. fmt.Printf вместо логгера
**Файлы:** `core_lifecycle.go` (15 мест), `config_service.go:90`, `user_service.go:145`
**Статус:** ПОДТВЕРЖДЕНО.
**Решение:** Заменить на `logger.Info()`, `logger.Warn()`, `logger.Error()` из `internal/logger`.

---

### Fix 39. Хардкод путей — не работает вне Docker
**Файл:** `backend/internal/app/providers.go:155-165`
**Статус:** ПОДТВЕРЖДЕНО — `/app/data/warp`, `/app/data/geo`, `/app/data/backups`.
**Решение:** Добавить в `config.yaml` и `Config` struct:
```yaml
data:
  warp_dir: "/app/data/warp"
  geo_dir: "/app/data/geo"
  backup_dir: "/app/data/backups"
  data_dir: "/app/data"
```
В `providers.go`:
```go
a.Warp = services.NewWARPService(db.DB, cfg.Data.WarpDir)
```

---

### Fix 40. Неполная инициализация сервисов
**Файл:** `backend/internal/app/providers.go:126,137,140`
**Статус:** ПОДТВЕРЖДЕНО.
**Проблема:** `NotificationService("", "", "", "", "")` — все каналы пустые. `SubscriptionService(db, "", cache)` — base URL пустой. `TrafficCollector(db, settings, 0, ...)` — интервал 0.
**Решение:** Передавать значения из конфига:
```go
a.Notifications = services.NewNotificationService(db.DB,
    cfg.Notifications.TelegramToken, cfg.Notifications.TelegramChatID,
    cfg.Notifications.WebhookURL, cfg.Notifications.EmailSMTP)
a.Subscriptions = services.NewSubscriptionService(db.DB, cfg.App.PanelURL, a.Cache)
a.Traffic = services.NewTrafficCollector(db.DB, a.Settings, cfg.Traffic.CollectInterval, ...)
```

---

### Fix 41. ProtectedRoute — hasRunRef блокирует ре-аутентификацию
**Файл:** `frontend/src/router/ProtectedRoute.tsx:34-37`
**Статус:** ПОДТВЕРЖДЕНО — auth check выполняется только один раз.
**Решение:** Убрать `hasRunRef`, использовать `useEffect` с зависимостью от `accessToken`:
```typescript
useEffect(() => {
  if (!accessToken) {
    route('/login', true)
    return
  }
  verifyAuth()
}, [accessToken])
```

---

### Fix 42. useSessionExpired — модульная переменная shared между табами
**Файл:** `frontend/src/hooks/useSessionExpired.ts:5`
**Статус:** ПОДТВЕРЖДЕНО — `let hasShownSessionExpired = false` на уровне модуля.
**Проблема:** Один таб показал тост → другие никогда не покажут до перезагрузки.
**Решение:** Использовать sessionStorage или ref внутри компонента:
```typescript
const hasShownRef = useRef(false)
```

---

## P3 — CI/CD, тесты, документация

### Fix 43. CI security scan не блокирует уязвимости
**Файл:** `.github/workflows/security-scan.yml`
**Статус:** ПОДТВЕРЖДЕНО — все сканы с `continue-on-error: true`.
**Решение:** Убрать `continue-on-error` с gosec и govulncheck (оставить на Snyk, т.к. требует API key).

---

### Fix 44. E2E тесты не запускаются на PR
**Файл:** `.github/workflows/test.yml`
**Статус:** ПОДТВЕРЖДЕНО — `if: github.event_name == 'push' && github.ref == 'refs/heads/main'`.
**Решение:** Убрать условие или заменить на `if: always()`.

---

### Fix 45. Тесты-пустышки
**Файл:** `backend/tests/unit/api/handlers_test.go`
**Статус:** ПОДТВЕРЖДЕНО — все 10 тестов только `assert.NotNil(handler)`.
**Решение — вариант А:** Удалить файл (тесты не тестируют ничего).
**Решение — вариант Б:** Переписать с реальными HTTP-запросами через `httptest`.

---

### Fix 46. Тесты принимают 500 как валидный ответ
**Файлы:** `cores_test.go`, `subscriptions_test.go`, `warp_test.go`
**Статус:** ПОДТВЕРЖДЕНО — `assert.Contains(t, []int{200, 500}, resp.StatusCode)`.
**Решение:** Настроить тестовую среду так, чтобы 500 не возникал, и проверять `assert.Equal(t, 200, resp.StatusCode)`.

---

### Fix 47. setupTestDB возвращает nil
**Файлы:** `handlers_test.go:80-82`, `config_service_test.go:54-56`
**Статус:** ПОДТВЕРЖДЕНО.
**Решение:** Использовать SQLite in-memory для тестов:
```go
func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)
    db.AutoMigrate(&models.User{}, &models.Inbound{}, ...)
    return db
}
```

---

### Fix 48. Playwright на неправильном порту
**Файл:** `frontend/playwright.config.ts:28,45`
**Статус:** ПОДТВЕРЖДЕНО — порт 3000, должен быть 5173.
**Решение:** Заменить `http://localhost:3000` → `http://localhost:5173`.

---

### Fix 49. Моки тестов не синхронизированы
**Файл:** `frontend/src/test/setup.ts:39-45`
**Статус:** ПОДТВЕРЖДЕНО — `token`/`setToken` вместо `accessToken`/`setTokens`.
**Решение:** Привести в соответствие с реальным authStore.

---

### Fix 50. Docker-образы без digest pinning
**Файл:** `docker/Dockerfile`
**Статус:** ПОДТВЕРЖДЕНО — `alpine:3.21`, `golang:1.25-alpine`, `node:22-alpine` без digest.
**Решение — вариант А (строгий):** Пиннить по sha256:
```dockerfile
FROM golang:1.25-alpine@sha256:<hash> AS go-builder
```
**Решение — вариант Б (прагматичный):** Оставить как есть — для self-hosted панели риск supply chain атак минимален.

---

### Fix 51. Слабый Argon2id
**Файл:** `backend/internal/auth/password.go:15`
**Статус:** ПОДТВЕРЖДЕНО — `ArgonTime = 1`, OWASP рекомендует 2-3.
**Решение:** Увеличить до `ArgonTime = 3`. **Важно:** новые хэши будут несовместимы со старыми. Нужна миграция паролей при следующем логине (перехэшировать после успешной проверки).

---

### Fix 52. X-XSS-Protection заголовок (deprecated)
**Файл:** `backend/internal/middleware/security.go:10`
**Статус:** ПОДТВЕРЖДЕНО — заголовок присутствует.
**Контекст:** `X-XSS-Protection: 1; mode=block` deprecated в современных браузерах. В некоторых случаях может создавать дополнительные XSS-уязвимости (через CSP bypass).
**Решение:** Удалить строку. CSP уже обеспечивает защиту.

---

### Fix 53. Искусственная задержка 100мс в Login
**Файл:** `frontend/src/pages/Login.tsx:62`
**Статус:** ПОДТВЕРЖДЕНО — `setTimeout(resolve, 100)`.
**Решение:** Убрать, дождаться завершения `setTokens` нормально через async/await.

---

### Fix 54. Мёртвый код во фронтенде
**Файлы:** `Tooltip.tsx`, `Checkbox.tsx`, `TableSkeleton.tsx`, `CardSkeleton.tsx`, `errorMessages.ts`
**Статус:** ПОДТВЕРЖДЕНО — 5 файлов не импортируются нигде.
**Решение:** Удалить файлы или (если планируется использование) оставить с пометкой TODO.

---

## Пункты из внешнего отчёта, НЕ подтвердившиеся

| # | Утверждение | Почему неверно |
|---|------------|----------------|
| 1.2 | `.env` закоммичен в репо | `.env` в `.gitignore`, не коммитится |
| 1.7 | Нет rate limit на `/sub/` | Rate limiting реализован (`SubscriptionRateLimiter`) |
| 3.8 | checkFailedLoginAttempts — баг | Это фича: защита от bruteforce по разным usernames с одного IP |
| 3.11 | Дубли квот-нотификаций | `warned80`/`warned90` maps трекают уведомления |
| 5.1 | N+1 в ListUsers | `Preload("CreatedByAdmin")` используется |
| 5.2 | O(n²) в joinParts | Формально верно, но 2-3 элемента — нерелевантно |

---

## Сводка по объёму работ

| Приоритет | # фиксов | Описание | Файлов затронуто |
|-----------|----------|----------|------------------|
| P0        | 10       | Критические баги, ломающие функционал | ~15 |
| P1        | 16       | Безопасность, архитектура, производительность | ~25 |
| P2        | 16       | Качество кода, UX, дублирование | ~25 |
| P3        | 12       | CI/CD, тесты, документация | ~15 |
| **Итого** | **54**   | | **~80** |

## Рекомендуемый порядок исправлений

**Фаза 1 — Критические баги (P0):** Fix 1-10
Один PR, все фиксы независимы друг от друга.

**Фаза 2 — Безопасность и архитектура (P1):** Fix 11-26
Можно разбить на 2-3 PR по темам:
- Безопасность: 8, 17, 18, 19, 20, 25
- Бэкап-система: 12, 13, 14, 15, 16
- Бизнес-логика: 21, 22, 23, 24, 26

**Фаза 3 — Качество кода (P2):** Fix 27-42
- Backend performance: 27, 28, 29
- Frontend cleanup: 30-37, 41, 42
- Backend cleanup: 38, 39, 40

**Фаза 4 — CI/CD и тесты (P3):** Fix 43-54
