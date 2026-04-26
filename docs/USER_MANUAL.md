# 📖 Isolate Panel — Руководство пользователя

> Полное руководство для системных администраторов по установке, настройке и эксплуатации Isolate Panel.

---

## 📋 Содержание

1. [Подготовка VPS и оптимизация](#1--подготовка-vps-и-оптимизация)
2. [Установка](#2--установка)
3. [Безопасный доступ к панели](#3--безопасный-доступ-к-панели)
4. [Обзор интерфейса и фич](#4--обзор-интерфейса-и-фич)
5. [Управление ядрами](#5--управление-ядрами)
6. [Управление пользователями](#6--управление-пользователями)
7. [Инбаунды и аутбаунды](#7--инбаунды-и-аутбаунды)
8. [Подписки](#8--подписки)
9. [Мониторинг и статистика](#9--мониторинг-и-статистика)
10. [Сертификаты](#10--сертификаты)
11. [Cloudflare WARP](#11--cloudflare-warp)
12. [GeoIP / GeoSite](#12--geoip--geosite)
13. [Бэкапы](#13--бэкапы)
14. [Уведомления](#14--уведомления)
15. [Настройки панели](#15--настройки-панели)
16. [CLI](#16--cli)
17. [Траблшутинг](#17--траблшутинг)

---

## 1. 🖥️ Подготовка VPS и оптимизация

### Минимальные требования

| Ресурс | Минимум | Рекомендуется |
|---|---|---|
| CPU | 1 ядро | 2 ядра |
| RAM | 1 GB | 2 GB |
| Диск | 10 GB | 20 GB SSD |
| ОС | Ubuntu 22.04+ / Debian 12+ / Alpine 3.21 | Ubuntu 24.04 LTS |

### Увеличение лимитов file descriptors

Прокси-ядра открывают множество соединений. По умолчанию лимит — 1024, этого недостаточно для продакшена.

```bash
# Проверить текущий лимит
ulimit -n

# Увеличить для текущей сессии
ulimit -n 65535

# Установить перманентно — добавить в /etc/security/limits.conf
echo "* soft nofile 65535" >> /etc/security/limits.conf
echo "* hard nofile 65535" >> /etc/security/limits.conf
echo "root soft nofile 65535" >> /etc/security/limits.conf
echo "root hard nofile 65535" >> /etc/security/limits.conf
```

Для Docker (добавить в `/etc/docker/daemon.json`):

```json
{
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 65535,
      "Soft": 65535
    }
  }
}
```

```bash
systemctl restart docker
```

### Настройка TCP BBR

BBR (Bottleneck Bandwidth and RTT) — алгоритм контроля перегрузки, значительно улучшающий пропускную способность на высоколатентных каналах. Критически важен для Hysteria2 и TUIC.

```bash
# Проверить текущий алгоритм
sysctl net.ipv4.tcp_congestion_control

# Включить BBR
echo "net.core.default_qdisc=fq" >> /etc/sysctl.conf
echo "net.ipv4.tcp_congestion_control=bbr" >> /etc/sysctl.conf
sysctl -p

# Проверить
sysctl net.ipv4.tcp_congestion_control
# Ожидаемый вывод: net.ipv4.tcp_congestion_control = bbr
```

### Оптимизация sysctl для высоконагруженных прокси

Создайте файл `/etc/sysctl.d/99-isolate-panel.conf`:

```ini
# === Сеть ===

# Максимальное количество соединений
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 65535

# Размер буферов TCP
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216

# Таймауты TCP
net.ipv4.tcp_fin_timeout = 15
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_max_syn_backlog = 65535

# Keepalive
net.ipv4.tcp_keepalive_time = 300
net.ipv4.tcp_keepalive_intvl = 30
net.ipv4.tcp_keepalive_probes = 5

# BBR congestion control
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr

# === Безопасность ===

# Защита от SYN flood
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_max_syn_backlog = 65535

# Не отправлять ICMP redirects
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0

# Не принимать ICMP redirects
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0

# Логирование странных пакетов
net.ipv4.conf.all.log_martians = 1

# === Файловая система ===

# Увеличить лимит inotify (для мониторинга логов)
fs.inotify.max_user_watches = 524288
fs.inotify.max_user_instances = 512
```

Применить:

```bash
sysctl -p /etc/sysctl.d/99-isolate-panel.conf
```

### Базовый Firewall (iptables)

```bash
# Политика по умолчанию
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Разрешить loopback
iptables -A INPUT -i lo -j ACCEPT

# Разрешить установленные соединения
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Разрешить SSH
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Разрешить порты прокси (пример: 443 TCP+UDP)
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p udp --dport 443 -j ACCEPT

# Разрешить диапазон портов инбаундов (2000-2050)
iptables -A INPUT -p tcp --match multiport --dports 2000:2050 -j ACCEPT
iptables -A INPUT -p udp --match multiport --dports 2000:2050 -j ACCEPT

# Сохранить правила
apt-get install -y iptables-persistent  # Debian/Ubuntu
iptables-save > /etc/iptables/rules.v4
```

> ⚠️ **Не открывайте порт 8080 в firewall!** Панель должна быть доступна только через SSH-туннель.

---

## 2. 📦 Установка

### Вариант A: Автоматическая установка (рекомендуется)

```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/install.sh)
```

Скрипт автоматически:
- Создаёт директорию `/opt/isolate-panel`
- Скачивает `docker-compose.yml` и `.env.example`
- Запускает `docker compose up -d`

### Вариант B: Ручная установка через Docker Compose

```bash
# 1. Создать рабочую директорию
mkdir -p /opt/isolate-panel && cd /opt/isolate-panel

# 2. Скачать docker-compose
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/docker-compose.yml -o docker-compose.yml

# 3. Скачать шаблон переменных окружения
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/.env.example -o .env

# 4. Настроить переменные
nano .env
```

### Обязательные переменные окружения

Отредактируйте `.env` — **обязательно** задайте:

| Переменная | Описание | Пример |
|---|---|---|
| `JWT_SECRET` | Секретный ключ для JWT | Генерируйте: `openssl rand -base64 64` |
| `ADMIN_PASSWORD` | Пароль администратора | Сильный пароль из 16+ символов |

```bash
# Генерация безопасного JWT-секрета
echo "JWT_SECRET=$(openssl rand -base64 64)" >> .env

# Установка пароля администратора
sed -i 's/ADMIN_PASSWORD=admin/ADMIN_PASSWORD=YourStrongPassword123!/' .env
```

### Опциональные переменные

| Переменная | По умолчанию | Описание |
|---|---|---|
| `APP_ENV` | `production` | Режим: `development` / `production` |
| `PORT` | `8080` | Порт панели (localhost only) |
| `TZ` | `UTC` | Таймзона: `Europe/Moscow`, `Asia/Yekaterinburg` и т.д. |
| `DATABASE_PATH` | `/app/data/isolate-panel.db` | Путь к SQLite внутри контейнера |
| `LOG_LEVEL` | `info` | Уровень: `debug`, `info`, `warn`, `error` |
| `MONITORING_MODE` | `lite` | Режим мониторинга: `lite` (60 сек) / `full` (10 сек) |
| `TELEGRAM_BOT_TOKEN` | — | Токен Telegram-бота для уведомлений |
| `TELEGRAM_CHAT_ID` | — | ID чата Telegram |
| `WEBHOOK_URL` | — | URL для webhook-уведомлений |
| `WEBHOOK_SECRET` | — | Секрет для HMAC-подписи webhook |
| `BACKUP_ENABLED` | `false` | Автоматические бэкапы |
| `BACKUP_SCHEDULE` | `0 2 * * *` | Cron-расписание бэкапов |
| `BACKUP_RETENTION` | `7` | Количество хранимых бэкапов |

### Запуск

```bash
docker compose up -d

# Проверить статус
docker compose ps

# Посмотреть логи
docker compose logs -f isolate-panel
```

### Обновление

```bash
cd /opt/isolate-panel
docker compose pull
docker compose up -d
```

> Данные (БД, логи, конфиги ядер) хранятся в Docker volumes и не теряются при обновлении.

### Удаление

```bash
cd /opt/isolate-panel
docker compose down
rm -rf /opt/isolate-panel
# Данные сохранены в ./data/ — удалите вручную при необходимости
```

---

## 3. 🔐 Безопасный доступ к панели

### Принцип

Isolate Panel **никогда не привязывается к публичному интерфейсу**. Панель слушает `127.0.0.1:8080` — это означает, что она доступна только локально на сервере. Это не баг, а **функция безопасности**.

Единственный способ добраться до веб-интерфейса — SSH-туннель.

### Подключение через SSH-туннель

#### Базовый туннель

```bash
ssh -L 8080:localhost:8080 user@your-server-ip
```

После выполнения этой команды на вашем локальном компьютере:
- Откройте браузер и перейдите на <http://localhost:8080>
- Трафик шифруется SSH — так же безопасно, как и сам SSH

#### Фоновый туннель

```bash
ssh -fNL 8080:localhost:8080 user@your-server-ip
```

- `-f` — уйти в фон после подключения
- `-N` — не выполнять удалённую команду (только туннель)

Закрыть фоновый туннель:

```bash
# Найти процесс
ps aux | grep "ssh -fNL"

# Убить
kill <PID>
```

#### Туннель через jump-хост

Если ваш сервер доступен только через промежуточный хост:

```bash
ssh -J jump-user@jump-host -L 8080:localhost:8080 user@target-server
```

#### Настройка SSH-алиаса

Упростите подключение, добавив в `~/.ssh/config`:

```
Host isolate
    HostName your-server-ip
    User root
    LocalForward 8080 localhost:8080
    ServerAliveInterval 60
    ServerAliveCountMax 3
```

Теперь достаточно:

```bash
ssh isolate
```

И открыть <http://localhost:8080>.

### Первый вход

1. Откройте <http://localhost:8080> в браузере
2. Введите логин: `admin`
3. Введите пароль: значение `ADMIN_PASSWORD` из `.env`
4. **Сразу смените пароль** после первого входа (Settings → Admin Password)

---

## 4. 🖱️ Обзор интерфейса и фич

### Страницы панели

| Страница | Описание |
|---|---|
| 📊 **Dashboard** | Реалтайм-статистика: пользователи, соединения, трафик, статус ядер |
| 👥 **Users** | CRUD пользователей, квоты, сроки, привязка к инбаундам |
| 📡 **Inbounds** | Управление входящими подключениями (протокол, порт, TLS) |
| 🔀 **Outbounds** | Управление исходящими подключениями (direct, block, proxy chains) |
| 🔄 **Cores** | Управление ядрами: старт/стоп/рестарт, логи, статус |
| 🔒 **Certificates** | ACME/ручные сертификаты, продление, отзыв |
| 💾 **Backups** | Создание/восстановление зашифрованных бэкапов |
| 🔔 **Notifications** | История уведомлений, настройка Telegram/Webhook |
| 🌍 **Geo Rules** | Правила маршрутизации по странам и категориям |
| ☁️ **WARP Routes** | Маршруты Cloudflare WARP, пресеты |
| ⚙️ **Settings** | Режим мониторинга, сброс трафика, общие настройки |
| 🔌 **Active Connections** | Текущие соединения с возможностью disconnect/kick |

### Навигация

- **Sidebar** — основная навигация между разделами
- **Тёмная/светлая тема** — переключается в верхней панели
- **Автовыход** — при истечении JWT access-токена (15 мин), автоматически обновляется через refresh-токен

---

## 5. 🔄 Управление ядрами

### Обзор трёх ядер

| Ядро | Специфика | API статистики |
|---|---|---|
| **Sing-box** | Универсальное ядро: все общие протоколы + Naive, AnyTLS, TUIC, Tor | Clash API (HTTP, порт 9090) |
| **Xray** | XHTTP, продвинутый VLESS с XTLS-Vision | gRPC StatsService (порт 10085) |
| **Mihomo** | ShadowsocksR, Snell, Mieru, Sudoku, TrustTunnel, MASQUE | Clash API (HTTP, порт 9091) |

### Как ядра работают в панели

Все три ядра запускаются как **дочерние процессы Supervisord** внутри одного Docker-контейнера:

```
Контейнер isolate-panel
├── supervisord (PID 1)
│   ├── isolate-server  (Go-бинарник, порт 8080)
│   ├── xray            (порт 10085 — gRPC API)
│   ├── sing-box        (порт 9090 — Clash API)
│   └── mihomo          (порт 9091 — Clash API)
```

По умолчанию в production ядра **не запускаются автоматически** (`autostart=false`). Они стартуют лениво — когда вы создаёте первый инбаунд для конкретного ядра.

### Запуск / остановка / перезапуск

Через UI (Cores → кнопки) или CLI:

```bash
# Запустить ядро
isolate-panel core start xray
isolate-panel core start singbox
isolate-panel core start mihomo

# Остановить
isolate-panel core stop xray

# Перезапустить
isolate-panel core restart singbox
```

### Логи ядер

Через UI (Cores → Logs) или напрямую:

```bash
# Логи Xray
docker exec isolate-panel cat /var/log/supervisor/xray-stdout*.log

# Логи Sing-box
docker exec isolate-panel cat /var/log/supervisor/singbox-stdout*.log

# Логи Mihomo
docker exec isolate-panel cat /var/log/supervisor/mihomo-stdout*.log

# Живой поток логов
docker exec isolate-panel tail -f /var/log/supervisor/xray-stdout*.log
```

### Конфиги ядер

Конфиги генерируются автоматически панелью на основе инбаундов/аутбаундов в БД:

| Ядро | Формат конфига | Путь внутри контейнера |
|---|---|---|
| Xray | JSON | `/app/data/cores/xray/config.json` |
| Sing-box | JSON | `/app/data/cores/singbox/config.json` |
| Mihomo | YAML | `/app/data/cores/mihomo/config.yaml` |

> ⚠️ **Не редактируйте конфиги вручную!** Панель перегенерирует их при любом изменении инбаунда/аутбаунда. Ручные правки будут затёрты.

---

## 6. 👥 Управление пользователями

### Создание пользователя

1. **Users → Create User**
2. Заполните:
   - **Username** — логин пользователя (уникальный)
   - **Email** — опционально
   - **Traffic Limit** — квота трафика (оставьте пустым для безлимита)
   - **Expiry Date** — срок действия (оставьте пустым для бессрочной)
3. UUID, пароль и subscription_token генерируются автоматически

### Квоты трафика

| Порог | Действие |
|---|---|
| **80%** | Предупреждающее уведомление (Telegram/Webhook) |
| **90%** | Критическое предупреждение |
| **100%** | Авто-блокировка — пользователь отключается, ядро перезагружается (graceful reload) |

### Сброс трафика

- **Ручной:** Users → кнопка «Reset Traffic»
- **Автоматический:** Настройки → Traffic Reset → Weekly / Monthly

### Привязка к инбаундам

Пользователь не может подключиться без привязки к инбаундам:

1. **Inbounds → выберите инбаунд → Users → Add Users**
2. Или **Users → выберите пользователя → Inbounds → Assign**

Пользователь автоматически получает все инбаунды, к которым привязан, в своей подписке.

---

## 7. 📡 Инбаунды и аутбаунды

### Создание инбаунда

1. **Inbounds → Create Inbound**
2. **Выберите ядро** → список протоколов фильтруется автоматически
3. **Выберите протокол** → форма адаптируется (Smart Forms)
4. **Настройте параметры:**
   - Port — уникальный порт для этого инбаунда
   - TLS — включить и выбрать сертификат (если требуется протоколом)
   - REALITY — альтернатива TLS (не требует сертификата)
   - Transport — WebSocket / gRPC / HTTP / HTTPUpgrade
5. **Assign Users** — привяжите пользователей

### Примеры типичных конфигураций

#### VLESS + REALITY + Vision (Xray)

```
Ядро: Xray
Протокол: VLESS
Порт: 443
TLS: REALITY
Flow: xtls-rprx-vision
```

Не требует сертификата — самый простой способ поднять VLESS.

#### VMess + WebSocket + TLS (Sing-box)

```
Ядро: Sing-box
Протокол: VMess
Порт: 8443
TLS: Включён (выбрать сертификат)
Transport: WebSocket
WS Path: /vmess
```

#### Hysteria2 (Sing-box)

```
Ядро: Sing-box
Протокол: Hysteria2
Порт: 443 UDP
TLS: Включён (обязательно для QUIC)
Upload/Download: 100/100 Mbps
```

### Аутбаунды

Аутбаунды определяют, куда идёт трафик от ядер:

| Тип | Назначение |
|---|---|
| **Direct** | Напрямую в интернет |
| **Block** | Заблокировать трафик |
| **DNS** | DNS-резолвинг |
| **Proxy** | Через другой прокси-сервер (VMess, VLESS, Trojan, SS, Hysteria2, TUIC) |
| **WARP** | Через Cloudflare WARP |

---

## 8. 🔗 Подписки

### Форматы подписок

Панель генерирует подписки в 4 форматах:

| Формат | Endpoint | Описание |
|---|---|---|
| **V2Ray** | `GET /sub/:token` | Base64-encoded список vmess://, vless:// ссылок |
| **Clash** | `GET /sub/:token/clash` | YAML-конфигурация для Clash/Mihomo-клиентов |
| **Sing-box** | `GET /sub/:token/singbox` | JSON-конфигурация для Sing-box-клиентов |
| **Isolate** | `GET /sub/:token/isolate` | Кастомный JSON с профилями |

### Авто-детект

`GET /sub/:token` автоматически определяет формат по `Accept`-заголовку:
- `text/yaml` → Clash
- `application/json` → Sing-box
- По умолчанию → V2Ray

### Короткие ссылки

Каждый пользователь получает короткую ссылку: `https://your-server/s/:code` (8-символьный код), которая редиректит на полную подписку.

### QR-код

`GET /sub/:token/qr` — генерирует QR-код с ссылкой подписки для быстрого сканирования мобильными клиентами.

### Безопасность подписок

- Token-based аутентификация (не требует JWT)
- Rate limiting: IP (30/ч), Token (10/ч), Global (1000/ч)
- Логирование каждого доступа: IP, User-Agent, страна, формат
- Детекция подозрительной активности (VPN/ISP-паттерны)

---

## 9. 📊 Мониторинг и статистика

### Режимы мониторинга

| Режим | Интервал | RAM | Точность | Для кого |
|---|---|---|---|---|
| **Lite** | 60 сек | ~30 MB | ±1 минута | Слабые VPS (1 GB RAM) |
| **Full** | 10 сек | ~100 MB | ±10 секунд | Мощные серверы (2+ GB RAM) |

Переключение: **Settings → Monitoring Mode**

### Dashboard

Реалтайм-дашборд показывает:

- 📊 Количество пользователей (всего / активных)
- 🔌 Активные соединения
- 📈 Трафик (upload / download)
- 🔄 Статус ядер (running / stopped)
- 📉 7-дневный график трафика
- 🏆 Top-пользователи по трафику

Данные обновляются через **WebSocket** каждые 5 секунд. При обрыве WS автоматически переключается на HTTP-polling.

### Активные соединения

**Active Connections** — список текущих соединений с возможностью:

- Фильтрация по пользователю / инбаунду
- Disconnect — закрыть конкретное соединение
- Kick — отключить все соединения пользователя

### Агрегация трафика

Сырые данные агрегируются автоматически:

| Гранулярность | Хранение |
|---|---|
| Raw | По умолчанию 90 дней (настраивается) |
| Hourly | Бессрочно |
| Daily | Бессрочно |

---

## 10. 🔒 Сертификаты

### Получение сертификата через ACME

1. **Certificates → Request Certificate**
2. Укажите:
   - **Domain** — ваш домен (например, `proxy.example.com`)
   - **Email** — для Let's Encrypt
   - **Challenge type** — DNS-01 (Cloudflare) или HTTP-01
3. При выборе DNS-01 → укажите Cloudflare API Token

### Предварительные требования для ACME

**DNS-01 (Cloudflare):**
- Домен должен управляться через Cloudflare
- Нужен Cloudflare API Token с правами `Zone:DNS:Edit`

**HTTP-01:**
- Домен должен резолвиться на IP вашего сервера
- Порт 80 должен быть открыт в firewall

> Для прокси-серверов **рекомендуется DNS-01**, т.к. не требует открытия порта 80.

### Ручная загрузка сертификата

1. **Certificates → Upload**
2. Загрузите `fullchain.pem` и `privkey.pem`

### Продление

- **Автопродление** включено по умолчанию для ACME-сертификатов
- Панель проверяет срок и продлевает за 30 дней до истечения
- При продлении отправляется уведомление (Telegram/Webhook)

### Привязка к инбаунду

При создании TLS-инбаунда выберите сертификат из списка. Панель автоматически подставит пути к сертификату в конфиг ядра.

---

## 11. ☁️ Cloudflare WARP

### Что такое WARP

Cloudflare WARP — это WireGuard-туннель через инфраструктуру Cloudflare. Позволяет маршрутизировать определённый трафик (например, к ChatGPT, Netflix) через WARP, обходя блокировки.

### Включение WARP

1. **WARP Routes → Register WARP**
2. Панель автоматически зарегистрирует WARP-аккаунт через wgcf
3. WARP-токены обновляются каждые 24 часа автоматически

### Управление маршрутами

Создайте правила, какой трафик идёт через WARP:

| Тип ресурса | Пример |
|---|---|
| Domain | `chat.openai.com` |
| Domain | `claude.ai` |
| IP | `1.2.3.4` |
| CIDR | `10.0.0.0/8` |

### Пресеты

Один клик для добавления групп доменов:

| Пресет | Содержит |
|---|---|
| 🎮 **Gaming** | Steam, Epic, Xbox, PlayStation домены |
| 📺 **Streaming** | Netflix, Hulu, Disney+, HBO домены |
| 🤖 **AI Services** | ChatGPT, Claude, Gemini, Midjourney домены |
| 📱 **Social** | Instagram, TikTok, Twitter/X домены |

---

## 12. 🌍 GeoIP / GeoSite

### Автообновление

Базы GeoIP и GeoSite обновляются автоматически каждые 7 дней.

### Правила маршрутизации

Создайте правила на основе географии:

| Тип правила | Пример | Действие |
|---|---|---|
| Страна | `CN` (China) | Block / Direct / Proxy |
| Категория | `ads` | Block |
| Категория | `social` | Route через WARP |

Правила автоматически включаются в генерацию конфигов ядер.

---

## 13. 💾 Бэкапы

### Создание бэкапа

1. **Backups → Create Backup**
2. Выберите компоненты:
   - ✅ Database (SQLite)
   - ✅ Core configs
   - ✅ Certificates
   - ✅ WARP data
   - ✅ GeoIP/GeoSite databases
3. Бэкап шифруется **AES-256-GCM** (потоковое шифрование — не нагружает RAM)

### Автоматические бэкапы

Настройка через `.env`:

```ini
BACKUP_ENABLED=true
BACKUP_SCHEDULE=0 2 * * *    # каждый день в 02:00
BACKUP_RETENTION=7            # хранить 7 последних бэкапов
```

### Восстановление

1. **Backups → выберите бэкап → Restore**
2. Панель дешифрует и восстановит выбранные компоненты
3. **Внимание:** восстановление перезапишет текущие данные

---

## 14. 🔔 Уведомления

### Telegram-бот

1. Создайте бота через [@BotFather](https://t.me/BotFather)
2. Получите `BOT_TOKEN`
3. Узнайте свой `CHAT_ID` (напишите боту, затем `https://api.telegram.org/bot<TOKEN>/getUpdates`)
4. Укажите в `.env`:

```ini
TELEGRAM_BOT_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
TELEGRAM_CHAT_ID=123456789
```

### Webhook

```ini
WEBHOOK_URL=https://your-webhook.example.com/notify
WEBHOOK_SECRET=your-hmac-secret
```

Каждый webhook-запрос подписан HMAC-SHA256 для верификации.

### Настраиваемые триггеры

| Событие | Описание |
|---|---|
| `quota_exceeded` | Пользователь превысил квоту трафика |
| `expiry_warning` | Подписка истекает через 7/3/1 дней |
| `cert_renewed` | Сертификат успешно продлён |
| `core_error` | Ядро упало или выдало ошибку |
| `failed_login` | Неудачная попытка входа в панель |
| `user_created` | Создан новый пользователь |
| `user_deleted` | Удалён пользователь |

---

## 15. ⚙️ Настройки панели

### Доступные настройки (Settings)

| Настройка | Описание | Значения |
|---|---|---|
| **Monitoring Mode** | Режим сбора статистики | `lite` / `full` |
| **Traffic Reset** | Периодичность сброса трафика | `disabled` / `weekly` / `monthly` |
| **Backup Schedule** | Cron-расписание бэкапов | Cron expression |
| **Backup Retention** | Количество хранимых бэкапов | Целое число |
| **Data Retention** | Срок хранения raw-статистики | Дни (по умолчанию 90) |
| **Panel Name** | Название панели (в UI) | Строка |

---

## 16. 🖥️ CLI

Isolate Panel поставляется с CLI-инструментом для управления из терминала.

### Установка

```bash
# CLI уже включён в Docker-образ
docker exec -it isolate-panel isolate-panel --help

# Или собрать отдельно
cd cli && go build -o isolate-panel .
```

### Аутентификация CLI

```bash
# Войти (сохраняет токены)
isolate-panel login --url http://localhost:8080 --username admin --password

# Выйти
isolate-panel logout
```

### Основные команды

```bash
# Пользователи
isolate-panel user list
isolate-panel user create --username alice
isolate-panel user delete --id 1

# Ядра
isolate-panel core start xray
isolate-panel core stop singbox
isolate-panel core restart mihomo

# Бэкапы
isolate-panel backup create
isolate-panel backup list
isolate-panel backup download --id 1

# Сертификаты
isolate-panel cert list
```

---

## 17. 🔧 Траблшутинг

### Логи Docker

```bash
# Логи панели
docker compose logs -f isolate-panel

# Логи Supervisord (все процессы)
docker exec isolate-panel supervisorctl status

# Логи конкретного ядра
docker exec isolate-panel supervisorctl tail xray stdout
docker exec isolate-panel supervisorctl tail singbox stderr
docker exec isolate-panel supervisorctl tail mihomo stdout
```

### Логи приложения

```bash
# Логи панели (Zerolog JSON)
docker exec isolate-panel cat /var/log/isolate-panel/*.log

# Структурированный поиск
docker exec isolate-panel grep '"level":"error"' /var/log/isolate-panel/*.log
```

### Типичные проблемы

#### ❌ Ядро не запускается

```bash
# Проверить статус
docker exec isolate-panel supervisorctl status

# Посмотреть ошибку ядра
docker exec isolate-panel supervisorctl tail xray stderr

# Частые причины:
# - Неверный конфиг (проверьте /app/data/cores/xray/config.json)
# - Порт занят (проверьте: ss -tlnp | grep <port>)
# - Отсутствует бинарник (проверьте: ls -la /app/data/cores/xray/)
```

#### ❌ Панель недоступна через SSH-туннель

```bash
# 1. Проверить, что контейнер запущен
docker compose ps

# 2. Проверить, что панель слушает
docker exec isolate-panel ss -tlnp | grep 8080

# 3. Проверить healthcheck
docker exec isolate-panel /docker-healthcheck.sh

# 4. Проверить SSH-туннель
ssh -v -L 8080:localhost:8080 user@server-ip
```

#### ❌ Подписка не работает

```bash
# Проверить, что пользователь привязан к инбаундам
# UI → Users → выберите пользователя → Inbounds

# Проверить, что инбаунд включён
# UI → Inbounds → is_enabled = true

# Проверить, что ядро запущено
# UI → Cores → is_running = true

# Проверить access-логи подписки
docker exec isolate-panel sqlite3 /app/data/isolate-panel.db \
  "SELECT * FROM subscription_accesses ORDER BY accessed_at DESC LIMIT 10;"
```

#### ❌ Сертификат не выдаётся

```bash
# Проверить Cloudflare API Token
# - Права: Zone:DNS:Edit
# - Зона: правильный домен

# Проверить логи ACME
docker exec isolate-panel grep -i acme /var/log/isolate-panel/*.log

# Проверить, что DNS-запись создалась
dig _acme-challenge.your-domain.com TXT
```

#### ❌ Высокое потребление RAM

```bash
# Переключить в lite-режим (если full)
# Settings → Monitoring Mode → lite

# Проверить потребление
docker stats isolate-panel

# При WARP — проверить wgcf-токен
docker exec isolate-panel cat /app/data/warp/warp-account.json
```

#### ❌ Бэкап не создаётся

```bash
# Проверить, что бэкапы включены
docker exec isolate-panel env | grep BACKUP

# Ручной бэкап для теста
# UI → Backups → Create Backup

# Проверить место на диске
df -h /opt/isolate-panel/data
```

### Полная переустановка

```bash
# Остановить
docker compose down

# Сохранить данные
cp -r /opt/isolate-panel/data /root/isolate-backup

# Удалить всё
rm -rf /opt/isolate-panel

# Установить заново
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/install.sh)

# Восстановить данные
cp -r /root/isolate-backup/* /opt/isolate-panel/data/
docker compose restart
```

---

## 📞 Поддержка

- **Документация:** [docs/MASTER_PLAN.md](MASTER_PLAN.md), [docs/ARCHITECTURE.md](ARCHITECTURE.md)
- **Issues:** [GitHub Issues](https://github.com/isolate-project/isolate-panel/issues)
- **Security:** Сообщайте об уязвимостях через GitHub Security Advisories
