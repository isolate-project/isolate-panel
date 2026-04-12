# Isolate Panel — Production Readiness: Проблемы и решения (v5)

## Обзор

**15 проблем**, общая оценка: **~4 часа**.

**Изменения в v5:**
- **#11f**: Разъяснение SNI vs ServerName для всех протоколов
- **#12**: CHANGELOG.md тоже заменить
- **#13**: CLOUDFLARE_API_TOKEN — объяснение назначения; .env.example — анализ полноты

---

## 🔴 Проблема 1: Password `json:"-"`

```diff
- Password string `gorm:"not null" json:"password"`
+ Password string `gorm:"not null" json:"-"`
```

> **Сложность**: 1 строка.

---

## 🔴 Проблема 3: Захардкоженные API-ключи Sing-box / Mihomo

| Ядро | Протокол | Аутентификация | Проблема? |
|------|---------|---------------|-----------|
| **Xray** | gRPC dokodemo-door :10085 | Нет секрета (by design) | ❌ |
| **Sing-box** | HTTP REST :9090 | `"isolate-singbox-key"` | ✅ Захардкожен |
| **Mihomo** | HTTP REST :9091 | `"isolate-mihomo-key"` | ✅ Захардкожен |

**Решение**: Автогенерация в `docker-entrypoint.sh` → `/app/data/.core-secrets`.

> **Сложность**: ~45 мин.

---

## 🔴 Проблема 4: Swagger в production

```go
if !cfg.IsProduction() { /* swagger routes */ }
```

> **Сложность**: ~5 мин.

---

## 🔴 Проблема 11a: `SERVER_IP` placeholder + panelURL dead code

### Как это работает сейчас

```
Пользователь создаёт Inbound:
    listen_address = "0.0.0.0"   ← bind-адрес (на каком интерфейсе слушать)
    port = 443
    tls_enabled = true
    tls_cert_id = 5              ← сертификат для proxy.example.com
         │
         ▼
Генератор подписок (subscription_service.go:390-393):
    serverAddr = inbound.ListenAddress     // = "0.0.0.0"
    if serverAddr == "0.0.0.0" || "" {
        serverAddr = "SERVER_IP"            // ← литерал-placeholder!
    }
         │
         ▼
Результат: vless://uuid@SERVER_IP:443     ← НЕРАБОЧАЯ ССЫЛКА
```

**Проблемы найдены:**
1. `ListenAddress` — это **bind-адрес** (какой сетевой интерфейс слушать), а не публичный адрес сервера
2. `panelURL` — хранится в `SubscriptionService.panelURL`, но **НИГДЕ не используется** в `generateProxyLink()` (строка 390). Это **мёртвый код**
3. `cert.Domain` — доступен через `inbound.TLSCertID → certificates.domain`, но тоже **не используется** в генераторе

### Как должно работать

Адрес сервера в подписках должен определяться **каскадно**:

```
1. cert.Domain (из привязанного сертификата) — ЛУЧШИЙ вариант
   → если к inbound привязан сертификат для proxy.example.com,
     то server = proxy.example.com
     
2. panelURL hostname (fallback) — если сертификата нет
   → если panelURL = "https://panel.example.com",
     то server = panel.example.com

3. "SERVER_IP" (последний fallback) — если ничего не задано
```

**Почему cert.Domain — приоритет #1?**

Потому что панель **уже управляет сертификатами**. Когда пользователь:
1. Запрашивает сертификат через ACME или загружает вручную
2. Привязывает сертификат к inbound через `tls_cert_id`
3. → Генератор подписок должен автоматически взять домен из привязанного сертификата

Это **устраняет необходимость** в `APP_PANEL_URL` для основного сценария (inbound с TLS-сертификатом).

### Решение

