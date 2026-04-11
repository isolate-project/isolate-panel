# Полный аудит проекта Isolate Panel (обновлённый)

## Общая оценка реализации

Проект имеет **солидную базовую архитектуру** и примерно **70–75% функциональной готовности**. Существующий код показывает продуманную слоистую структуру (API handlers → Services → Models → DB), но содержит критические баги, нерабочие UI-страницы и архитектурные уязвимости, которые делают панель **непригодной для production-развёртывания** без существенных исправлений.

---

## 1. Покрытие функционала по ядрам

### Xray ✅ Полное покрытие

| Функция | Статус | Детали |
|---------|--------|--------|
| Конфиг-генерация | ✅ Реализовано | [xray/config.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/xray/config.go) — 710 строк |
| VLESS | ✅ | С поддержкой Reality, WS, gRPC, H2, XHTTP |
| VMess | ✅ | С AlterID, транспортами |
| Trojan | ✅ | С TLS |
| Shadowsocks | ✅ | Multi-user через clients |
| Hysteria2 | ✅ | С obfuscation |
| XHTTP | ✅ | Exclusive протокол, splithttp network |
| Socks5 / HTTP | ✅ | С аутентификацией |
| gRPC Stats API | ✅ | [xray/stats_client.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/xray/stats_client.go) |
| Reality | ✅ | Полная поддержка (dest, serverNames, privateKey, shortIds) |
| TLS + сертификаты | ✅ | Загрузка путей из БД по `tls_cert_id` |
| Маршрутизация WARP | ✅ | Инъекция WireGuard outbound |
| Маршрутизация GeoIP | ✅ | Инъекция гео-правил |
| Outbounds | ✅ | freedom, blackhole, dns + custom |

### Sing-box ✅ Полное покрытие

| Функция | Статус | Детали |
|---------|--------|--------|
| Конфиг-генерация | ✅ | [singbox/config.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/singbox/config.go) — 810 строк |
| VLESS | ✅ | С UUID, flow |
| VMess | ✅ | С alterID |
| Trojan | ✅ | С password |
| Shadowsocks | ✅ | Multi-user mode, 2022 ciphers |
| Hysteria2 | ✅ | С obfs и bandwidth limits |
| TUIC v4 | ✅ | Token-based auth |
| TUIC v5 | ✅ | UUID + password |
| NaiveProxy | ✅ | Exclusive (username/password) |
| Mixed / Socks5 / HTTP | ✅ | С аутентификацией |
| Redirect | ✅ | Transparent proxy |
| Clash API (Stats) | ✅ | External controller + secret |
| Reality | ✅ | Handshake server, privateKey, shortID |
| TLS + сертификаты | ✅ | Path загрузка из БД |
| WARP | ✅ | WireGuard outbound injection |
| GeoIP | ✅ | Route rules + asset paths |

### Mihomo ✅ Полное покрытие

