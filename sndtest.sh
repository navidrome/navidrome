#!/usr/bin/env bash
set -e

# Користувач, під яким працює твій контейнер / служба
APP_USER="bug"   # змінити на потрібного користувача
MUSIC_DIR="/big/Muz"
DATA_DIR="/nvme1/navidrome"

echo "=== Перевірка групи audio ==="
if id -nG "$APP_USER" | grep -qw "audio"; then
  echo "OK: Користувач $APP_USER є в групі audio"
else
  echo "WARN: Користувач $APP_USER *не* є в групі audio"
fi

echo
echo "=== Перевірка пристроїв /dev/snd ==="
if [ -d /dev/snd ]; then
  echo "/dev/snd існує"
  for dev in /dev/snd/*; do
    perms=$(stat -c "%A %U:%G" "$dev")
    echo " $dev → $perms"
  done
else
  echo "ERR: /dev/snd не існує"
fi

echo
echo "=== Перевірка прав на директорії музики та даних ==="
for dir in "$MUSIC_DIR" "$DATA_DIR"; do
  if [ -d "$dir" ]; then
    perms=$(stat -c "%A %U:%G" "$dir")
    echo "$dir → $perms"
    # Перевірка читання / запису для користувача
    if sudo -u "$APP_USER" test -r "$dir" && sudo -u "$APP_USER" test -w "$dir"; then
      echo "  OK: $APP_USER має читати й писати в $dir"
    else
      echo " WARN: $APP_USER не має повного доступу до $dir"
    fi
  else
    echo "ERR: Директорія $dir не знайдена"
  fi
done

echo
echo "=== Рекомендації ==="
echo "• Переконайся, що $APP_USER в групі audio або інша група, яка має доступ до /dev/snd"
echo "• Якщо права /dev/snd недостатні, можна зробити:
  sudo chmod a+rw /dev/snd/*"
echo "  або встановити ACL, щоб user мав доступ (setfacl)."
echo "• Переконайся, що директорії музики й даних мають власника або групу, що включає $APP_USER"