```go
// subscription_service.go — заменить строки 390-393
func (s *SubscriptionService) resolveServerAddr(inbound models.Inbound) string {
    addr := inbound.ListenAddress
    if addr == "0.0.0.0" || addr == "" || addr == "::" {
        // 1. Домен из привязанного сертификата
        if inbound.TLSCertID != nil {
            var cert models.Certificate
            if s.db.First(&cert, *inbound.TLSCertID).Error == nil && cert.Domain != "" {
                return cert.Domain
            }
        }
        // 2. Hostname из panelURL
        if u, err := url.Parse(s.panelURL); err == nil && u.Hostname() != "" &&
           u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" {
            return u.Hostname()
        }
        // 3. Последний fallback
        return "SERVER_IP"
    }
    return addr
}
```

### APP_PANEL_URL — нужен или нет?

| Сценарий | Нужен ли APP_PANEL_URL? | Почему |
|---------|:---:|--------|
| Inbound с TLS-сертификатом | ❌ | Домен берётся из `cert.Domain` автоматически |
| Inbound без TLS (Shadowsocks plain) | ✅ | Нет сертификата → нужен fallback |
| Inbound c Reality | ❌ | Reality SNI из `serverNames`, а server address из cert или panelURL |
| Все inbounds с сертификатами | ❌ | Всё автоматически |

**Вывод**: `APP_PANEL_URL` нужен **только** как fallback для inbound'ов без TLS-сертификатов. В типичном деплое с доменом и сертификатами — **не обязателен**.

В `.env.example` добавить как **опциональный** с пояснением:

```bash
# Panel URL (optional - used as fallback for subscription links when no certificate is bound)
# If your inbounds have TLS certificates, the domain is extracted automatically
# APP_PANEL_URL=https://your-server-ip-or-domain
```

> **Сложность**: ~15 мин (fix resolveServerAddr + activate dead panelURL).

---

## 🟡 Проблема 11f: SNI в подписках — разъяснение и реализация

### SNI vs ServerName — что из себя представляет

| Понятие | Где используется | Что означает | Кто задаёт |
|---------|-----------------|-------------|------------|
| **ServerName** (серверная сторона) | Конфиг ядра: `tlsSettings.serverName` | Домен из TLS-сертификата. Ядро использует его для правильного TLS handshake | **Автоматически** из `cert.Domain` при привязке сертификата к inbound |
| **SNI** (клиентская сторона) | Подписочная ссылка: `&sni=domain.com` | Домен, который **клиент** отправит в TLS Client Hello. Сервер выбирает сертификат по SNI | **Должен совпадать** с ServerName сервера = `cert.Domain` |
| **Reality serverNames** | `reality_config_json.serverNames` | Домены для **маскировки под легитимный сайт** (google.com, etc.) | **Пользователь вручную** — это НЕ из сертификата |

### Как это работает в проекте сейчас

**На стороне ядра (серверный конфиг)** — ✅ работает правильно:

```
Inbound (tls_enabled=true, tls_cert_id=5)
     │
     ▼
Certificate (id=5, domain="proxy.example.com", cert_path="...", key_path="...")
     │
     ▼
Xray config:   streamSettings.tlsSettings.serverName = "proxy.example.com"   ✅
Sing-box:      tls.server_name = "proxy.example.com"                         ✅
```

**На стороне подписки (клиентский конфиг)** — ❌ НЕ подставляется:

```
VLESS link:  vless://uuid@1.2.3.4:443?security=tls&type=tcp#MyServer
                                      ^^^^^^^^^^^^
                                      sni НЕ указан — клиент отправит
                                      SNI = server address (IP)
                                      → TLS handshake FAIL если сервер
                                        ожидает SNI = domain
```

### Два сценария подключения

#### Сценарий 1: Обычный TLS (TLSEnabled=true, RealityEnabled=false)

Сервер предъявляет реальный сертификат на `proxy.example.com`. Клиент **должен** отправить SNI = `proxy.example.com`.

**Что ставить в подписку**: `sni = cert.Domain` (из привязанного сертификата)

