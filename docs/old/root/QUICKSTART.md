# Isolate Panel — Quick Start 🚀

> Быстрый деплой панели на VPS. Время: ~5 минут.

## Требования

- VPS с Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)
- 1 CPU / 1 GB RAM минимум
- Docker 20.10+ и Docker Compose 2.0+

---

## Вариант 1: Автоматическая установка (рекомендуется)

```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/install.sh)
```

Скрипт автоматически:
- Установит Docker и Docker Compose (если не установлены)
- Создаст `/opt/isolate-panel/` со всеми файлами
- Сгенерирует безопасный JWT secret и пароль admin
- Запустит контейнер

После установки в терминале будут выведены **логин и пароль** — сохраните их.

**Обновление:**
```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/install.sh) --update
```

**Удаление:**
```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/install.sh) --uninstall
```

---

## Вариант 2: Ручная установка

### 1. Установите Docker (если не установлен)

```bash
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker
```

### 2. Создайте директорию и скачайте файлы

```bash
mkdir -p /opt/isolate-panel && cd /opt/isolate-panel

# Скачайте файлы
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/docker-compose.yml -o docker-compose.yml
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/.env.example -o .env
```

### 3. Настройте `.env`

```bash
nano .env
```

**Обязательные** параметры:

```env
# Сгенерируйте: openssl rand -base64 64
JWT_SECRET=ваш-длинный-секретный-ключ

# Пароль администратора (смените после первого входа!)
ADMIN_PASSWORD=ваш-надёжный-пароль
```

Сохраните и установите права:

```bash
chmod 600 .env
```

### 4. Запустите

```bash
docker compose up -d
```

Первый запуск занимает ~30 секунд (скачивание готового образа).

### 5. Проверьте статус

```bash
docker compose ps          # контейнер должен быть healthy
docker logs isolate-panel  # проверить логи запуска
```

---

## Доступ к панели

Панель слушает на `127.0.0.1:8080` и **не доступна** из интернета напрямую — это сделано намеренно для безопасности.

### С вашего компьютера через SSH туннель

```bash
ssh -L 8080:localhost:8080 root@ваш-сервер-ip
```

Затем откройте в браузере: **http://localhost:8080**

- **Логин:** `admin`
- **Пароль:** значение `ADMIN_PASSWORD` из `.env`

> 💡 **Совет:** Для фонового туннеля используйте `ssh -fNL 8080:localhost:8080 root@ip`

---

## Первая настройка

### 1. Смените пароль
Зайдите в **Settings** → **Change Password**

### 2. Запустите прокси-ядра
Перейдите в **Cores** → нажмите ▶️ **Start** для нужных ядер (Xray, Sing-box, Mihomo)

### 3. Создайте пользователя
**Users** → **Add User** → укажите имя, лимит трафика, срок действия

### 4. Создайте подключение (Inbound)
**Inbounds** → **Add Inbound** → выберите протокол (VLESS, VMess, Trojan, Hysteria2, …) → укажите порт

> ⚠️ **Порты для inbound'ов:** По умолчанию открыт диапазон `2000-2100` (TCP+UDP).
> Если нужны другие порты — отредактируйте `docker-compose.yml` и перезапустите:
> ```bash
> nano docker-compose.yml   # измените port range
> docker compose up -d      # применить
> ufw allow 2000:2100/tcp   # открыть в фаерволе
> ufw allow 2000:2100/udp
> ```

### 5. Привяжите пользователя к inbound
Откройте inbound → **Manage Users** → добавьте пользователя

### 6. Получите подписку
**Users** → нажмите на пользователя → **Subscription Links** → скопируйте ссылку

Вставьте ссылку в прокси-клиент: v2rayN, Clash, Sing-box, Streisand и др.

---

## Подписки (Subscriptions)

Подписочные ссылки доступны на отдельном порту (443 по умолчанию):
- `https://ваш-домен/sub/:token` — авто-определение формата
- `https://ваш-домен/sub/:token/clash` — Clash формат
- `https://ваш-домен/sub/:token/singbox` — Sing-box формат

Если привязан TLS-сертификат — подписки отдаются по HTTPS автоматически.

---

## TLS-сертификаты

Панель поддерживает автоматическое получение сертификатов через Let's Encrypt:

1. Перейдите в **Certificates** → **Add Certificate**
2. Укажите домен и метод верификации (HTTP-01 или DNS-01)
3. Для DNS-01 (Cloudflare) укажите API Token в `.env`:
   ```env
   CLOUDFLARE_API_KEY=ваш-api-key
   ```
4. Сертификат будет автоматически обновляться

---

## Управление

```bash
cd /opt/isolate-panel

docker compose ps            # статус контейнера
docker compose logs -f       # логи в реальном времени
docker compose restart       # перезапуск
docker compose stop          # остановка
docker compose start         # запуск
docker compose down          # полная остановка (данные сохранятся)
```

---

## Обновление

**Через install.sh (рекомендуется):**
```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/install.sh) --update
```

**Вручную:**
```bash
cd /opt/isolate-panel
docker compose pull && docker compose up -d
```

> ℹ️ Данные (БД, конфиги ядер, сертификаты) сохраняются в `./data/` и не теряются при обновлении.

---

## Файловая структура

```
/opt/isolate-panel/
├── docker-compose.yml       # Docker конфигурация
├── .env                     # Переменные окружения (секреты)
├── data/                    # Persistent данные (volume)
│   ├── isolate-panel.db     # Основная БД (SQLite)
│   ├── .core-secrets        # Авто-сгенерированные API ключи ядер
│   └── cores/
│       ├── xray/
│       │   ├── xray           # Бинарник
│       │   └── config.json    # Конфиг (управляется панелью)
│       ├── mihomo/
│       │   ├── mihomo
│       │   └── config.yaml
│       └── singbox/
│           ├── sing-box
│           └── config.json
└── logs/                    # Логи (volume)
    ├── supervisord.log
    └── isolate-panel/
```

---

## Troubleshooting

### Контейнер не запускается
```bash
docker logs isolate-panel        # посмотреть ошибки
docker compose ps                 # проверить статус
```

### Ядра не запускаются через панель
```bash
docker exec isolate-panel supervisorctl status
docker exec isolate-panel cat /var/log/supervisor/xray-stderr.log
```

### Порт занят
```bash
ss -tlnp | grep :443             # кто занимает порт
```

### Сброс пароля admin
```bash
docker exec isolate-panel sqlite3 /app/data/isolate-panel.db \
  "UPDATE admins SET password='' WHERE username='admin';"
docker compose restart
# Пароль сбросится на значение ADMIN_PASSWORD из .env
```

---

## Безопасность

| Мера | Описание |
|------|----------|
| 🔒 SSH-only доступ | Панель на localhost, только через SSH туннель |
| 🔑 JWT + Argon2id | Короткоживущие токены (15 мин access, 7 дней refresh) |
| 🛡️ TOTP 2FA | Двухфакторная аутентификация (опционально) |
| 🚫 Rate limiting | Защита от брутфорса (login: 5/мин, API: 60/мин) |
| 📋 Audit log | Логирование всех действий администратора |
| 🔐 API keys | Авто-генерация ключей для ядер при первом запуске |

---

**Готово!** 🎉 Панель установлена и работает.

📖 Подробная документация: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | [docs/API.md](docs/API.md)
