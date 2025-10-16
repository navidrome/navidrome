FROM deluan/navidrome:latest

# Встановлюємо mpv та звукові бібліотеки
RUN apk add --no-cache mpv alsa-lib alsa-utils

# Додаємо користувача navidrome до групи audio (за потреби)
# Наприклад, GID звукової групи на хості — 29. Замініть якщо треба.
RUN addgroup navidrome audio || true

# Перевизначаємо точку входу (entrypoint) щоб передати аргументи, якщо потрібно
ENTRYPOINT ["/app/navidrome"]
CMD ["--configfile", "/data/navidrome.toml"]