```go
// Для V2Ray links:
if inbound.TLSEnabled && !inbound.RealityEnabled && inbound.TLSCertID != nil {
    var cert models.Certificate
    if s.db.First(&cert, *inbound.TLSCertID).Error == nil {
        params.Set("sni", cert.Domain)      // клиент отправит правильный SNI
    }
}

// Для Clash:
if sni := ...; sni != "" {
    entry += fmt.Sprintf("    servername: %s\n", sni)  // Clash использует "servername"
}

// Для Sing-box:
tlsConfig["server_name"] = cert.Domain
```

#### Сценарий 2: Reality (RealityEnabled=true)

Reality **не использует реальный сертификат**. Вместо этого он маскируется под другой домен (google.com, microsoft.com и т.д.). `serverNames` в Reality — это домены, которые пользователь выбирает вручную для маскировки.

**Что ставить в подписку**: `sni = reality_config_json.serverNames[0]` (задан пользователем при создании inbound)

```go
if inbound.RealityEnabled && inbound.RealityConfigJSON != "" {
    var realityCfg map[string]interface{}
    json.Unmarshal([]byte(inbound.RealityConfigJSON), &realityCfg)
    if serverNames, ok := realityCfg["serverNames"].([]interface{}); ok && len(serverNames) > 0 {
        if sn, ok := serverNames[0].(string); ok {
            params.Set("sni", sn)
            // + для Reality-специфичные параметры:
            params.Set("pbk", realityCfg["publicKey"])   // public key
            params.Set("sid", realityCfg["shortIds"][0])  // short ID
            params.Set("fp", "chrome")                     // fingerprint
        }
    }
}
```

### Какие протоколы затрагиваются

| Протокол | Использует TLS? | Нужен SNI? | Нужен Reality SNI? |
|---------|----------------|-----------|-------------------|
| VLESS | По выбору | ✅ Если TLS | ✅ Если Reality |
| VMess | По выбору | ✅ Если TLS | ❌ (VMess не поддерж. Reality) |
| Trojan | Всегда TLS | ✅ **Обязательно** | ✅ Если Reality |
| Shadowsocks | Нет | ❌ | ❌ |
| Hysteria2 | **Всегда** (QUIC = TLS) | ✅ **Обязательно** | ❌ (не подд. Reality) |
| TUIC v4/v5 | **Всегда** (QUIC = TLS) | ✅ **Обязательно** | ❌ (не подд. Reality) |
| NaiveProxy | Всегда HTTPS | ✅ По умолчанию server address | ❌ |
| HTTP/SOCKS5/Mixed | По выбору | ⚠️ Если TLS | ❌ |

**Hysteria2 и TUIC** — особый случай: они работают поверх QUIC, который **всегда** использует TLS. Даже если в UI `tls_enabled` не отмечен, Hysteria2/TUIC **всё равно** создают TLS-подключение. Поэтому для них SNI тоже обязателен если есть сертификат.

### Подробная реализация

#### Шаг 1: Helper для получения SNI

```go
// subscription_service.go
type inboundTLSInfo struct {
    SNI    string // домен для SNI (из сертификата или Reality)
    IsTLS  bool   // включён ли TLS
}

func (s *SubscriptionService) getInboundTLSInfo(inbound models.Inbound) inboundTLSInfo {
    info := inboundTLSInfo{IsTLS: inbound.TLSEnabled}

    // Reality — SNI из serverNames (задан пользователем)
    if inbound.RealityEnabled && inbound.RealityConfigJSON != "" {
        var realityCfg map[string]interface{}
        if json.Unmarshal([]byte(inbound.RealityConfigJSON), &realityCfg) == nil {
            if sns, ok := realityCfg["serverNames"].([]interface{}); ok && len(sns) > 0 {
                if sn, ok := sns[0].(string); ok {
                    info.SNI = sn
                }
            }
        }
        info.IsTLS = true // Reality = всегда TLS
        return info
    }

    // Обычный TLS — SNI из сертификата
    if inbound.TLSEnabled && inbound.TLSCertID != nil {
        var cert models.Certificate
        if s.db.First(&cert, *inbound.TLSCertID).Error == nil {
            info.SNI = cert.Domain
        }
    }

    return info
}
```

#### Шаг 2: V2Ray link generators — 6 протоколов

