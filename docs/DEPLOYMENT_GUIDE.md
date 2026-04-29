# Руководство по развёртыванию Isolate Panel на VPS

> Полное пошаговое руководство по установке, настройке и запуску Isolate Panel на виртуальном сервере (VPS).

---

## Содержание

1. [Требования](#требования)
2. [Быстрая установка (рекомендуется)](#быстрая-установка-рекомендуется)
3. [Ручная установка](#ручная-установка)
4. [Настройка окружения (.env)](#настройка-окружения-env)
5. [Настройка брандмауэра](#настройка-брандмауэра)
6. [Доступ к панели](#доступ-к-панели)
7. [Настройка подписок (публичный доступ)](#настройка-подписок-публичный-доступ)
8. [SSL/TLS сертификаты](#ssltls-сертификаты)
9. [Первоначальная настройка](#первоначальная-настройка)
10. [Обновление](#обновление)
11. [Резервное копирование](#резервное-копирование)
12. [Мониторинг и логи](#мониторинг-и-логи)
13. [Решение проблем](#решение-проблем)

---

## Требования

### Минимальные

| Компонент | Требование |
|-----------|-----------|
| **OS** | Ubuntu 20.04+, Debian 11+, CentOS 8+, Alpine Linux |
| **CPU** | 1 vCPU |
| **RAM** | 1 GB (контейнер ограничен 896MB) |
| **Disk** | 10 GB SSD |
| **Docker** | 20.10+ |
| **Docker Compose** | 2.0+ (плагин `docker compose`) |

### Рекомендуемые

| Компонент | Требование |
|-----------|-----------|
| **CPU** | 2 vCPU |
| **RAM** | 1 GB |
| **Disk** | 20 GB SSD |
| **Сеть** | Публичный IP, открытые порты 443/tcp, 443/udp + диапазон входящих портов |

### Примечание о безопасности

> **Панель администратора доступна ТОЛЬКО через SSH-туннель** — она не открывается на публичном IP. Это архитектурное решение безопасности, а не ограничение.
>
> Подписочные endpoints (для пользователей) работают на порту 443 и доступны из интернета.

---

## Быстрая установка (рекомендуется)

### Автоматический скрипт (одна команда)

```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/master/docker/install.sh)
```

Скрипт автоматически:
1. Проверит права root
2. Установит Docker (если не установлен)
3. Проверит Docker Compose
4. Создаст директории в `/opt/isolate-panel`
5. Скачает `docker-compose.yml` и `.env.example`
6. Сгенерирует JWT_SECRET и ADMIN_PASSWORD
7. Запустит контейнер
8. Выведет инструкции по доступу

**Сохраните выведенные учётные данные!** Они отображаются один раз.

### Обновление через скрипт

```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/master/docker/install.sh) --update
```

---

## Ручная установка

### Шаг 1: Установка Docker и Docker Compose

**Ubuntu / Debian:**
```bash
# Обновление системы
sudo apt update && sudo apt upgrade -y

# Установка зависимостей
sudo apt install -y apt-transport-https ca-certificates curl gnupg lsb-release

# Добавление репозитория Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Установка Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Запуск и автозагрузка
sudo systemctl enable docker
sudo systemctl start docker
```

**CentOS / RHEL / Rocky / Alma:**
```bash
# Установка Docker
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Запуск
sudo systemctl enable docker
sudo systemctl start docker
```

**Проверка:**
```bash
docker --version
docker compose version
```

### Шаг 2: Создание директорий

```bash
sudo mkdir -p /opt/isolate-panel/{data,logs}
cd /opt/isolate-panel
```

### Шаг 3: Скачивание конфигурации

```bash
# Docker Compose
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/master/docker/docker-compose.yml -o docker-compose.yml

# Пример окружения
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/master/docker/.env.example -o .env.example
```

### Шаг 4: Настройка .env файла

```bash
cp .env.example .env
nano .env
```

**Обязательные параметры:**

| Переменная | Описание | Пример |
|------------|----------|--------|
| `JWT_SECRET` | Секретный ключ для JWT (автогенерация при первом запуске, но рекомендуется задать явно) | `openssl rand -base64 64` |
| `ADMIN_PASSWORD` | Пароль администратора (обязательно задать! Если пусто — будет `admin`) | `MyStr0ngP@ssw0rd` |
| `APP_PANEL_URL` | IP или домен вашего сервера | `https://your-vps-ip` |

Генерация JWT_SECRET:
```bash
openssl rand -base64 64
```

**Пример минимального .env:**
```env
# === Обязательно: ADMIN_PASSWORD ===
# JWT_SECRET генерируется автоматически при первом запуске,
# но рекомендуется задать явно для воспроизводимости
JWT_SECRET=your-generated-secret-here-change-me

# ADMIN_PASSWORD ОБЯЗАТЕЛЕН! Если пусто — будет "admin" (смените сразу)
ADMIN_PASSWORD=your-strong-password-here

# === Приложение ===
APP_ENV=production
PORT=8080
TZ=Europe/Moscow
APP_PANEL_URL=https://your-server-ip-or-domain

# === База данных ===
DATABASE_PATH=/app/data/isolate-panel.db

# === Логирование ===
LOG_LEVEL=info

# === Мониторинг ===
MONITORING_MODE=lite

# === Администратор ===
ADMIN_USERNAME=admin
```

**Защита .env файла:**
```bash
chmod 600 .env
```

### Шаг 5: Запуск

```bash
cd /opt/isolate-panel
sudo docker compose pull
sudo docker compose up -d
```

Проверка статуса:
```bash
sudo docker compose ps
sudo docker logs isolate-panel
```

---

## Настройка окружения (.env)

### Полный список переменных

#### JWT аутентификация

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `JWT_SECRET` | — | **Рекомендуется**. Автогенерируется при первом запуске, но явное задание надёжнее |

#### Приложение

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `APP_ENV` | `production` | `production` или `development` |
| `PORT` | `8080` | Порт панели (внутри контейнера) |
| `TZ` | `UTC` | Часовой пояс (например, `Europe/Moscow`) |
| `APP_PANEL_URL` | — | URL сервера для подписок |

#### CORS

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `CORS_ORIGINS` | — | Разрешённые origins через запятую. В production оставьте пустым |

#### База данных

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `DATABASE_PATH` | `/app/data/isolate-panel.db` | Путь к SQLite внутри контейнера |

#### Логирование

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | — | `json` или `console` |
| `LOG_OUTPUT` | — | `stdout`, `file`, `both` |
| `APP_BODY_LIMIT` | `2048` | Лимит тела запроса в KB |

#### Мониторинг

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `MONITORING_MODE` | `lite` | `lite` (60с, ~30MB RAM) или `full` (10с, ~100MB RAM) |

#### Ядра (cores)

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `CORES_XRAY_API_ADDR` | `127.0.0.1:10085` | Xray gRPC API адрес |
| `CORES_SINGBOX_API_ADDR` | `127.0.0.1:9090` | Sing-box Clash API адрес |
| `CORES_MIHOMO_API_ADDR` | `127.0.0.1:9091` | Mihomo API адрес |
| `CORES_SINGBOX_API_KEY` | — | API ключ Sing-box |
| `CORES_MIHOMO_API_KEY` | — | API ключ Mihomo |

#### Безопасность Argon2id

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `SECURITY_ARGON2_TIME` | `3` | Итерации |
| `SECURITY_ARGON2_MEMORY` | `65536` | Память (KB) |
| `SECURITY_ARGON2_THREADS` | `4` | Потоки |
| `SECURITY_ARGON2_KEY_LENGTH` | `32` | Длина ключа |
| `SECURITY_ARGON2_SALT_LENGTH` | `16` | Длина соли |

#### Резервное копирование

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `BACKUP_ENABLED` | `false` | Включить автобэкапы |
| `BACKUP_SCHEDULE` | `0 2 * * *` | Расписание (cron) |
| `BACKUP_RETENTION` | `7` | Хранить копий |

#### Уведомления

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `TELEGRAM_BOT_TOKEN` | — | Токен Telegram бота |
| `TELEGRAM_CHAT_ID` | — | ID чата Telegram |
| `WEBHOOK_URL` | — | URL для webhook |
| `WEBHOOK_SECRET` | — | Секрет для подписи webhook |

#### Подписки

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `ISOLATE_SUBSCRIPTION_HOST` | `127.0.0.1` | Хост подписок. **Для публичного доступа установите `0.0.0.0`** (иначе подписки не будут доступны из интернета) |
| `ISOLATE_SUBSCRIPTION_ALLOW_HTTP` | `false` | Разрешить HTTP (не рекомендуется) |

#### Cloudflare DNS (для ACME)

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `CLOUDFLARE_API_TOKEN` | — | API токен Cloudflare |
| `CLOUDFLARE_API_KEY` | — | Глобальный API ключ |
| `CLOUDFLARE_EMAIL` | — | Email Cloudflare |

---

## Настройка брандмауэра

### UFW (Ubuntu / Debian)

```bash
# Установка (если не установлен)
sudo apt install ufw -y

# Базовые правила
sudo ufw default deny incoming
sudo ufw default allow outgoing

# SSH (не забудьте!)
sudo ufw allow 22/tcp

# Isolate Panel — входящие порты для прокси
sudo ufw allow 443/tcp   # HTTPS подписки + TLS inbounds
sudo ufw allow 443/udp   # QUIC (Hysteria2, TUIC)
sudo ufw allow 2000:2050/tcp  # Диапазон входящих портов
sudo ufw allow 2000:2050/udp  # Диапазон входящих портов (UDP)

# Дополнительные порты (если нужны)
# sudo ufw allow 8443/tcp
# sudo ufw allow 8443/udp

# Включение
sudo ufw enable
sudo ufw status verbose
```

### firewalld (CentOS / RHEL / Rocky)

```bash
# Запуск
sudo systemctl enable firewalld
sudo systemctl start firewalld

# SSH
sudo firewall-cmd --permanent --add-service=ssh

# Isolate Panel
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --permanent --add-port=443/udp
sudo firewall-cmd --permanent --add-port=2000-2050/tcp
sudo firewall-cmd --permanent --add-port=2000-2050/udp

# Перезагрузка
sudo firewall-cmd --reload
sudo firewall-cmd --list-all
```

### iptables

> **⚠️ ВНИМАНИЕ:** Если вы подключены к VPS через SSH, выполняйте команды **последовательно** в одной строке через `&&`, иначе рискуете потерять доступ при промежуточном сбое. Также убедитесь, что порт 22 (SSH) открыт **перед** изменением политики INPUT.

```bash
# Установка iptables-persistent для автосохранения правил (Debian/Ubuntu)
sudo apt install iptables-persistent -y

# --- Безопасный подход: добавляем правила без полной очистки ---
# Локальный трафик
sudo iptables -A INPUT -i lo -j ACCEPT

# Established connections
sudo iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# SSH (убедитесь, что это правило существует ДО смены политики!)
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Isolate Panel
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -p udp --dport 443 -j ACCEPT
sudo iptables -A INPUT -p tcp --match multiport --dports 2000:2050 -j ACCEPT
sudo iptables -A INPUT -p udp --match multiport --dports 2000:2050 -j ACCEPT

# Смена политики INPUT на DROP (только после добавления SSH!)
sudo iptables -P INPUT DROP
sudo iptables -P FORWARD DROP

# Сохранение (через iptables-persistent)
sudo netfilter-persistent save
```

**Если нужно полностью пересоздать правила (опасно!):**

```bash
# ⚠️ ВСЕГДА оставляйте открытым SSH ПЕРЕД выполнением!
# Выполняйте в одной строке:
sudo iptables -P INPUT ACCEPT && \
sudo iptables -F && \
sudo iptables -A INPUT -i lo -j ACCEPT && \
sudo iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT && \
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT && \
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT && \
sudo iptables -A INPUT -p udp --dport 443 -j ACCEPT && \
sudo iptables -A INPUT -p tcp -m multiport --dports 2000:2050 -j ACCEPT && \
sudo iptables -A INPUT -p udp -m multiport --dports 2000:2050 -j ACCEPT && \
sudo iptables -P INPUT DROP && \
sudo netfilter-persistent save
```

---

## Доступ к панели

### Через SSH-туннель (рекомендуется)

**Прямой туннель:**
```bash
ssh -L 8080:localhost:8080 root@your-vps-ip
```

**Фоновый туннель:**
```bash
ssh -fNL 8080:localhost:8080 root@your-vps-ip
```

**Через jump host:**
```bash
ssh -J jump-host -L 8080:localhost:8080 root@your-vps-ip
```

После установки туннеля откройте в браузере:
```
http://localhost:8080
```

**Учётные данные по умолчанию:**
- Логин: `admin`
- Пароль: значение `ADMIN_PASSWORD` из `.env`

### Постоянный туннель (autossh)

```bash
# Установка autossh
sudo apt install autossh -y

# Автозапуск через systemd
sudo tee /etc/systemd/system/isolate-tunnel.service > /dev/null << 'EOF'
[Unit]
Description=Isolate Panel SSH Tunnel
After=network.target

[Service]
Type=simple
User=your-local-user
ExecStart=/usr/bin/autossh -M 0 -NL 8080:localhost:8080 root@your-vps-ip -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o ExitOnForwardFailure=yes
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable isolate-tunnel
sudo systemctl start isolate-tunnel
```

---

## Настройка подписок (публичный доступ)

По умолчанию панель слушает только на `127.0.0.1:8080`. Подписочный listener работает на порту 443 и доступен из интернета.

> **⚠️ КРИТИЧНО:** По умолчанию `ISOLATE_SUBSCRIPTION_HOST=127.0.0.1`. Внутри Docker-контейнера это означает, что listener привязан к loopback и **не получит** трафик с хоста, даже если порт 443 проброшен. Чтобы подписки работали публично, добавьте в `.env`:
> ```env
> ISOLATE_SUBSCRIPTION_HOST=0.0.0.0
> ```
> После изменения перезапустите: `docker compose restart`

### Без SSL (не рекомендуется для production)

Если у вас нет домена и сертификата, подписки будут доступны по HTTP. Это небезопасно.

### С SSL (рекомендуется)

#### Вариант 1: Let's Encrypt (AutoTLS)

1. Укажите домен в настройках панели (Settings → Certificates)
2. Включите ACME / Let's Encrypt
3. Панель автоматически получит и обновит сертификат

#### Вариант 2: Ручная загрузка сертификата

1. Получите сертификат (Let's Encrypt, ZeroSSL, и т.д.)
2. Загрузите в панель: Certificates → Upload
3. Укажите домен и привяжите к inbound'ам

#### Вариант 3: Cloudflare (рекомендуется)

1. Настройте DNS-запись вашего домена на Cloudflare
2. Включите "Proxied" (оранжевое облако)
3. В `.env` укажите `CLOUDFLARE_API_TOKEN`
4. Панель автоматически создаст и обновит сертификаты через DNS-01 challenge

---

## SSL/TLS сертификаты

### Автоматическое получение (Let's Encrypt)

1. Перейдите в панель: **Certificates**
2. Нажмите **Request Certificate**
3. Введите домен (например, `panel.example.com`)
4. Выберите провайдера: **Let's Encrypt**
5. Укажите email для уведомлений
6. Нажмите **Request**

Сертификат будет автоматически обновляться.

### Ручная загрузка

1. **Certificates** → **Upload**
2. Загрузите:
   - Certificate (cert.pem / fullchain.pem)
   - Private Key (privkey.pem)
3. Укажите домен

### Проверка сертификата

```bash
# Проверка срока действия
echo | openssl s_client -servername your-domain.com -connect your-domain.com:443 2>/dev/null | openssl x509 -noout -dates

# Проверка цепочки
echo | openssl s_client -servername your-domain.com -connect your-domain.com:443 2>/dev/null | openssl x509 -noout -text
```

---

## Первоначальная настройка

### 1. Первый вход

1. Откройте панель через SSH-туннель
2. Войдите с учётными данными из `.env`
3. **Сразу смените пароль!** Settings → Security → Change Password

### 2. Настройка 2FA (рекомендуется)

1. **Settings** → **Security** → **Two-Factor Authentication**
2. Отсканируйте QR-код в приложении-аутентификаторе (Google Authenticator, Authy, и т.д.)
3. Введите код подтверждения
4. Сохраните backup codes

### 3. Запуск ядер (cores)

1. **Cores** в боковом меню
2. Выберите ядро (Xray, Sing-box, или Mihomo)
3. Нажмите **Start**

### 4. Создание inbound'ов

1. **Inbounds** → **Create**
2. Выберите протокол (VLESS, VMess, Trojan, и т.д.)
3. Настройте порт (в диапазоне 2000-2050, если используете стандартный docker-compose)
4. Настройте TLS / REALITY (опционально)
5. Сохраните

### 5. Создание пользователей

1. **Users** → **Create**
2. Укажите username, email (опционально)
3. Настройте лимит трафика (опционально)
4. Укажите срок действия (опционально)
5. Назначьте inbound'ы
6. Сохраните

### 6. Генерация подписок

1. Откройте пользователя
2. Нажмите **Subscription**
3. Скопируйте ссылку или QR-код
4. Импортируйте в клиент (v2rayN, Nekoray, Streisand, и т.д.)

---

## Обновление

### Через install.sh

```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/master/docker/install.sh) --update
```

### Вручную

```bash
cd /opt/isolate-panel

# Скачать новый docker-compose
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/master/docker/docker-compose.yml -o docker-compose.yml

# Обновить образ
sudo docker compose pull

# Перезапуск
sudo docker compose up -d

# Проверка
sudo docker compose ps
```

### Откат (если что-то сломалось)

```bash
cd /opt/isolate-panel

# Посмотреть доступные образы
sudo docker images ghcr.io/isolate-project/isolate-panel

# Откат к предыдущей версии
sudo docker compose down
sudo docker tag ghcr.io/isolate-project/isolate-panel:previous-version ghcr.io/isolate-project/isolate-panel:latest
sudo docker compose up -d
```

---

## Резервное копирование

### Автоматическое (внутри панели)

1. **Settings** → **Backup**
2. Включите **Auto Backup**
3. Настройте расписание (cron)
4. Укажите количество копий для хранения

### Ручное копирование данных

```bash
# Остановка контейнера
sudo docker compose down

# Копирование
cd /opt/isolate-panel
sudo tar -czf backup-$(date +%Y%m%d-%H%M%S).tar.gz data/ .env docker-compose.yml

# Запуск
sudo docker compose up -d
```

### Восстановление

```bash
# Остановка
sudo docker compose down

# Очистка данных (ОСТОРОЖНО!)
sudo rm -rf /opt/isolate-panel/data/*

# Распаковка бэкапа
cd /opt/isolate-panel
sudo tar -xzf backup-YYYYMMDD-HHMMSS.tar.gz

# Запуск
sudo docker compose up -d
```

---

## Мониторинг и логи

### Логи контейнера

```bash
# Все логи
sudo docker logs isolate-panel

# Последние 100 строк
sudo docker logs --tail 100 isolate-panel

# В реальном времени
sudo docker logs -f isolate-panel
```

### Логи внутри контейнера

```bash
# Панель
sudo docker exec isolate-panel cat /var/log/isolate-panel/panel.out.log

# Ошибки панели
sudo docker exec isolate-panel cat /var/log/isolate-panel/panel.err.log

# Xray
sudo docker exec isolate-panel cat /var/log/isolate-panel/xray.out.log

# Sing-box
sudo docker exec isolate-panel cat /var/log/isolate-panel/singbox.out.log

# Mihomo
sudo docker exec isolate-panel cat /var/log/isolate-panel/mihomo.out.log
```

### Мониторинг ресурсов

```bash
# Docker stats
sudo docker stats isolate-panel

# Процессы внутри контейнера
sudo docker exec isolate-panel ps aux

# Использование диска
sudo du -sh /opt/isolate-panel/data/
```

### Health check

```bash
# Проверка здоровья контейнера
sudo docker inspect --format='{{.State.Health.Status}}' isolate-panel

# Вручную
sudo docker exec isolate-panel /docker-healthcheck.sh
```

---

## Решение проблем

### Контейнер не запускается

```bash
# Проверить логи
sudo docker logs isolate-panel

# Проверить конфигурацию
sudo docker compose config

# Проверить права на .env
ls -la /opt/isolate-panel/.env
```

### Нет доступа к панели через SSH-туннель

```bash
# Проверить, слушает ли панель на 8080
sudo docker exec isolate-panel ss -tlnp | grep 8080
```

### Ядро (core) не запускается

```bash
# Проверить конфигурацию ядра
sudo docker exec isolate-panel cat /app/data/cores/xray/config.json
# Для форматированного вывода установите jq на хосте: apt install jq
# sudo docker exec isolate-panel cat /app/data/cores/xray/config.json | jq .
```

### Проблемы с портами

```bash
# Проверить, не заняты ли порты (на хосте)
sudo ss -tlnp | grep -E ':443|:200[0-9]'

# Или через fuser
sudo fuser 443/tcp 443/udp 2000/tcp 2050/tcp 2>/dev/null

# Проверить docker-compose.yml на конфликты портов
grep -A 20 "ports:" /opt/isolate-panel/docker-compose.yml
```

### База данных повреждена

```bash
# Остановка
sudo docker compose down

# Бэкап повреждённой базы
sudo mv /opt/isolate-panel/data/isolate-panel.db /opt/isolate-panel/data/isolate-panel.db.bak

# Пересоздание (данные будут потеряны!)
sudo docker compose up -d

# Восстановление из бэкапа (если есть)
sudo cp /path/to/backup/isolate-panel.db /opt/isolate-panel/data/
sudo docker compose restart
```

### Обновление не применяется

```bash
# Принудительная пересборка
sudo docker compose down
sudo docker compose pull
sudo docker compose up -d --force-recreate
```

---

## Безопасность

### Чек-лист безопасности

- [ ] Сменён пароль администратора по умолчанию
- [ ] Включена 2FA (двухфакторная аутентификация)
- [ ] JWT_SECRET сгенерирован случайно (не используется значение по умолчанию)
- [ ] Панель доступна только через SSH-туннель (не открыта в интернет)
- [ ] Подписки работают через HTTPS (порт 443)
- [ ] Брандмауэр настроен (открыты только нужные порты)
- [ ] Включено автоматическое обновление сертификатов (Let's Encrypt)
- [ ] Настроены уведомления о критических событиях (Telegram/webhook)
- [ ] Включено автоматическое резервное копирование
- [ ] .env файл имеет права 600 (только владелец)

### Ограничение доступа по IP (дополнительно)

Если вы хотите ограничить доступ к подпискам по IP:

```bash
# В .env
CORS_ORIGINS=https://your-domain.com
```

Или настройте на уровне обратного прокси (nginx/traefik).

---

## Полезные команды

```bash
# Перезапуск панели
cd /opt/isolate-panel && sudo docker compose restart

# Полный перезапуск (с очисткой)
cd /opt/isolate-panel && sudo docker compose down && sudo docker compose up -d

# Выполнить команду внутри контейнера
sudo docker exec -it isolate-panel sh

# Копировать файл из контейнера
sudo docker cp isolate-panel:/app/data/isolate-panel.db ./backup.db

# Копировать файл в контейнер
sudo docker cp ./backup.db isolate-panel:/app/data/isolate-panel.db

# Очистка старых образов
sudo docker image prune -a

# Очистка логов
sudo truncate -s 0 /opt/isolate-panel/logs/*.log
```

---

## Ссылки

- [Оригинальный репозиторий](https://github.com/isolate-project/isolate-panel)
- [Docker образы](https://github.com/isolate-project/isolate-panel/pkgs/container/isolate-panel)
- [Пользовательское руководство](docs/USER_MANUAL.md) (RU)
- [API документация](docs/API.md)
- [CLI справочник](docs/CLI.md)
