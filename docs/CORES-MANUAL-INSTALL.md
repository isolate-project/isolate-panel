# Ручная установка ядер (Cores)

Если автоматическая загрузка ядер не работает (из-за ограничений сети, блокировок GitHub и т.д.), выполните эти шаги:

## Способ 1: Скачать ядра на хост и скопировать в контейнер

### 1. Скачайте ядра на хост

```bash
cd /mnt/Games/syncthing-shared-folder/isolate-panel/data/cores

# Xray
mkdir -p xray
cd xray
wget https://github.com/XTLS/Xray-core/releases/download/v26.3.27/Xray-linux-64.zip
unzip Xray-linux-64.zip
rm Xray-linux-64.zip
chmod +x xray
cd ..

# Mihomo
mkdir -p mihomo
cd mihomo
wget https://github.com/MetaCubeX/mihomo/releases/download/v1.19.23/mihomo-linux-amd64-v1.19.23.gz
gunzip mihomo-linux-amd64-v1.19.23.gz
mv mihomo-linux-amd64-v1.19.23 mihomo
chmod +x mihomo
cd ..

# Sing-box
mkdir -p singbox
cd singbox
wget https://github.com/SagerNet/sing-box/releases/download/v1.13.8/sing-box-1.13.8-linux-amd64.tar.gz
tar -xzf sing-box-1.13.8-linux-amd64.tar.gz
mv sing-box-1.13.8-linux-amd64/sing-box .
rm -rf sing-box-1.13.8-linux-amd64 sing-box-1.13.8-linux-amd64.tar.gz
chmod +x sing-box
cd ..
```

### 2. Проверьте наличие ядер

```bash
ls -la /mnt/Games/syncthing-shared-folder/isolate-panel/data/cores/*/
```

Ожидаемый вывод:
```
xray/xray
mihomo/mihomo
singbox/sing-box
```

### 3. Перезапустите контейнер

```bash
docker compose restart
```

Ядра будут автоматически обнаружены при следующем запуске!

---

## Способ 2: Скачать ядра внутри контейнера

### 1. Зайдите в контейнер

```bash
docker exec -it isolate-panel /bin/sh
```

### 2. Скачайте ядра вручную

```sh
# Xray
cd /app/data/cores/xray
wget https://github.com/XTLS/Xray-core/releases/download/v26.3.27/Xray-linux-64.zip
unzip Xray-linux-64.zip
rm Xray-linux-64.zip
chmod +x xray

# Mihomo
cd /app/data/cores/mihomo
wget https://github.com/MetaCubeX/mihomo/releases/download/v1.19.23/mihomo-linux-amd64-v1.19.23.gz
gunzip mihomo-linux-amd64-v1.19.23.gz
mv mihomo-linux-amd64-v1.19.23 mihomo
chmod +x mihomo

# Sing-box
cd /app/data/cores/singbox
wget https://github.com/SagerNet/sing-box/releases/download/v1.13.8/sing-box-1.13.8-linux-amd64.tar.gz
tar -xzf sing-box-1.13.8-linux-amd64.tar.gz -C /tmp
mv /tmp/sing-box-1.13.8-linux-amd64/sing-box .
rm -rf /tmp/sing-box-1.13.8-linux-amd64
chmod +x sing-box
```

### 3. Выйдите из контейнера

```sh
exit
```

### 4. Перезапустите панель

```bash
docker compose restart isolate-panel
```

---

## Способ 3: Использовать зеркала

Если GitHub заблокиан, используйте зеркала:

### Для Xray:
```bash
wget https://mirror.ghproxy.com/https://github.com/XTLS/Xray-core/releases/download/v26.3.27/Xray-linux-64.zip
```

### Для Mihomo:
```bash
wget https://mirror.ghproxy.com/https://github.com/MetaCubeX/mihomo/releases/download/v1.19.23/mihomo-linux-amd64-v1.19.23.gz
```

### Для Sing-box:
```bash
wget https://mirror.ghproxy.com/https://github.com/SagerNet/sing-box/releases/download/v1.13.8/sing-box-1.13.8-linux-amd64.tar.gz
```

---

## Проверка установки

После установки ядер проверьте их статус:

```bash
# Проверить наличие файлов
docker exec isolate-panel ls -lh /app/data/cores/*/xray /app/data/cores/*/mihomo /app/data/cores/*/sing-box

# Проверить логи загрузки
docker logs isolate-panel | grep -E "(✅|Downloading)"

# Запустить ядра через панель
# Откройте http://localhost:8080/cores и нажмите Start для каждого ядра
```

---

## Альтернативные источники

Если официальные релизы недоступны, попробуйте:

1. **Gitee (China mirror):**
   - https://gitee.com/mirrors/Xray-core
   - https://gitee.com/mirrors/mihomo

2. **IPFS:**
   - Xray: ipfs://Qm... (search on ipfs.io)
   
3. **Локальная сборка:**
   - Скомпилируйте ядро из исходников
   - Скопируйте бинарник в `/app/data/cores/{name}/`

---

## Примечания

- Ядра сохраняются в volume `./data/cores/` и **не удаляются** при перезапуске контейнера
- При обновлении образа Docker ядра остаются на месте
- Для обновления версии ядра - просто скачайте новую версию поверх старой
- Минимальные требования к дисковому пространству: ~100MB на все три ядра
