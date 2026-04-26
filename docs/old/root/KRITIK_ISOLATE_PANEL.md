# Критический обзор проекта Isolate Panel

Проанализированы все три слоя проекта: бэкенд (Go), фронтенд (Preact), документация и конфигурация. Ниже — полный разбор по категориям.

---

## 🔴 КРИТИЧЕСКИЕ ПРОБЛЕМЫ (Security / Data Loss)

### 1. Скомпрометированные секреты в репозитории
- **`docker/.env`** содержит реальный JWT-секрет и пароль `admin`. Файл закоммичен, несмотря на `.gitignore` (который исключает только корневой `.env`, но не `docker/.env`). Любой, кто имеет доступ к репозиторию, может подписывать произвольные JWT-токены.
- **CLAUDE.md** тоже содержит JWT-секрет в примере конфигурации.

### 2. Неправильная конвертация `int → string` — сломанный кэш и Content-Length
- **`internal/cache/manager.go:129`** — `string(rune(userID))` преобразует число в Unicode-символ, а не в строку с цифрами. Для userID=1 ключ становится `"subscription:\x01:v2ray"`. Кэш подписок **не работает вообще**.
- **`internal/api/backup.go:233`** — та же ошибка: `string(rune(len(data)))` для `Content-Length`. Заголовок получает мусорный символ вместо числа.

### 3. Пароли пользователей в ответах API
- **`internal/api/users.go:263`** и **`internal/services/user_service.go:57`** — `UserResponse` включает поле `Password` с прокси-паролем пользователя в открытом виде. Это утечка учётных данных.

### 4. Паники из-за unchecked type assertions
- **`internal/api/auth.go:270,299,339,379,419`** — `c.Locals("admin_id").(uint)` без comma-ok. Если middleware не установил значение (или middleware был пропущен), сервер упадёт с паникой на каждом запросе.
- То же в **`internal/api/users.go:37`**.

### 5. Неправильные API-эндпоинты в фронтенде
- **`src/pages/Notifications.tsx:27`** — использует `backupApi.list()` для загрузки уведомлений.
- **`src/pages/Notifications.tsx:36-44`** — удаляет уведомления через `backupApi.delete()`.
- Это copy-paste ошибка — страница уведомлений полностью нерабочая.

### 6. `class` вместо `className` в JSX
- **`src/pages/Notifications.tsx`**, **`src/pages/WarpRoutes.tsx`**, **`src/pages/GeoRules.tsx`** — во всех трёх страницах используется `class` вместо `className`. Стили **не применяются вообще**, страницы выглядят сломанными.

### 7. Нефункциональный UI
- **`src/pages/Certificates.tsx:280-323`** — модальное окно загрузки сертификата: кнопка upload не имеет `onClick`, поле domain не привязано к state. Полностью нерабочее.
- **`src/pages/Inbounds.tsx:283`** — кнопка "Assign" с `onClick={() => {}}`.
- **`src/pages/Notifications.tsx:52`** — отправка тестового уведомления с комментарием `// Will be implemented`.

---

## 🟠 СЕРЬЁЗНЫЕ АРХИТЕКТУРНЫЕ ПРОБЛЕМЫ

### 8. Порт 8080 открыт на все интерфейсы в Docker
- **`docker/docker-compose.yml`** — `ports: "8080:8080"` вместо `"127.0.0.1:8080:8080"`. Проект позиционируется как «доступ только через SSH-туннель», но Docker exposes порт на всю сеть.

### 9. Внешняя зависимость от `sqlite3` CLI
- **`internal/services/backup_service.go:272,841`** — бэкап/рестор используют `exec.Command("sqlite3", ...)`. Если `sqlite3` не установлен в системе (а в Alpine-образе его нет по умолчанию), все бэкапы падают. Зависимость не задокументирована.

### 10. Коллизия имён файлов при бэкапе конфигов
- **`internal/services/backup_service.go:867-872`** — и Xray, и Sing-box хранят конфиги как `config.json`. При бэкапе они перезаписывают друг друга. При ресторе один конфиг затирает другой.

### 11. OOM при скачивании бэкапа
- **`internal/services/backup_service.go:997-1009`** — `DownloadBackup` читает весь файл в память через `os.ReadFile`. Для больших бэкапов — Out Of Memory.

### 12. N+1 запросы в сборщике трафика
- **`internal/services/traffic_collector.go:172,212-218`** — каждый sample = отдельный INSERT + SELECT+UPDATE для каждого пользователя. При десятках пользователей — сотни запросов каждые 5 секунд.
- **`internal/services/connection_tracker.go:135-149`** — SELECT + INSERT/UPDATE для каждого соединения. Нет upsert (`ON CONFLICT`).

### 13. Race condition при запуске ядер
- **`internal/cores/manager.go:57,135`** — `time.Sleep(1 * time.Second)` после запуска ядра. Ядро может не успеть стартовать за 1 секунду или упасть сразу после запуска. Нужен retry loop с backoff.

### 14. XML injection в Supervisor
- **`internal/cores/supervisor.go:124-150`** — XML-RPC запрос собирается через конкатенацию строк. Если имя процесса содержит `<`, `>`, `&` — XML будет невалидным.

### 15. Spoofing rate limiting
- **`internal/middleware/ratelimit.go:91`** — `X-Forwarded-For` используется напрямую для rate limiting login. Заголовок можно подделать и обойти защиту от брутфорса.

### 16. Токен в URL WebSocket
- **`src/pages/Dashboard.tsx:45`** — access token передаётся как query parameter в WebSocket URL. Токены в URL попадают в логи сервера, историю браузера, referrer headers.

---

## 🟡 ПРОБЛЕМЫ КАЧЕСТВА КОДА

