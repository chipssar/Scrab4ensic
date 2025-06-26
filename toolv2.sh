#!/bin/bash

DEST="./EvidenciasD"
mkdir -p "$DEST"


# === logs ====

### DE ESTO FALLAR PUES VOLCAMOS TODA LA CARPETADE LOGS ###

if [[ -d /var/log ]] then
 cp -r /var/log "$DEST/log"
 else
 echo "[!] No se encontro /var/log"
fi


# === bash history de todos los usuarios ===

for dir in /home/*; do 
  user=$(basename "$dir")
  hist="$dir/.bash_history"
  if [ -d "$hist" ]; then 
    echo "==== Historial de $user ====" >> "$DEST/historico_bash.txt"
    cat "$hist" >> "$DEST/historico_bash.txt"
    echo >> "$DEST/historico_bash.txt"
  fi
done

[ -d /root/.bash_history ] && echo "==== Historial de root ====" >> $DEST/historico_bashRoot.txt && cat /root/.bash_history >> $DEST/historico_bashRoot.txt

# === conexiones activas ===

#Procesos
 
lsof -i -n -P > "$DEST/PuertosxProcesos.txt"
 
# TCP y UDP activas
ss -tuln > "$DEST/tcp_udp.txt"


# usuarios actual
USER_NAME=$(logname)
USER_HOME="/home/$USER_NAME"

# === SSH config ===

if [[ -d /etc/ssh/sshd_config.d ]] then
    cp -r /etc/ssh/sshd_config.d "$DEST/sshd_config.d"

else
    echo "[!] No se encontro /etc/ssh/sshd_config.d"
fi

if [[ -d /etc/ssh/ssh_config.d ]] then
    cp -r /etc/ssh/ssh_config.d "$DEST/ssh_config.d"

else
    echo "[!] No se encontro /etc/ssh/ssh_config"
fi

if [[ -d /etc/ssh ]] then
    cp -r /etc/ssh "$DEST/ssh"

else
    echo "[!] No se encontro /etc/ssh"
fi

# === crontab ===
if [[ -d /var/spool/cron ]] then
 cp -r /var/spool/cron "$DEST/cron/"

else
    echo "[!] No se encontró /var/spool/cron"
fi

if [[ -d /etc/crontab ]] then
 cp -r /etc/crontab "$DEST/crontab.txt"

else
    echo "[!] No se encontró /etc/crontab"
fi

# cron.hourly

if [[ -d /etc/cron.hourly ]] then
 cp -r /etc/cron.hourly "$DEST/cron.hourly"

else
    echo "[!] No se encontró /etc/cron.hourly"
fi

# cron.daily

if [[ -d /etc/cron.daily ]] then
 cp -r /etc/cron.daily "$DEST/cron.daily"

else
    echo "[!] No se encontró /etc/cron.daily"
fi

# cron.weekly

if [[ -d /etc/cron.weekly ]] then
 cp -r /etc/cron.weekly "$DEST/cron.weekly"

else
    echo "[!] No se encontró /etc/cron.weekly"
fi

# cron.monthly

if [[ -d /etc/cron.monthly ]] then
 cp -r /etc/cron.monthly "$DEST/cron.monthly"

else
    echo "[!] No se encontró /etc/cron.monthly"
fi

# === audit

if [[ -d /var/spool/audit ]] then
  cp -r /etc/spool/audit "$DEST/audit"

 else
    echo "[!] No se encontro /etc/spool/audit"
fi

# === interfaces de red ===
if [[ -d /etc/network/interfaces ]] then 
    cp -r /etc/network/interfaces "$DEST/interfaces"
else 
    echo "[!] No se encontro /etc/network/interfaces"
fi

if [[ -d /etc/sysconfig/network-scripts ]] then
 cp /etc/sysconfig/network-scripts/ifcfg-* "$DEST/ifcfg-" 
else
  echo "[!] No se encontraron archivos ifcfg-*"
fi

# netconfig

if [[ -d /etc/netconfig ]] then 
    cp /etc/netconfig "$DEST/netconfig"
else 
    echo "[!] No se encontro /etc/netconfig"
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
[[ -d /etc/hosts ]] && cp /etc/hosts "$DEST/hosts.txt"
[[ -d /etc/hostname ]] && cp /etc/hostname "$DEST/hostname.txt"

# === logs de Apache (si aplica) ===
[[ -d /var/log/apache2/access.log ]] && cp /var/log/apache2/access.log "$DEST/access.log" || echo "[!] No se encontró access.log"
[[ -d /var/log/apache2/error.log ]] && cp /var/log/apache2/error.log "$DEST/error.log" || echo "[!] No se encontró error.log"

# === directorios temporales ===
[[ -d /tmp ]] && cp -r /tmp "$DEST/tmp" || echo "[!] No se encontró /tmp"
[[ -d /var/tmp ]] && cp -r /var/tmp "$DEST/var_tmp" || echo "[!] No se encontró /var/tmp"
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