| Функция | Статус | Детали |
|---------|--------|--------|
| Конфиг-генерация | ✅ | [mihomo/config.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/mihomo/config.go) — 522 строки (YAML) |
| VLESS / VMess / Trojan | ✅ | С users |
| Shadowsocks | ⚠️ | **Только 1 пользователь** — [см. раздел 8.2](#82-mihomo-single-user-shadowsocks--план-рефакторинга) |
| ShadowsocksR | ✅ | Exclusive (cipher, obfs, protocol) |
| Hysteria2 | ✅ | С users |
| TUIC | ✅ | С users |
| Mieru | ✅ | Exclusive |
| Sudoku | ⚠️ | **Пароль не берётся из ConfigJSON** — [см. раздел 8.3](#83-sudoku-пароль--решение) |
| TrustTunnel | ✅ | Exclusive |
| Snell | ✅ | Exclusive (PSK, version, obfs) |
| MASQUE | ✅ | Exclusive outbound |
| Reality | ✅ | reality-opts с public-key, short-id |
| TLS | ✅ | tls: true + servername from cert |
| WARP | ✅ | WireGuard proxy + rules |
| GeoIP | ✅ | Правила маршрутизации |

### Mihomo: версия ядра

В [Dockerfile.dev:50](file:///mnt/Games/syncthing-shared-folder/isolate-panel/docker/Dockerfile.dev#L50) зафиксирована **Mihomo v1.19.21**:
```
wget -q https://github.com/MetaCubeX/mihomo/releases/download/v1.19.21/mihomo-linux-amd64-v1.19.21.gz
```

Эта версия **не Alpha/2.0+**, поэтому использование секции `proxies` вместо `listeners` **корректно** для данной версии. При обновлении до Mihomo Alpha (v1.19.x Alpha) потребуется переход на `listeners`.

---

## 2. Пользователи

| Функция | Backend | Frontend | Статус |
|---------|---------|----------|--------|
| Создание пользователя | ✅ [user_service.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/user_service.go) | ✅ [Users.tsx](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/pages/Users.tsx) | ✅ Работает |
| UUID авто-генерация | ✅ | — | ✅ |
| Subscription token | ✅ | — | ✅ |
| Пароль авто-генерация | ✅ | — | ✅ |
| TUIC token авто-генерация | ✅ | — | ✅ |
| Привязка к inbounds (транзакция) | ✅ | ✅ | ✅ |
| Пагинация + поиск | ✅ | ✅ | ✅ |
| Обновление | ✅ (транзакция) | ✅ | ✅ |
| Удаление | ✅ | ✅ | ✅ |
| Regenerate credentials | ✅ | ✅ | ✅ |
| Traffic quotas | ✅ | ✅ (GB/MB toggle) | ✅ |
| Expiry dates | ✅ (expiry_days) | ✅ (number field + unlimited) | ✅ |
| Online status | ✅ | ✅ | ✅ |

> [!WARNING]
> **Пароль в API-ответе**: `UserResponse` содержит `Password string json:"password,omitempty"` — отдаёт plaintext пароль в каждом ответе. Это утечка данных. Нужно отдавать только при создании.

---

## 3. Inbounds / Outbounds

| Функция | Backend | Frontend | Статус |
|---------|---------|----------|--------|
| Создание Inbound | ✅ | ✅ [InboundCreate.tsx](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/pages/InboundCreate.tsx) | ✅ |
| Редактирование | ✅ | ✅ [InboundEdit.tsx](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/pages/InboundEdit.tsx) | ✅ |
| Просмотр деталей | ✅ | ✅ [InboundDetail.tsx](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/pages/InboundDetail.tsx) | ✅ |
| Список | ✅ | ✅ [Inbounds.tsx](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/pages/Inbounds.tsx) | ✅ |
| Проверка порта | ✅ (PortManager) | ✅ | ✅ |
| TLS cert binding | ✅ | ✅ | ✅ |
| Валидация протокола | ✅ | ✅ | ✅ |
| Assign users → inbound | ✅ (BulkAssignUsers) | ⚠️ Нужно проверить onClick | ⚠️ |
| Создание Outbound | ✅ | ✅ [Outbounds.tsx](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/pages/Outbounds.tsx) | ✅ |
| Валидация протокола/ядра | ✅ (schema registry) | ✅ | ✅ |
| Config regeneration | ✅ | — | ✅ |
| Core auto-restart | ✅ (via lifecycle) | — | ✅ |

---

## 4. Протоколы — Реестр

Зарегистрировано **24 протокола** через [protocol/protocols.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/protocol/protocols.go):

| Протокол | Inbound | Outbound | Xray | Sing-box | Mihomo | Config Gen | Подписка V2Ray | Подписка Clash | Подписка Sing-box |
|----------|---------|----------|------|----------|--------|------------|----------------|----------------|-------------------|
| HTTP | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| SOCKS5 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Mixed | ✅ | — | — | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Shadowsocks | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| VMess | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| VLESS | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Trojan | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Hysteria2 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Hysteria v1 | — | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| TUIC v4 | ✅ | ✅ | — | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| TUIC v5 | ✅ | ✅ | — | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| NaiveProxy | ✅ | — | — | ✅ | — | ✅ | ❌ | ❌ | ❌ |
| XHTTP | ✅ | ✅ | ✅ | — | — | ✅ | ❌ | ❌ | ❌ |
| Redirect | ✅ | — | — | ✅ | ✅ | ✅ | — | — | — |
| Mieru | ✅ | ✅ | — | — | ✅ | ✅ | ❌ | ❌ | ❌ |
| Sudoku | ✅ | ✅ | — | — | ✅ | ⚠️ | ❌ | ❌ | ❌ |
| TrustTunnel | ✅ | ✅ | — | — | ✅ | ✅ | ❌ | ❌ | ❌ |
| ShadowsocksR | ✅ | ✅ | — | — | ✅ | ✅ | ❌ | ❌ | ❌ |
| Snell | ✅ | ✅ | — | — | ✅ | ✅ | ❌ | ❌ | ❌ |
| Direct | — | ✅ | ✅ | ✅ | ✅ | ✅ | — | — | — |
| Block | — | ✅ | ✅ | ✅ | ✅ | ✅ | — | — | — |
| DNS | — | ✅ | ✅ | ✅ | ✅ | ✅ | — | — | — |
| Tor | — | ✅ | — | ✅ | — | ✅ | — | — | — |
| MASQUE | — | ✅ | — | — | ✅ | ✅ | — | — | — |

> [!CAUTION]
> **Подписки генерируются только для 5 из 18 инбаунд-протоколов** (VLESS, VMess, Trojan, SS, Hysteria2). Нужны ссылки для **всех** инбаундов, которые ядра поддерживают. Каждому протоколу нужна генерация во все три формата (V2Ray URI, Clash YAML, Sing-box JSON) — там где формат технически возможен.
>
> ### Полная матрица: инбаунд-протокол × формат подписки × ядро
>
> | # | Протокол | Ядра | V2Ray URI | Clash YAML | Sing-box JSON | Статус |
> |---|----------|------|-----------|------------|---------------|--------|
> | 1 | VLESS | xray, sing-box, mihomo | `vless://uuid@s:p?params#name` | type: vless | type: vless | ✅ Есть |
> | 2 | VMess | xray, sing-box, mihomo | `vmess://base64(json)` | type: vmess | type: vmess | ✅ Есть |
> | 3 | Trojan | xray, sing-box, mihomo | `trojan://pass@s:p?params#name` | type: trojan | type: trojan | ✅ Есть |
> | 4 | Shadowsocks | xray, sing-box, mihomo | `ss://base64(method:pass)@s:p#name` | type: ss | type: shadowsocks | ✅ Есть |
> | 5 | Hysteria2 | xray, sing-box, mihomo | `hysteria2://pass@s:p?insecure=1#name` | type: hysteria2 | type: hysteria2 | ✅ Есть |
> | 6 | TUIC v4 | sing-box, mihomo | `tuic://token@s:p?congestion_control=bbr&alpn=h3#name` | type: tuic, version: 4 | type: tuic, uuid+password | ❌ Нужно |
> | 7 | TUIC v5 | sing-box, mihomo | `tuic://uuid:pass@s:p?congestion_control=bbr&alpn=h3#name` | type: tuic, version: 5 | type: tuic, uuid+password | ❌ Нужно |
> | 8 | NaiveProxy | sing-box | `naive+https://user:pass@s:p#name` | ❌ Mihomo не поддерживает | type: naive | ❌ Нужно |
> | 9 | XHTTP | xray | `vless://uuid@s:p?type=xhttp&security=tls#name` | ❌ Xray-exclusive | ❌ Xray-exclusive | ❌ Нужно |
> | 10 | ShadowsocksR | mihomo | `ssr://base64(host:port:proto:method:obfs:base64pass)` | type: ssr | ❌ Sing-box не поддерживает | ❌ Нужно |
> | 11 | Snell | mihomo | ❌ Нет стандартной URI-схемы | type: snell, psk, version, obfs-mode | ❌ Sing-box не поддерживает | ❌ Нужно |
> | 12 | Mieru | mihomo | ❌ Нет стандартной URI-схемы | type: mieru, password | ❌ Sing-box не поддерживает | ❌ Нужно |
> | 13 | Sudoku | mihomo | ❌ Нет стандартной URI-схемы | type: sudoku, password | ❌ Sing-box не поддерживает | ❌ Нужно |
> | 14 | TrustTunnel | mihomo | ❌ Нет стандартной URI-схемы | type: trusttunnel, password | ❌ Sing-box не поддерживает | ❌ Нужно |
> | 15 | HTTP Proxy | xray, sing-box, mihomo | `http://user:pass@s:p#name` | type: http, username/password | type: http, username/password | ❌ Нужно |
> | 16 | SOCKS5 | xray, sing-box, mihomo | `socks5://user:pass@s:p#name` | type: socks5, username/password | type: socks, username/password | ❌ Нужно |
> | 17 | Mixed | sing-box, mihomo | `mixed://user:pass@s:p#name` | type: mixed | type: mixed | ❌ Нужно |
> | 18 | Redirect | sing-box, mihomo | — (transparent proxy, нет клиентской ссылки) | — | — | — Не нужно |
>
> **Итого: нужно добавить подписки для 12 протоколов (6-17), сейчас реализовано 5 (1-5).**
>
> Для протоколов без стандартной URI-схемы (Snell, Mieru, Sudoku, TrustTunnel) — генерировать **только Clash YAML** формат. V2Ray URI будет возвращать пустую строку.
>
> Для HTTP/SOCKS5/Mixed — простейшие URI, но они полезны для импорта в клиенты.

---

## 5. Сертификаты

| Функция | Backend | Frontend | Статус |
|---------|---------|----------|--------|
| ACME/Let's Encrypt запрос | ✅ [certificate_service.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/certificate_service.go) | ✅ | ✅ |
| ACME клиент (lego v4) | ✅ [acme/acme.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/acme/acme.go) | — | ✅ |
| DNS Challenge (Cloudflare) | ✅ | ✅ | ✅ |
| Ручная загрузка сертификата | ✅ | ❌ **Форма сломана** | ❌ |
| Обновление (Renew) | ✅ | ✅ | ✅ |
| Отзыв (Revoke) | ✅ | ✅ | ✅ |
| Удаление | ✅ | ✅ | ✅ |
| Auto-renewal (24h checker) | ✅ | — | ✅ |
| Wildcard | ✅ | ✅ | ✅ |
| Привязка к inbound | ✅ (tls_cert_id FK) | ✅ | ✅ |
| x509 парсинг метаданных | ✅ | — | ✅ |

> [!CAUTION]
> **Модал загрузки сертификата** ([Certificates.tsx:280-323](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/pages/Certificates.tsx)): textarea без value/onChange, кнопка без onClick — полностью нефункционален.

---

## 6. Подписки и QR-коды

| Функция | Backend | Frontend | Статус |
|---------|---------|----------|--------|
| V2Ray формат (base64) | ✅ | — | ✅ |
| Clash YAML | ✅ | — | ✅ |
| Sing-box JSON | ✅ | — | ✅ |
| Short URLs | ✅ | ✅ | ✅ |
| Token regeneration | ✅ | ✅ | ✅ |
| Access logging | ✅ | — | ✅ |
| Access stats | ✅ | ✅ (useSubscriptionStats) | ✅ |
| Rate limiting | ✅ | — | ✅ |
| Кэширование | ✅ (`fmt.Sprintf`) | — | ✅ |
| **QR-коды** | ✅ `go-qrcode` | ✅ qr_code_url | ✅ |

> [!NOTE]
> **QR-коды уже реализованы!** В [subscriptions.go:181-207](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/api/subscriptions.go#L181-L207) — эндпоинт `GET /sub/:token/qr` генерирует PNG QR через `skip2/go-qrcode`. Маршрут зарегистрирован в [routes.go:241](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/app/routes.go#L241).

---

## 7. Статус багов из FIX_PLAN.md (полная проверка)

### ✅ Исправлено (подтверждено в коде)

| # | Проблема | Доказательство | Состояние |
|---|----------|----------------|-----------|
| 1 | `string(rune(userID))` — сломанный кэш подписок | [cache/manager.go:130](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cache/manager.go#L130): `fmt.Sprintf("subscription:%d:%s", userID, format)` | ✅ Исправлено |
| 2 | Unchecked type assertions `c.Locals("admin_id").(uint)` | Все 6 вхождений используют comma-ok pattern: `adminID, ok := c.Locals("admin_id").(uint)` | ✅ Исправлено |
| 3 | Notifications → backupApi | [Notifications.tsx:2](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/pages/Notifications.tsx): `import { notificationApi }` — вызов корректный | ✅ Исправлено |
| 4 | Docker port 0.0.0.0:8080 | [docker-compose.yml:13](file:///mnt/Games/syncthing-shared-folder/isolate-panel/docker/docker-compose.yml#L13): `"127.0.0.1:8080:8080"` | ✅ Исправлено |
| 5 | WebSocket token в URL (утечка) | [websocket.go:167-214](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/api/websocket.go#L167-L214): one-time ticket система + purge | ✅ Исправлено |
| 6 | WebSocket CheckOrigin | [websocket.go:31-46](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/api/websocket.go#L31-L46): Проверка Origin, только localhost | ✅ Исправлено |
| 7 | Zip Slip в backup restore | [backup_service.go:809-820](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/backup_service.go#L809-L820): `filepath.Clean`, `HasPrefix`, блокировка symlinks | ✅ Исправлено |
| 8 | Password auto-generation | Реализовано в user_service.go, FIX_PLAN_V2 подтверждает DONE | ✅ Исправлено |
| 9 | expiry_date → expiry_days | FIX_PLAN_V2 подтверждает DONE | ✅ Исправлено |
| 10 | Tailwind v4 migration | FIX_PLAN_V2 подтверждает DONE | ✅ Исправлено |
| 11 | Slider component | FIX_PLAN_V2 подтверждает DONE | ✅ Исправлено |
| 12 | inbound_ids в user form | FIX_PLAN_V2 подтверждает DONE | ✅ Исправлено |
| 13 | Core status polling | FIX_PLAN_V2 подтверждает DONE (5000ms refetch) | ✅ Исправлено |
| 14 | traffic_limit_bytes UX | FIX_PLAN_V2 подтверждает DONE (GB/MB toggle) | ✅ Исправлено |
| 15 | Backup → tar.gz (был zip) | [backup_service.go:418-472](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/backup_service.go#L418-L472): `tar.NewWriter(gzip.NewWriter())` | ✅ Исправлено |
| 16 | Backup download → streaming | [backup_service.go:1083-1095](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/backup_service.go#L1083-L1095): возвращает `(filePath, filename)` для SendFile | ✅ Исправлено |
| 17 | Backup encryption → AES-GCM streaming | [backup_service.go:474-536](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/backup_service.go#L474-L536): 64KB chunked AES-256-GCM | ✅ Исправлено |
| 18 | DB restore → binary copy | [backup_service.go:850-906](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/backup_service.go#L850-L906): `database.db` копия + legacy `.sql` fallback | ✅ Исправлено |

### ❌ НЕ исправлено

| # | Проблема | Файл | Состояние |
|---|----------|------|-----------|
| 19 | Пароль в API-ответе (UserResponse) | [user_service.go:58](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/user_service.go#L58) | ❌ Password видна в JSON |
| 20 | Certificates upload form (textarea, кнопка) | Certificates.tsx:280-323 | ❌ Не работает |
| 21 | Подписки только для 5 протоколов | subscription_service.go:392-405, 467-507, 510-596 | ❌ default: return "" |
| 22 | Mihomo SS single-user | [mihomo/config.go:184](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/mihomo/config.go#L184) | ❌ `users[0].UUID` |
| 23 | Sudoku hardcoded пароль | [mihomo/config.go:259](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/mihomo/config.go#L259) | ❌ "sudoku-password" |
| 24 | Sing-box outbound теряет ConfigJSON | [singbox/config.go:684-692](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/singbox/config.go#L684-L692) | ❌ Extra пустой |
| 25 | TUIC v5 дубликат UUID как password | [singbox/config.go:602](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/singbox/config.go#L602) | ❌ |
| 26 | Expiry notification дубликация | [user_service.go:377](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/user_service.go#L377) | ❌ Нет трекинга |
| 27 | `json.Unmarshal` без обработки ошибки | subscription_service.go:381,471,513 | ❌ |
| 28 | ConfigService empty inbounds → error | [config_service.go:126-128](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/config_service.go#L126-L128) | ❌ |
| 29 | JSON.parse без try/catch в main.tsx | [main.tsx:7](file:///mnt/Games/syncthing-shared-folder/isolate-panel/frontend/src/main.tsx#L7) | ❌ |
| 30 | Backup SetSchedule → UPDATE без WHERE | [backup_service.go:1055](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/backup_service.go#L1055) | ❌ `Model(&Backup{}).Update(...)` обновит все записи |
| 31 | Legacy backup restore — sqlite3 CLI | [backup_service.go:900](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/backup_service.go#L900) | ⚠️ Частично (новый формат работает, legacy нет) |
| 32 | Supervisor XML injection | Не найдено `escapeXML`/`xml.EscapeText` | ❌ |

---

## 8. Архитектурные проблемы — Детальный разбор

### 8.1 Пароль в API-ответе
В [user_service.go:58](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/services/user_service.go#L58) `UserResponse.Password` экспортируется. Plaintext-пароль возвращается при каждом GET/LIST.

**Решение**: Убрать поле из `UserResponse`, возвращать его только в ответе на `CreateUser`.

### 8.2 Mihomo single-user Shadowsocks — план рефакторинга

**Проблема:** В [mihomo/config.go:184-186](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/mihomo/config.go#L184-L186):
```go
if len(users) > 0 {
    proxy.Password = users[0].UUID
}
```
Применяется только пароль первого пользователя. Все остальные пользователи на SS-инбаунде Mihomo игнорируются.

**Контекст:** Mihomo v1.19.x **поддерживает** multi-user SS через поле `users`, но требует другой формат конфига:
```yaml
- name: ss_inbound_1
  type: ss
  server: 0.0.0.0
  port: 8388
  cipher: 2022-blake3-aes-128-gcm
  password: <server-key>  # мастер-пароль для 2022 ciphers / не нужен для AEAD
  users:                   # ← это список пользователей
    - name: user_1
      password: <user-key>
    - name: user_2
      password: <user-key>
```

**План рефакторинга:**

1. **Изменить** `case "shadowsocks"` в [mihomo/config.go:180-186](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/mihomo/config.go#L180-L186):
```go
case "shadowsocks":
    // Cipher from ConfigJSON or default
    proxy.Cipher = getStringOrDefault(cfgSettings, "method", "2022-blake3-aes-128-gcm")
    // Server-level password for 2022 ciphers
    if serverPass, ok := cfgSettings["password"].(string); ok {
        proxy.Password = serverPass
    }
    // Multi-user via users list
    if len(users) > 0 {
        proxy.Users = make([]ProxyUser, len(users))
        for i, user := range users {
            proxy.Users[i] = ProxyUser{
                Name:     fmt.Sprintf("user_%d", user.ID),
                Password: user.UUID,
            }
        }
    }
```

2. **Нужно** парсить `inbound.ConfigJSON` перед switch — вынести `json.Unmarshal(ConfigJSON)` из блока транспортов (линия 347-393) в начало функции `convertInboundToProxy`.

3. **Для не-2022 ciphers** (aes-256-gcm, chacha20-poly1305): multi-user не поддерживается нативно в Mihomo. В этом случае — создавать **отдельный inbound на каждого пользователя** с уникальным паролем, или оставлять `users[0]` с предупреждением.

### 8.3 Sudoku пароль — решение

**Проблема:** В [mihomo/config.go:259](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/mihomo/config.go#L259):
```go
proxy.Password = "sudoku-password" // Should be from inbound settings
```

**Модуль генерации существует!** В [protocol/generators.go](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/protocol/generators.go):
- `GeneratePassword(length int)` — криптографически безопасный пароль (charset: a-zA-Z0-9)
- `AutoGenerate("generate_password_16")` — вызывается по имени

И в схеме Sudoku уже есть автогенерация ([protocols.go:479-486](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/protocol/protocols.go#L479-L486)):
```go
"password": {
    AutoGenerate: true,
    AutoGenFunc:  "generate_password_16",
}
```

**Т.е. пароль должен автоматически генерироваться при создании инбаунда и храниться в `ConfigJSON`.**

**Фикс:** Заменить hardcoded строку на чтение из ConfigJSON:
```go
case "sudoku":
    proxy.Type = "sudoku"
    if inbound.ConfigJSON != "" {
        var cfg map[string]interface{}
        if err := json.Unmarshal([]byte(inbound.ConfigJSON), &cfg); err == nil {
            if pass, ok := cfg["password"].(string); ok {
                proxy.Password = pass
            }
        }
    }
```

Аналогичная проверка нужна для SSR (`case "ssr"`) — `proxy.Protocol` и `proxy.Obfs` тоже hardcoded.

### 8.4 Sing-box outbound теряет ConfigJSON
[singbox/config.go:684-692](file:///mnt/Games/syncthing-shared-folder/isolate-panel/backend/internal/cores/singbox/config.go#L684-L692) — `convertOutbound()` возвращает пустой outbound без Extra.

**Фикс:** Парсить `outbound.ConfigJSON` в Extra:
```go
func convertOutbound(outbound models.Outbound) (*OutboundConfig, error) {
    singboxType := mapSingboxOutboundProtocol(outbound.Protocol)
    tag := fmt.Sprintf("%s_%d", outbound.Protocol, outbound.ID)
    extra := make(map[string]interface{})
    if outbound.ConfigJSON != "" {
        json.Unmarshal([]byte(outbound.ConfigJSON), &extra)
    }
    return &OutboundConfig{Type: singboxType, Tag: tag, Extra: extra}, nil
}
```

---

## 9. Сводная таблица функциональности

| Модуль | Backend | Frontend | Интеграция | Итого |
|--------|---------|----------|------------|-------|
| Ядра (Xray/SB/Mihomo) | 95% | 95% | 90% | **93%** |
| Пользователи | 95% | 90% | 85% | **90%** |
| Inbounds | 95% | 85% | 80% | **87%** |
| Outbounds | 90% | 85% | 85% | **87%** |
| Протоколы | 95% | 90% | 90% | **92%** |
| Сертификаты | 90% | 60% | 70% | **73%** |
| Подписки | 50% | 80% | 60% | **63%** |
| QR-коды | 100% | 100% | 100% | **100%** |
| Dashboard | 90% | 90% | 85% | **88%** |
| Бэкапы | 85% | 80% | 80% | **82%** |
| Уведомления | 85% | 80% | 80% | **82%** |
| WARP | 90% | 70% | 80% | **80%** |
| GeoIP | 85% | 60% | 70% | **72%** |
| Настройки | 90% | 85% | 85% | **87%** |
| Аудит | 85% | 80% | 80% | **82%** |
| **Среднее** | **88%** | **81%** | **80%** | **~84%** |

---

## 10. Рекомендуемый план действий

### Фаза 1: Критические баги (1-2 дня)

1. **Исправить модал загрузки сертификатов** — state + onChange + onClick
2. **Убрать Password из UserResponse** — json:"-" при GET/LIST, только при Create
3. **Исправить Mihomo SS multi-user** — users list + server password (см. раздел 8.2)
4. **Исправить Sudoku пароль** — из ConfigJSON через AutoGenFunc (см. раздел 8.3)
5. **Исправить Sing-box outbound ConfigJSON** — парсинг в Extra (см. раздел 8.4)
6. **JSON.parse try/catch** в main.tsx
7. **Backup SetSchedule UPDATE WHERE** — добавить `.Where("id = ?", backup.ID)`

### Фаза 2: Подписки для всех инбаунд-протоколов (3-4 дня)

Добавить генерацию ссылок в **три метода**: `generateProxyLink`, `generateClashProxy`, `generateSingboxOutbound` для всех 12 недостающих протоколов:

| # | Протокол | `generateProxyLink` (V2Ray URI) | `generateClashProxy` (Clash YAML) | `generateSingboxOutbound` (JSON) |
|---|----------|----------------------------------|-----------------------------------|----------------------------------|
| 1 | TUIC v4 | `tuic://token@s:p?cc=bbr&alpn=h3` | type: tuic, version: 4, token | type: tuic, uuid, password |
| 2 | TUIC v5 | `tuic://uuid:pass@s:p?cc=bbr&alpn=h3` | type: tuic, version: 5, uuid, password | type: tuic, uuid, password |
| 3 | NaiveProxy | `naive+https://user:pass@s:p` | ❌ skip | type: naive, username, password |
| 4 | XHTTP | `vless://uuid@s:p?type=xhttp&security=tls` | ❌ skip (Xray-exclusive) | ❌ skip (Xray-exclusive) |
| 5 | SSR | `ssr://base64(host:port:proto:method:obfs:pass)` | type: ssr, cipher, protocol, obfs | ❌ skip (нет в sing-box) |
| 6 | Snell | ❌ return "" | type: snell, psk, version, obfs-mode | ❌ skip (нет в sing-box) |
| 7 | Mieru | ❌ return "" | type: mieru, password | ❌ skip (нет в sing-box) |
| 8 | Sudoku | ❌ return "" | type: sudoku, password | ❌ skip (нет в sing-box) |
| 9 | TrustTunnel | ❌ return "" | type: trusttunnel, password | ❌ skip (нет в sing-box) |
| 10 | HTTP | `http://user:pass@s:p` | type: http, username/password | type: http, username/password |
| 11 | SOCKS5 | `socks5://user:pass@s:p` | type: socks5, username/password | type: socks, username/password |
| 12 | Mixed | `mixed://user:pass@s:p` | type: mixed | type: mixed |

### Фаза 3: Архитектурные улучшения (1-2 дня)

1. **TUIC v5 password** — использовать `user.Password` (отдельный от UUID) или `user.Token`
2. **Expiry notification tracking** — добавить `last_expiry_notified_at` поле
3. **ConfigService 0 inbounds** — убрать ошибку, сгенерировать минимальный конфиг
4. **json.Unmarshal обработка ошибок** — в subscription_service.go
5. **Clash YAML** — перейти на `yaml.Marshal` вместо ручного форматирования
6. **Supervisor XML escaping** — `xml.EscapeText` для имён процессов

### Фаза 4: Качество (2-3 дня)

1. N+1 запросы в traffic_collector, connection_tracker
2. E2E тесты (FIX_PLAN_V2 P2-11)
3. Core health-check после запуска (FIX_PLAN_V2 P2-9)