### 17. Компоненты внутри компонентов (фронтенд)
- **`src/pages/Dashboard.tsx:81`** — `StatCard` определяется внутри `Dashboard`, пересоздаётся каждый рендер.
- **`src/pages/Users.tsx:98,154`** — `UserActionMenu` и `TrafficDisplay` внутри компонента.
- **`src/app.tsx:29-147`** — 16 почти идентичных `Protected*` обёрток. Огромный boilerplate.

### 18. Erratic indentation в JSX
- **`Cores.tsx`**, **`Settings.tsx`**, **`Outbounds.tsx`**, **`Certificates.tsx`**, **`InboundCreate.tsx`**, **`InboundEdit.tsx`**, **`InboundDetail.tsx`**, **`NotFound.tsx`**, **`ActiveConnections.tsx`** — `<CardContent>` и `</Card>` имеют хаотичную индентацию. Код трудно читать.

### 19. `alert()` и `confirm()` вместо дизайн-системы
- **`Backups.tsx`**, **`WarpRoutes.tsx`**, **`GeoRules.tsx`**, **`Users.tsx`** — используются нативные браузерные диалоги вместо toast/modal системы приложения.

### 20. Отключённые страницы полностью оторваны от темы
- **`Notifications.tsx`**, **`WarpRoutes.tsx`**, **`GeoRules.tsx`** — используют захардкоженные Tailwind-классы (`bg-blue-600`, `text-gray-900`) вместо CSS-переменных дизайн-системы. В тёмной теме эти страницы не переключаются.

### 21. `fmt.Printf` вместо структурированного логгера
- **`internal/services/core_lifecycle.go:51,53,57`**, **`config_service.go:90`** — сообщения не попадут в лог-файлы при file-only output.

### 22. Слабый Argon2id
- **`internal/auth/password.go:15`** — `ArgonTime = 1`. OWASP рекомендует минимум 2-3 итерации. Хэширование быстрее, чем должно быть.

### 23. `X-XSS-Protection` заголовок
- **`internal/middleware/security.go:10`** — заголовок депрекейтед и может создавать XSS-уязвимости в некоторых браузерах.

### 24. Проверка origin в WebSocket
- **`internal/api/websocket.go:28-30`** — `CheckOrigin` всегда возвращает `true`. Любой origin может попробовать WebSocket upgrade, что открывает возможность resource exhaustion.

### 25. Неполная инициализация зависимостей
- **`internal/services/core_lifecycle.go:21-28`** — `configService` и `notificationService` устанавливаются через отдельные сеттеры после конструктора. Хрупкий паттерн.
- **`internal/app/providers.go:126,137`** — `NewNotificationService` вызывается с пустыми строками, `SubscriptionService` с пустым `panelURL` (дефолт `localhost:8080`).

### 26. Искусственная задержка 100мс
- **`src/pages/Login.tsx:62`** — `await new Promise(resolve => setTimeout(resolve, 100))` перед редиректом. Workaround для race condition с Zustand persistence.

### 27. `storage` event не работает для текущей вкладки
- **`src/hooks/useSessionExpired.ts:17`** — событие `storage` срабатывает только в других вкладках. Если токен удалён в текущей вкладке, сессия не завершится.

---

## 🔵 ДОКУМЕНТАЦИЯ И КОНФИГУРАЦИЯ

### 28. Фрагментированная и устаревшая документация
- ~50+ markdown файлов, многие на русском, многие — промежуточные статусы разработки (`PHASE_0_COMPLETE.md`, `PHASE_1_COMPLETE.md` и т.д.).
- `docs/DEVELOPMENT.md` говорит Go 1.23+ и Node 20, CI использует Go 1.25 и Node 22.
- `QUICKSTART.md` содержит захардкоженный слабый JWT-секрет, пароль `admin123`, и инструкцию открыть порт 8080 (противоречит принципу SSH-only).
- Нет `CHANGELOG.md`, нет сгенерированной OpenAPI/Swagger документации.

### 29. CI security scan не блокирует уязвимости
- **`.github/workflows/security-scan.yml`** — все шаги с `continue-on-error: true`. Snyk pinned to `@master`. Security gates никогда не падают.

### 30. E2E тесты не запускаются на PR
- **`.github/workflows/test.yml`** — E2E только на push в main. Сломанные E2E-тесты можно замёржить.

### 31. Playwright настроен на неправильный порт
- **`frontend/playwright.config.ts:28,45`** — `baseURL: http://localhost:3000`, но Vite работает на 5173. E2E тесты не подключатся.

### 32. Нет алиасов `react` → `preact/compat`
- **`frontend/vite.config.ts`** — `react-chartjs-2` и `react-i18next` импортируют из `react`. Без алиасов это может привести к дублированию React в бандле.

### 33. Моки тестов не синхронизированы с реальным кодом
- **`frontend/src/test/setup.ts:39-45`** — мок authStore экспортирует `token`/`setToken`, реальный store использует `accessToken`/`setTokens`.

---

## СВОДНАЯ СТАТИСТИКА

| Категория | Критических | Серьёзных | Средних |
|-----------|-------------|-----------|---------|
| Безопасность | 7 | 3 | 4 |
| Архитектура | 2 | 6 | 5 |
| Качество кода | 3 | 3 | 12 |
| Документация/CI | 1 | 2 | 4 |
| **Итого** | **13** | **14** | **25** |

Проект имеет хорошую базовую архитектуру (слоистая структура, тесты, миграции), но содержит критические баги, которые делают часть функционала нерабочей (кэш подписок, бэкапы, уведомления, сертификаты) и серьёзные уязвимости (скомпрометированные секреты, открытый порт, XSS через localStorage).