**VLESS** (поддерживает и TLS и Reality):
```diff
  if inbound.TLSEnabled {
      params.Set("security", "tls")
  }
  if inbound.RealityEnabled {
      params.Set("security", "reality")
  }
+ tlsInfo := s.getInboundTLSInfo(inbound)
+ if tlsInfo.SNI != "" {
+     params.Set("sni", tlsInfo.SNI)
+ }
```

**VMess** (только TLS, не Reality):
```diff
  if inbound.TLSEnabled {
      vmessConfig["tls"] = "tls"
+     tlsInfo := s.getInboundTLSInfo(inbound)
+     if tlsInfo.SNI != "" {
+         vmessConfig["sni"] = tlsInfo.SNI
+     }
  }
```

**Trojan** (аналогично VLESS):
```diff
+ tlsInfo := s.getInboundTLSInfo(inbound)
+ if tlsInfo.SNI != "" {
+     params.Set("sni", tlsInfo.SNI)
+ }
```

**Hysteria2** (всегда TLS):
```diff
- return fmt.Sprintf("hysteria2://%s@%s:%d?insecure=1#%s", ...)
+ params := url.Values{}
+ params.Set("insecure", "1")
+ tlsInfo := s.getInboundTLSInfo(inbound)
+ if tlsInfo.SNI != "" {
+     params.Set("sni", tlsInfo.SNI)
+ }
+ return fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s", user.UUID, server, inbound.Port,
+     params.Encode(), url.PathEscape(inbound.Name))
```

**TUIC v4/v5** (всегда TLS):
```diff
  if inbound.TLSEnabled {
      params.Set("allow_insecure", "1")
  }
+ tlsInfo := s.getInboundTLSInfo(inbound)
+ if tlsInfo.SNI != "" {
+     params.Set("sni", tlsInfo.SNI)
+ }
```

#### Шаг 3: Clash proxy generator

```diff
  case "trojan":
-     return fmt.Sprintf("...    sni: %s\n    ...", name, server, inbound.Port, user.UUID, server), name
+     sni := server
+     if tlsInfo := s.getInboundTLSInfo(inbound); tlsInfo.SNI != "" {
+         sni = tlsInfo.SNI
+     }
+     return fmt.Sprintf("...    sni: %s\n    ...", name, server, inbound.Port, user.UUID, sni), name
```

Для `vless`, `vmess` — добавить `servername` если TLS:
```diff
+     if tlsInfo := s.getInboundTLSInfo(inbound); tlsInfo.SNI != "" && inbound.TLSEnabled {
+         entry += fmt.Sprintf("    servername: %s\n", tlsInfo.SNI)
+     }
```

Для `hysteria2`, `tuic` — аналогично добавить `sni`.

#### Шаг 4: Sing-box outbound generator

Одно изменение в месте создания `tlsConfig` (строка 719-722):
```diff
  tlsConfig := map[string]interface{}{
      "enabled":  true,
      "insecure": true,
  }
+ if tlsInfo := s.getInboundTLSInfo(inbound); tlsInfo.SNI != "" {
+     tlsConfig["server_name"] = tlsInfo.SNI
+ }
```

> **Общая сложность**: ~30-40 мин. Один helper + правки в 3 генераторах.

---

## 🟡 Проблема 11.X: Sub listener dual-mode (без изменений)

Архитектура Smart Dual-Mode Sub Listener — отдельный Fiber на `0.0.0.0:SUB_PORT` с автоматическим TLS из БД сертификатов. Не требует Caddy/Nginx.

> **Сложность**: ~1-1.5 ч.

---

## 🟡 Проблемы 11b-11e `/sub/` (без изменений)

| # | Проблема | Решение | Сложность |
|---|---------|---------|-----------|
| 11b | Username в Content-Disposition | `filename=subscription.txt` | ~2 мин |
| 11c | Нет валидации формата токена | Проверка длины и формата → 404 | ~5 мин |
| 11d | Open redirect в /s/ | `HasPrefix(fullURL, "/sub/")` | ~2 мин |
| 11e | Cache не инвалидируется | `InvalidateUserCache()` при деактивации | ~10 мин |

