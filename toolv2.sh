#!/bin/bash

DEST="./EvidenciasD"
mkdir -p "$DEST"

# usuarios actual
USER_NAME=$(logname)
USER_HOME="/home/$USER_NAME"

# === SSH config ===

if [[ -f /etc/ssh/sshd_config ]] then
    cp /etc/ssh/sshd_config "$DEST/"

else
    echo "[!] No se encontro /etc/ssh/sshd_config"
fi

# === crontab del usuario ===
if [[ -d /var/spool/cron ]] then
 cp -r /var/spool/cron "$DEST/cron/"

else
    echo "[!] No se encontró /var/spool/cron"
fi

if [[ -d /etc/crontab ]] then
 cp -r /var/spool/cron "$DEST/cron/"

else
    echo "[!] No se encontró /etc/crontab"
fi

# === interfaces de red ===
if [[ -f /etc/network/interfaces ]] then 
    cp /etc/network/interfaces "$DEST/"
else 
    echo "[!] No se encontro /etc/network/interfaces"
fi

if [[ -d /etc/sysconfig/network-scripts ]] then
 cp /etc/sysconfig/network-scripts/ifcfg-* "$DEST/" 2>/dev/null
else
  echo "[!] No se encontraron archivos ifcfg-*"
fi

# === reglas de iptables ===
iptables-save > "$DEST/iptables.rules"

# === lista de procesos ===
ps aux > "$DEST/process_list.txt"
pstree > "$DEST/process_list_tree.txt"

# === servicios ===
service --status-all > "$DEST/services_status.txt" 2>/dev/null || systemctl list-units --type=service > "$DEST/services_status.txt"

if command -v service &>/dev/null; then
  service --status-all > "$DEST/services_status.txt" 2>/dev/null
else
  systemctl list-units --type=service > "$DEST/services_status2.txt"
fi

# === últimos inicios de sesión ===
last > "$DEST/last_logins.txt"

# === hosts y hostname ===
[[ -f /etc/hosts ]] && cp /etc/hosts "$DEST/"
[[ -f /etc/hostname ]] && cp /etc/hostname "$DEST/"

# === logs de Apache (si aplica) ===
[[ -f /var/log/apache2/access.log ]] && cp /var/log/apache2/access.log "$DEST/"
[[ -f /var/log/apache2/error.log ]] && cp /var/log/apache2/error.log "$DEST/"

# === directorios temporales ===
[[ -d /tmp ]] && cp -r /tmp "$DEST/tmp" || echo "[!] No se encontró /tmp"
[[ -d /var/tmp ]] && cp -r /var/tmp "$DEST/var_tmp" || echo "[!] No se encontró /var/tmp]"
[[ -d /dev/shm ]] && cp -r /dev/shm "$DEST/dev_shm" || echo "[!] No se encontró /dev/shm"

# === búsqueda de archivos .babyk ===
find / -type f -name "*.babyk" 2>/dev/null > "$DEST/archivos_babyk.txt"

# === copiar carpetas ocultas del usuario ===
HIDDEN_DIRS=(".ssh" ".config" ".local" ".cache" ".gnupg")

for dir in "${HIDDEN_DIRS[@]}"; do
  if [[ -d "$USER_HOME/$dir" ]]; then
    cp -r "$USER_HOME/$dir" "$DEST/"
  fi
done

echo "[✓] Extracción completada. Archivos guardados en: $DEST"

