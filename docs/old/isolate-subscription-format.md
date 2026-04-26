# Спецификация формата подписки Isolate

**Версия:** 1  
**Эндпоинт:** `GET /sub/{token}/isolate`  
**Content-Type:** `application/json`  
**Кодировка:** UTF-8, форматирование с отступом в 2 пробела

---

## Обзор

Isolate — кастомный JSON-формат подписки, предназначенный для прокси-клиентов, которым нужно знать, какому ядру (Xray / Sing-box / Mihomo) принадлежит каждый инбаунд. В отличие от форматов V2Ray/Clash/Sing-box, которые «схлопывают» все прокси в единый плоский список, Isolate группирует инбаунды по типу ядра — что позволяет клиенту маршрутизировать каждое соединение через правильный прокси-движок.

Ключевые принципы дизайна:
1. **Группировка по ядру** — клиент точно знает, какой движок использовать для каждого инбаунда
2. **Самодостаточность** — каждый инбаунд содержит и структурированные поля, и готовую к использованию строку `raw_link`
3. **Метаданные профиля** — username, UUID, трафик, срок действия, URL обновления в одном месте
4. **Расширяемость** — версионированная схема, опциональные поля с `omitempty`

---

## Структура верхнего уровня

```json
{
  "version": 1,
  "profile": { ... },
  "cores": {
    "Xray": { "inbounds": [ ... ] },
    "Sing-box": { "inbounds": [ ... ] },
    "Mihomo": { "inbounds": [ ... ] }
  }
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `version` | `int` | Версия схемы. Сейчас всегда `1`. |
| `profile` | `IsolateProfile` | Метаданные аккаунта пользователя. |
| `cores` | `map[string]IsolateCore` | Инбаунды, сгруппированные по отображаемому имени ядра. Ключи: `"Xray"`, `"Sing-box"`, `"Mihomo"`. Отсутствует, если у пользователя нет инбаундов для данного ядра. |

---

## IsolateProfile

```json
{
  "username": "alice",
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "traffic_used": 1073741824,
  "traffic_limit": 536870912000,
  "expire": 1735689600,
  "update_interval_hours": 24,
  "subscription_url": "https://panel.example.com/sub/abc123/isolate"
}
```

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| `username` | `string` | ✅ | Имя пользователя на панели. Используется для авторизации HTTP/SOCKS5/Mixed/Naive. |
| `uuid` | `string` | ✅ | UUID пользователя (v4). Используется как UUID для VLESS/VMess, пароль для Trojan/SS/HY2 и т.д. |
| `traffic_used` | `int64` | ✅ | Потреблено байт. `0` если трафик ещё не использован. |
| `traffic_limit` | `*int64` | ❌ | Лимит трафика в байтах. `null` = безлимит. |
| `expire` | `*int64` | ❌ | Unix-таймстемп окончания подписки. `null` = без срока действия. |
| `update_interval_hours` | `int` | ✅ | Как часто клиент должен переобновлять подписку. Сейчас всегда `24`. |
| `subscription_url` | `string` | ✅ | Полный URL для повторного получения этой Isolate-подписки. |

---

## IsolateCore

```json
{
  "inbounds": [ ... ]
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `inbounds` | `[]IsolateInbound` | Упорядоченный список инбаундов, назначенных этому ядру. |

---

## IsolateInbound

```json
{
  "id": 1,
  "name": "VLESS-Reality-WS",
  "protocol": "vless",
  "server": "example.com",
  "port": 443,
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "password": "",
  "method": "",
  "tls": { ... },
  "transport": { ... },
  "raw_link": "vless://uuid@server:port?..."
}
```

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| `id` | `uint` | ✅ | ID инбаунда в базе данных панели. Уникален в рамках экземпляра панели. |
| `name` | `string` | ✅ | Человекочитаемое имя инбаунда (задаётся админом). |
| `protocol` | `string` | ✅ | Идентификатор протокола. См. [Справочник протоколов](#справочник-протоколов). |
| `server` | `string` | ✅ | Имя хоста или IP сервера. Резолвится: домен сертификата → URL панели → listen address. |
| `port` | `int` | ✅ | Порт сервера (1–65535). |
| `uuid` | `string` | ❌ | UUID пользователя. Присутствует для VLESS, VMess, TUIC, XHTTP, HTTP, SOCKS5, Mixed, Naive. |
| `password` | `string` | ❌ | Пароль авторизации. Присутствует для Trojan, SS, HY2, Hysteria, Snell, SSR, AnyTLS, Mieru, Sudoku, TrustTunnel. |
| `method` | `string` | ❌ | Метод шифрования. Присутствует для Shadowsocks (`aes-256-gcm`, `chacha20-ietf-poly1305` и др.) и SSR. |
| `tls` | `map` | ❌ | Конфигурация TLS/Reality. `null` если TLS не используется. См. [Объект TLS](#объект-tls). |
| `transport` | `map` | ❌ | Конфигурация транспортного уровня. `null` для обычного TCP без flow. См. [Объект Transport](#объект-transport). |
| `raw_link` | `string` | ✅ | Готовая к использованию строка URI в формате V2Ray. Может быть пустой строкой (`""`) для протоколов без URI-схемы (mieru, sudoku, trusttunnel). |

**Какие поля авторизации присутствуют для каждого протокола:**

| Протокол | `uuid` | `password` | `method` | Примечания |
|----------|--------|------------|----------|------------|
| vless | ✅ user.UUID | ❌ | ❌ | flow в объекте transport |
| vmess | ✅ user.UUID | ❌ | ❌ | |
| trojan | ❌ | ✅ user.UUID | ❌ | |
| shadowsocks | ❌ | ✅ из конфига или user.UUID | ✅ из конфига или `aes-256-gcm` | |
| hysteria2 | ❌ | ✅ из конфига или user.UUID | ❌ | obfs в raw_link |
| tuic_v4 | ✅ user.UUID | ✅ user.Token или user.UUID | ❌ | |
| tuic_v5 | ✅ user.UUID | ✅ user.Token или user.UUID | ❌ | |
| anytls | ❌ | ✅ из конфига или user.UUID | ❌ | только sing-box |
| hysteria | ❌ | ✅ auth_str из конфига или user.UUID | ❌ | устарел; sing-box+mihomo |
| snell | ❌ | ✅ user.Token или user.UUID | ❌ | только mihomo |
| ssr | ❌ | ✅ из конфига или user.UUID | ✅ cipher из конфига или `chacha20-poly1305` | только mihomo |
| xhttp | ✅ user.UUID | ❌ | ❌ | только xray |
| http | ✅ user.Username | ✅ user.UUID | ❌ | |
| socks5 | ✅ user.Username | ✅ user.UUID | ❌ | |
| mixed | ✅ user.Username | ✅ user.UUID | ❌ | |
| naive | ✅ user.Username | ✅ user.UUID | ❌ | |
| mieru | ❌ | ✅ из конфига или user.UUID | ❌ | только mihomo |
| sudoku | ❌ | ✅ из конфига или user.UUID | ❌ | только mihomo |
| trusttunnel | ❌ | ✅ из конфига или user.UUID | ❌ | только mihomo |

---

## Объект TLS

Присутствует, когда на инбаунде включён `tls_enabled` или `reality_enabled`. В противном случае `null`.

### Стандартный TLS

```json
{
  "enabled": true,
  "sni": "example.com"
}
```

### Reality (XTLS-Reality)

```json
{
  "enabled": true,
  "sni": "sni.example.com",
  "reality": true,
  "public_key": "base64-кодированный-публичный-ключ",
  "short_id": "hex-строка",
  "fingerprint": "chrome"
}
```

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| `enabled` | `bool` | ✅ | Всегда `true`, когда объект TLS присутствует. |
| `sni` | `string` | ❌ | Домен Server Name Indication. Из домена сертификата (TLS) или serverNames[0] (Reality). |
| `reality` | `bool` | ❌ | `true` при использовании XTLS-Reality. |
| `public_key` | `string` | ❌ | Публичный ключ Reality (pbk). Base64-кодированный X25519 публичный ключ. |
| `short_id` | `string` | ❌ | Short ID Reality. Hex-строка, первый элемент из массива `shortIds` сервера. |
| `fingerprint` | `string` | ❌ | Отпечаток uTLS. По умолчанию `"chrome"`, если не настроен. Распространённые значения: `chrome`, `firefox`, `safari`, `edge`, `ios`. |

---

## Объект Transport

Присутствует, когда транспорт не является обычным TCP (без flow) или когда задан flow. `null` для обычного TCP без flow.

### TCP с flow (например XTLS Vision)

```json
{
  "type": "tcp",
  "flow": "xtls-rprx-vision"
}
```

### WebSocket

```json
{
  "type": "websocket",
  "path": "/ws-path",
  "host": "ws.example.com"
}
```

### gRPC

```json
{
  "type": "grpc",
  "service_name": "grpc-service-name"
}
```

### HTTP/2 (h2)

```json
{
  "type": "http",
  "path": "/h2-path",
  "host": "h2.example.com"
}
```

### HTTPUpgrade

```json
{
  "type": "httpupgrade",
  "path": "/upgrade-path",
  "host": "upgrade.example.com"
}
```

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| `type` | `string` | ✅ | Тип транспорта: `"tcp"`, `"websocket"`, `"grpc"`, `"http"`, `"httpupgrade"`. |
| `flow` | `string` | ❌ | Управление потоком. Только для VLESS+TCP. Значения: `"xtls-rprx-vision"`, `"xtls-rprx-vision-udp443"`. |
| `path` | `string` | ❌ | URL-путь. Присутствует для websocket, http, httpupgrade. |
| `host` | `string` | ❌ | Заголовок Host. Присутствует для websocket, http, httpupgrade. |
| `service_name` | `string` | ❌ | Имя gRPC-сервиса. Присутствует для grpc. |

---

## Справочник протоколов

### Поддерживаемые протоколы и привязка к ядрам

| Протокол | Xray | Sing-box | Mihomo | Ключ в Isolate | Формат raw_link V2Ray |
|----------|------|----------|--------|-----------------|----------------------|
| vless | ✅ | ✅ | ❌ | `vless` | `vless://uuid@server:port?...` |
| vmess | ✅ | ✅ | ❌ | `vmess` | `vmess://base64` |
| trojan | ✅ | ✅ | ❌ | `trojan` | `trojan://uuid@server:port?...` |
| shadowsocks | ✅ | ✅ | ❌ | `shadowsocks` | `ss://base64@server:port#name` |
| hysteria2 | ✅ | ✅ | ✅ | `hysteria2` | `hysteria2://uuid@server:port?...` |
| tuic_v4 | ❌ | ✅ | ❌ | `tuic_v4` | `tuic://token@server:port?...` |
| tuic_v5 | ❌ | ✅ | ✅ | `tuic_v5` | `tuic://uuid:password@server:port?...` |
| anytls | ❌ | ✅ | ❌ | `anytls` | `anytls://password@server:port?...` |
| xhttp | ✅ | ❌ | ❌ | `xhttp` | `vless://uuid@server:port?type=xhttp&...` |
| hysteria (v1) | ❌ | ✅ | ✅ | `hysteria` | ⚠️ устарел |
| snell | ❌ | ❌ | ✅ | `snell` | `snell://psk@server:port?version=3&...` |
| ssr | ❌ | ❌ | ✅ | `ssr` | `ssr://base64` |
| http | ✅ | ❌ | ❌ | `http` | `http://user:pass@server:port#name` |
| socks5 | ✅ | ❌ | ❌ | `socks5` | `socks5://user:pass@server:port#name` |
| mixed | ✅ | ✅ | ✅ | `mixed` | `mixed://user:pass@server:port#name` |
| naive | ❌ | ✅ | ❌ | `naive` | `naive+https://user:pass@server:port#name` |
| mieru | ❌ | ❌ | ✅ | `mieru` | `""` (нет URI-схемы) |
| sudoku | ❌ | ❌ | ✅ | `sudoku` | `""` (нет URI-схемы) |
| trusttunnel | ❌ | ❌ | ✅ | `trusttunnel` | `""` (нет URI-схемы) |

---

## Полный пример

Пользователь с 4 инбаундами на 3 ядрах:

```json
{
  "version": 1,
  "profile": {
    "username": "alice",
    "uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "traffic_used": 5368709120,
    "traffic_limit": 107374182400,
    "expire": 1751328000,
    "update_interval_hours": 24,
    "subscription_url": "https://panel.example.com/sub/dG9rZW4xMjM0/isolate"
  },
  "cores": {
    "Xray": {
      "inbounds": [
        {
          "id": 1,
          "name": "VLESS-Reality-Vision",
          "protocol": "vless",
          "server": "proxy.example.com",
          "port": 443,
          "uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
          "tls": {
            "enabled": true,
            "sni": "www.microsoft.com",
            "reality": true,
            "public_key": "aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789abcdefg",
            "short_id": "a1b2c3d4e5",
            "fingerprint": "chrome"
          },
          "transport": {
            "type": "tcp",
            "flow": "xtls-rprx-vision"
          },
          "raw_link": "vless://f47ac10b-58cc-4372-a567-0e02b2c3d479@proxy.example.com:443?security=reality&pbk=aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789abcdefg&sid=a1b2c3d4e5&fp=chrome&sni=www.microsoft.com&type=tcp&flow=xtls-rprx-vision#[Xray]VLESS-Reality-Vision"
        },
        {
          "id": 5,
          "name": "XHTTP-CDN",
          "protocol": "xhttp",
          "server": "cdn.example.com",
          "port": 443,
          "uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
          "tls": {
            "enabled": true,
            "sni": "cdn.example.com"
          },
          "transport": null,
          "raw_link": "vless://f47ac10b-58cc-4372-a567-0e02b2c3d479@cdn.example.com:443?type=xhttp&security=tls&sni=cdn.example.com&path=%2Fxhttp#[Xray]XHTTP-CDN"
        }
      ]
    },
    "Sing-box": {
      "inbounds": [
        {
          "id": 2,
          "name": "HY2-Main",
          "protocol": "hysteria2",
          "server": "hy2.example.com",
          "port": 8443,
          "password": "hy2-secret-password",
          "tls": {
            "enabled": true,
            "sni": "hy2.example.com"
          },
          "transport": null,
          "raw_link": "hysteria2://f47ac10b-58cc-4372-a567-0e02b2c3d479@hy2.example.com:8443?insecure=1&sni=hy2.example.com#[Sing-box]HY2-Main"
        },
        {
          "id": 3,
          "name": "AnyTLS-Direct",
          "protocol": "anytls",
          "server": "anytls.example.com",
          "port": 443,
          "password": "anytls-password-123",
          "tls": {
            "enabled": true,
            "sni": "anytls.example.com"
          },
          "transport": null,
          "raw_link": "anytls://anytls-password-123@anytls.example.com:443?security=tls&sni=anytls.example.com#[Sing-box]AnyTLS-Direct"
        }
      ]
    },
    "Mihomo": {
      "inbounds": [
        {
          "id": 4,
          "name": "SSR-Old",
          "protocol": "ssr",
          "server": "ssr.example.com",
          "port": 8388,
          "password": "ssr-password",
          "method": "chacha20-poly1305",
          "tls": null,
          "transport": null,
          "raw_link": "ssr://..."
        }
      ]
    }
  }
}
```

---

## Руководство по реализации клиента

### 1. Получение подписки

```
GET https://panel.example.com/sub/{token}/isolate
```

Ответ — JSON (НЕ base64-кодированный, в отличие от формата V2Ray). Парсится напрямую.

### 2. Обработка структуры

```
Спарсить JSON → Прочитать profile → Перебрать cores → Для каждого ядра перебрать inbounds
```

### 3. Подключение с использованием данных инбаунда

Каждый инбаунд предоставляет два способа подключения:

**Вариант А: Использовать структурированные поля** (рекомендуется для нативной реализации)
- Прочитать `protocol`, `server`, `port`, `uuid`/`password`/`method`, `tls`, `transport`
- Построить собственную конфигурацию подключения из этих полей
- Даёт полный контроль над параметрами соединения

**Вариант Б: Использовать `raw_link`** (быстрая интеграция)
- Распарсить `raw_link` как стандартный V2Ray/proxy URI
- Тот же формат, что используется в V2RayN, Clash, Shadowrocket и др.
- Пустая строка (`""`) означает отсутствие URI-схемы (mieru, sudoku, trusttunnel) — в этом случае используйте структурированные поля

### 4. Маршрутизация по ядру

Маппинг ключа ядра на прокси-движок:
- `"Xray"` → Xray-core
- `"Sing-box"` → ядро sing-box
- `"Mihomo"` → ядро Mihomo/Clash.Meta

Если протокол инбаунда поддерживается только одним ядром (например, `anytls` → только sing-box), он появится ТОЛЬКО под ключом этого ядра. Группировка гарантирует корректность.

### 5. Логика обновления

- Переобновлять подписку каждые `profile.update_interval_hours` часов
- Использовать `profile.subscription_url` как URL для запроса
- При обновлении — полностью заменить все сохранённые инбаунды (сервер мог добавить/удалить/изменить инбаунды)
- Если `profile.expire` задан и `now > profile.expire` — подписка истекла
- Если `profile.traffic_limit` задан и `profile.traffic_used >= profile.traffic_limit` — трафик исчерпан

### 6. Учёт трафика

- `profile.traffic_used` обновляется при каждом запросе подписки
- Клиент может сравнить локальный трафик с серверным для выявления расхождений
- Значение указано в **байтах** (не в битах, не в МБ)

---

## Логика резолва адреса сервера

Поле `server` определяется со следующим приоритетом:

1. **Домен сертификата** — если у инбаунда привязан `tls_cert_id`, используется домен сертификата
2. **Имя хоста из URL панели** — если сертификата нет, извлекается hostname из настроенного базового URL панели
3. **Фоллбэк** — литеральная строка `"SERVER_IP"`, если URL панели указывает на localhost

Если `listen_address` инбаунда — конкретный IP (не `0.0.0.0`, `::` или пустая строка), этот IP используется напрямую.

---

## Отличия от других форматов

| Аспект | V2Ray | Clash | Sing-box | Isolate |
|--------|-------|-------|----------|---------|
| Кодировка | Base64 | YAML | JSON | JSON |
| Группировка по ядру | ❌ плоский список | ❌ плоский список | ❌ плоский список | ✅ по ядру |
| Метаданные пользователя | ❌ | ❌ | ❌ | ✅ profile |
| Трафик/срок действия | ❌ | ❌ | ❌ | ✅ встроены |
| URL обновления | ❌ | ❌ | ❌ | ✅ subscription_url |
| Структурированные поля | ❌ только URI | ✅ YAML-поля | ✅ JSON-поля | ✅ JSON + URI |
| Детали авторизации протокола | встроены в URI | YAML-поля | JSON-поля | явные uuid/password/method |

---

## История изменений

- **v1** (2024) — Первоначальный формат. Инбаунды сгруппированы по ядру, метаданные профиля, объекты TLS/Reality/transport, фоллбэк raw_link.