---

## 🟡 Проблемы 5-8 (без изменений)

| # | Проблема | Сложность |
|---|---------|-----------|
| 5 | Goroutine leak в rate limiter | ~30 мин |
| 6 | Нет cleanup expired tokens | ~20 мин |
| 7 | `your-org` в install.sh | 2 строки |
| 8 | JWT TTL 30 дней | 2 числа |

---

## 🔴 Проблема 12: Следы `vovk4morkovk4` (обновлено)

| Файл | Строка | Действие |
|------|--------|----------|
| [LICENSE](file:///mnt/Games/syncthing-shared-folder/isolate-panel/LICENSE#L3) | `Copyright (c) 2026 vovk4morkovk4` | → `isolate-project` |
| [CHANGELOG.md](file:///mnt/Games/syncthing-shared-folder/isolate-panel/docs/CHANGELOG.md#L115) | `github.com/vovk4morkovk4/isolate-panel` | → `github.com/isolate-project/isolate-panel` |

> **Сложность**: 2 строки.

---

## 🟡 Проблема 13: Секреты и .env.example (обновлено)

### Для чего используется CLOUDFLARE_API_TOKEN

`CLOUDFLARE_API_TOKEN` — это API-токен Cloudflare, используемый для **DNS-01 ACME challenge** при автоматическом выпуске TLS-сертификатов Let's Encrypt.

```
Пользователь запрашивает сертификат для proxy.example.com
     │
     ▼
ACME Client (lego) → Cloudflare API → создаёт DNS TXT-запись
     │                                  _acme-challenge.proxy.example.com
     ▼
Let's Encrypt проверяет TXT-запись → выдаёт сертификат
     │
     ▼
lego → удаляет TXT-запись → сохраняет cert.pem + key.pem
```

**Не нужен** если: пользователь загружает сертификаты вручную (manual upload) или не использует ACME.

**Два варианта авторизации в Cloudflare:**
- `CLOUDFLARE_API_TOKEN` (рекомендуемый) — scoped API token с правами `Zone:DNS:Edit`
- `CLOUDFLARE_API_KEY` + `CLOUDFLARE_EMAIL` — глобальный ключ (legacy, менее безопасный)

### .env.example — анализ полноты

| Переменная | В .env.example? | Используется в коде? | Нужна для deploy? | Действие |
|-----------|:---:|:---:|:---:|---------|
| `JWT_SECRET` | ✅ | ✅ `config.yaml` | 🔴 Обязательна | ОК |
| `APP_ENV` | ✅ | ✅ | ✅ | ОК |
| `PORT` | ✅ | ✅ | ✅ | ОК |
| `TZ` | ✅ | ✅ | ✅ | ОК |
| `DATABASE_PATH` | ✅ | ✅ | ✅ | ОК |
| `LOG_LEVEL` | ✅ | ✅ | ✅ | ОК |
| `LOG_FORMAT` | ✅ | ✅ | ⚪ | ОК |
| `MONITORING_MODE` | ✅ | ✅ | ✅ | ОК |
| `CORES_XRAY_API_ADDR` | ✅ | ✅ | ⚪ | ОК |
| `CORES_SINGBOX_API_KEY` | ✅ | ✅ | ⚪ | ОК |
| `SECURITY_ARGON2_*` | ✅ | ✅ | ⚪ | ОК |
| `TELEGRAM_BOT_TOKEN` | ✅ | ✅ | ⚪ | ОК |
| `TELEGRAM_CHAT_ID` | ✅ | ✅ | ⚪ | ОК |
| `WEBHOOK_URL` | ✅ | ✅ | ⚪ | ОК |
| `WEBHOOK_SECRET` | ✅ | ✅ | ⚪ | ОК |
| `CLOUDFLARE_API_TOKEN` | ✅ | ✅ | ⚪ | ОК |
| `CLOUDFLARE_EMAIL` | ✅ | ✅ | ⚪ | ОК |
| **`ADMIN_USERNAME`** | ❌ | ✅ `entrypoint` | 🟡 Нужна | ⚠️ **Добавить** |
| **`ADMIN_PASSWORD`** | ❌ | ✅ `migrate/main.go` | 🟡 Нужна | ⚠️ **Добавить** |
| **`APP_PANEL_URL`** | ❌ | ✅ `config.app.panel_url` | ⚪ Опционально | 💡 **Добавить как опционально** (fallback для inbound'ов без сертификатов; если все inbounds с TLS cert — не нужен, домен определяется из cert.Domain) |
| **`CONFIG_PATH`** | ❌ | ✅ `main.go:43` | ⚪ Default OK | 💡 Добавить как опцию |
| **`CLOUDFLARE_API_KEY`** | ❌ | ✅ `providers.go:255` | ⚪ Альтернатива TOKEN | 💡 Добавить |

### Рекомендуемые дополнения в .env.example

```bash
# Admin Credentials (REQUIRED for first launch)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin
# ⚠️ CHANGE AFTER FIRST LOGIN!

# Panel URL (optional - fallback for subscription links without certificates)
# Not needed if all your inbounds have TLS certificates bound — domain
# is automatically extracted from cert.Domain
# APP_PANEL_URL=https://your-server-ip-or-domain

# Cloudflare DNS API (alternative to API Token — legacy, less secure)
# CLOUDFLARE_API_KEY=your-global-api-key
```

> **Сложность**: ~5 мин.

---

## 🟢 Проблемы без действий

| # | Почему ОК |
|---|-----------|
| 2 | CORS — SSH tunnel by design |
| 9 | In-memory rate limiter — single-instance |
| 10 | Нет TLS для панели — SSH tunnel |

---

## Сводная таблица

| # | Проблема | Уровень | Сложность |
|---|---------|---------|-----------|
| 1 | Password `json:"-"` | 🔴 | 1 строка |
| 3 | API ключи Sing-box + Mihomo | 🔴 | ~45 мин |
| 4 | Swagger в prod | 🔴 | ~5 мин |
| 11a | SERVER_IP + panelURL dead code | 🔴 | ~15 мин |
| 12 | `vovk4morkovk4` | 🔴 | 2 строки |
| 11f | SNI из сертификата/Reality | 🟡 | ~30-40 мин |
| 11.X | Sub listener dual-mode | 🟡 | ~1-1.5 ч |
| 11b | Username в headers | 🟡 | ~2 мин |
| 11c | Валидация токена | 🟡 | ~5 мин |
| 11d | Open redirect | 🟡 | ~2 мин |
| 11e | Cache invalidation | 🟡 | ~10 мин |
| 5 | Goroutine leak | 🟡 | ~30 мин |
| 6 | Cleanup tokens | 🟡 | ~20 мин |
| 7 | `your-org` в install.sh | 🟡 | 2 строки |
| 8 | JWT TTL | 🟡 | 2 числа |
| 13 | .env.example неполный | 🟡 | ~5 мин |

---

## Рекомендуемый порядок

### Фаза 1: Quick fixes (< 15 мин)
1. **#1** — `json:"-"`
2. **#12** — LICENSE + CHANGELOG
3. **#8** — JWT TTL
4. **#7** — `your-org` → `isolate-project`
5. **#11b** — Content-Disposition
6. **#11d** — Open redirect

### Фаза 2: Средние задачи (~1.5 часа)
7. **#4** — Swagger guard
8. **#11a** — SERVER_IP → panelURL/cert.Domain
9. **#11c** — Token validation
10. **#11e** — Cache invalidation
11. **#6** — DataRetention cleanup
12. **#11f** — SNI из сертификата/Reality
13. **#13** — .env.example дополнения

### Фаза 3: Основные задачи (~2 часа)
14. **#11.X** — Sub listener dual-mode
15. **#5** — Rate limiter goroutine leak
16. **#3** — API key autogeneration

**Общая оценка: ~4 часа.**
