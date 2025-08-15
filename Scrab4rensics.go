package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// #######################################################################################
// Scrab4rensics - Recolección forense de evidencia en equipos Linux                     #
// Autor: Alina Almonte                                                                  #
// Descripción: Este script recopila información del sistema, servicios, procesos, red,  #
// autenticación, historial de comandos y archivos relevantes para análisis forense.     #
// #######################################################################################
// Notas:                                                                                #
//  - Ejecutar como root para maximizar acceso (lectura de /etc/shadow, /root, etc.)     #
//  - En sistemas sin systemd, algunas salidas vendrán vacías; se guardan igual.         #
//  - El ZIP queda en /Desktop/<hostname>_YYYYMMDDThhmmssZ.zip                           #
//  - Enjoy the forensic evidence gathering! jeje ;D                                     #
// #######################################################################################

// ------------------------------
// Utilidades de SO / Detección
// ------------------------------

type DistroInfo struct {
	ID      string   // ej. ubuntu, debian, fedora
	IDLike  []string // ej. debian, rhel
	Family  string   // Debian, RedHat, SUSE, Arch, Gentoo, Slackware, Especializadas, Desconocida
	Pretty  string
	Version string
}

func readOSRelease() (map[string]string, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	kv := make(map[string]string)
	re := regexp.MustCompile(`^([A-Z_]+)=(.*)$`)
	for s.Scan() {
		line := s.Text()
		m := re.FindStringSubmatch(line)
		if len(m) == 3 {
			v := strings.Trim(m[2], "\"'")
			kv[m[1]] = v
		}
	}
	return kv, s.Err()
}

func detectDistro() (*DistroInfo, error) {
	kv, err := readOSRelease()
	if err != nil {
		// Fallback mínimo
		return &DistroInfo{ID: "unknown", Family: "Desconocida", Pretty: "Unknown Linux"}, nil
	}
	id := strings.ToLower(kv["ID"])
	idLike := strings.Fields(strings.ToLower(strings.ReplaceAll(kv["ID_LIKE"], ",", " ")))
	pretty := kv["PRETTY_NAME"]
	version := kv["VERSION_ID"]

	family := classifyFamily(id, idLike)
	return &DistroInfo{ID: id, IDLike: idLike, Family: family, Pretty: pretty, Version: version}, nil
}
func classifyFamily(id string, like []string) string {
	in := func(s string, arr []string) bool {
		return slices.Contains(arr, s)
	}
	// Normalización por ID directo
	switch id {
	case "debian", "ubuntu", "linuxmint", "pop", "elementary", "kali", "raspbian", "raspberrypi":
		return "Debian"
	case "fedora", "centos", "centos-stream", "almalinux", "rocky", "rhel":
		return "RedHat"
	case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles":
		return "SUSE"
	case "arch", "manjaro", "endeavouros", "arcolinux":
		return "Arch"
	case "gentoo", "sabayon", "calculate":
		return "Gentoo"
	case "slackware", "slackel", "salix":
		return "Slackware"
	case "tails", "steamos", "alpine":
		return "Especializadas"
	}
	// Heurística por ID_LIKE
	if in("debian", like) {
		return "Debian"
	}
	if in("rhel", like) || in("fedora", like) || in("centos", like) {
		return "RedHat"
	}
	if in("suse", like) || in("opensuse", like) {
		return "SUSE"
	}
	if in("arch", like) {
		return "Arch"
	}
	if in("gentoo", like) {
		return "Gentoo"
	}
	if in("slackware", like) {
		return "Slackware"
	}
	return "Desconocida"
}

// ------------------------------
// Recolección: comandos y archivos
// ------------------------------

type CollectPlan struct {
	Cmds  []CmdSpec
	Paths []PathSpec
}

type CmdSpec struct {
	Name     string
	Args     []string
	Out      string // ruta de salida relativa dentro del caso
	Optional bool   // no es error si no existe
}

type PathSpec struct {
	Src  string
	Dest string // ruta relativa
}

func which(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func runAndSave(base string, c CmdSpec) error {
	if !which(c.Name) {
		if c.Optional {
			return nil
		}
		return fmt.Errorf("comando no encontrado: %s", c.Name)
	}
	out, err := exec.Command(c.Name, c.Args...).CombinedOutput()
	// Guardar aunque falle (permiso o retorno != 0)
	abs := filepath.Join(base, c.Out)
	if mkerr := os.MkdirAll(filepath.Dir(abs), 0755); mkerr != nil {
		return mkerr
	}
	werr := os.WriteFile(abs, out, 0644)
	if err != nil {
		return werr
	}
	return werr
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return nil
	} // ignorar si no existe
	if info.Mode()&os.ModeSymlink != 0 {
		// Resolver symlink de forma segura
		target, err := os.Readlink(src)
		if err != nil {
			return nil
		}
		return copyPath(target, dst)
	}
	if info.IsDir() {
		return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(src, path)
			dest := filepath.Join(dst, rel)
			if info.IsDir() {
				return os.MkdirAll(dest, 0755)
			}
			return copyFile(path, dest)
		})
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return nil
	} // ignorar si no hay permisos
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func zipFolder(source, target string) error {
	zf, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zf.Close()
	zw := zip.NewWriter(zf)
	defer zw.Close()
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name, _ = filepath.Rel(filepath.Dir(source), path)
		if info.IsDir() {
			hdr.Name += "/"
		} else {
			hdr.Method = zip.Deflate
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
}

// ------------------------------
// Planes por familia de distribución
// ------------------------------

func buildPlan(di *DistroInfo) *CollectPlan {
	// Comandos comunes
	cmds := []CmdSpec{
		{Name: "hostnamectl", Args: []string{"status"}, Out: "sistema/hostnamectl.txt", Optional: true},
		{Name: "uname", Args: []string{"-a"}, Out: "sistema/uname.txt"},
		{Name: "date", Args: []string{"-u"}, Out: "sistema/fecha_utc.txt"},
		// Servicios / procesos / sesiones
		{Name: "systemctl", Args: []string{"list-units", "--type=service", "--state=running"}, Out: "servicios/servicios_activos.txt", Optional: true},
		{Name: "service", Args: []string{"--status-all"}, Out: "servicios/service_status_all.txt", Optional: true},
		{Name: "ps", Args: []string{"aux"}, Out: "procesos/ps_aux.txt"},
		{Name: "last", Args: []string{"-a"}, Out: "sesiones/ultimos_inicios.txt", Optional: true},
		{Name: "who", Args: []string{"-a"}, Out: "sesiones/who_a.txt", Optional: true},
		// Red
		{Name: "ip", Args: []string{"a"}, Out: "red/ip_a.txt", Optional: true},
		{Name: "ss", Args: []string{"-tulpn"}, Out: "red/conexiones_tcp_udp.txt", Optional: true},
		{Name: "ss", Args: []string{"-tpn"}, Out: "red/conexiones_internet.txt", Optional: true},
		{Name: "netstat", Args: []string{"-tulpn"}, Out: "red/netstat_tulpn.txt", Optional: true},
		// Cron / tareas
		{Name: "crontab", Args: []string{"-l"}, Out: "tareas/cron_usuario_actual.txt", Optional: true},
		{Name: "atq", Args: []string{}, Out: "tareas/atq.txt", Optional: true},
		// Journal (si systemd)
		{Name: "journalctl", Args: []string{"-xe"}, Out: "logs/journalctl_xe.txt", Optional: true},
	}

	// Paths comunes
	paths := []PathSpec{
		{Src: "/etc/passwd", Dest: "sistema/etc/passwd"},
		{Src: "/etc/shadow", Dest: "sistema/etc/shadow"},
		{Src: "/etc/group", Dest: "sistema/etc/group"},
		{Src: "/etc/hosts", Dest: "sistema/etc/hosts"},
		{Src: "/etc/resolv.conf", Dest: "sistema/etc/resolv.conf"},
		// SSH
		{Src: "/etc/ssh", Dest: "ssh_config"},
		// Historiales bash
		{Src: "/root/.bash_history", Dest: "bash_history/root.bash_history"},
		{Src: "/home", Dest: "bash_history/home"},
		// Temporales
		{Src: "/tmp", Dest: "temporales/tmp"},
		{Src: "/var/tmp", Dest: "temporales/var_tmp"},
		// Sesiones binarias
		{Src: "/var/log/wtmp", Dest: "sesiones/wtmp"},
		{Src: "/var/log/lastlog", Dest: "sesiones/lastlog"},
	}

	// Específicos por familia para auth logs, apache y gestor de paquetes
	switch di.Family {
	case "Debian":
		paths = append(paths,
			PathSpec{"/var/log/auth.log", "logs/autenticacion/auth.log"},
			PathSpec{"/var/log/apache2", "apache"},
			PathSpec{"/var/log/dpkg.log", "paquetes/dpkg.log"},
			PathSpec{"/var/log/apt", "paquetes/apt"},
			PathSpec{"/etc/cron.d", "tareas/etc_cron.d"},
			PathSpec{"/etc/cron.daily", "tareas/cron.daily"},
			PathSpec{"/etc/cron.hourly", "tareas/cron.hourly"},
			PathSpec{"/etc/crontab", "tareas/etc_crontab"},
		)
	case "RedHat":
		paths = append(paths,
			PathSpec{"/var/log/secure", "logs/autenticacion/secure"},
			PathSpec{"/var/log/httpd", "apache"},
			PathSpec{"/var/log/yum.log", "paquetes/yum.log"},
			PathSpec{"/var/log/dnf.log", "paquetes/dnf.log"},
			PathSpec{"/etc/cron.d", "tareas/etc_cron.d"},
			PathSpec{"/etc/crontab", "tareas/etc_crontab"},
			PathSpec{"/var/spool/cron", "tareas/var_spool_cron"},
		)
	case "SUSE":
		paths = append(paths,
			PathSpec{"/var/log/audit/audit.log", "logs/audit/audit.log"},
			PathSpec{"/var/log/apache2", "apache"},
			PathSpec{"/var/log/zypp/history", "paquetes/zypp_history"},
			PathSpec{"/etc/cron.d", "tareas/etc_cron.d"},
			PathSpec{"/etc/crontab", "tareas/etc_crontab"},
		)
	case "Arch":
		paths = append(paths,
			PathSpec{"/var/log/auth.log", "logs/autenticacion/auth.log"}, // si está rsyslog
			PathSpec{"/var/log/journal", "logs/journal"},
			PathSpec{"/var/log/httpd", "apache"},
			PathSpec{"/var/log/pacman.log", "paquetes/pacman.log"},
			PathSpec{"/etc/cron.d", "tareas/etc_cron.d"},
			PathSpec{"/etc/crontab", "tareas/etc_crontab"},
		)
	case "Gentoo":
		paths = append(paths,
			PathSpec{"/var/log/auth.log", "logs/autenticacion/auth.log"},
			PathSpec{"/var/log/apache2", "apache"},
			PathSpec{"/var/log/emerge.log", "paquetes/emerge.log"},
			PathSpec{"/etc/cron.d", "tareas/etc_cron.d"},
			PathSpec{"/etc/crontab", "tareas/etc_crontab"},
		)
	case "Slackware":
		paths = append(paths,
			PathSpec{"/var/log/secure", "logs/autenticacion/secure"},
			PathSpec{"/var/log/httpd", "apache"},
			PathSpec{"/var/log/packages", "paquetes/packages"},
			PathSpec{"/etc/crontab", "tareas/etc_crontab"},
		)
	case "Especializadas":
		paths = append(paths,
			PathSpec{"/var/log/auth.log", "logs/autenticacion/auth.log"},
			PathSpec{"/var/log/apache2", "apache"},
			PathSpec{"/etc/crontab", "tareas/etc_crontab"},
		)
	default:
		paths = append(paths,
			PathSpec{"/var/log", "logs/var_log"},
		)
	}

	return &CollectPlan{Cmds: cmds, Paths: paths}
}

func main() {
	di, _ := detectDistro()
	hostname, _ := os.Hostname()
	caseName := hostname
	if caseName == "" {
		caseName = "equipo"
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	rootDir := filepath.Join("/tmp", fmt.Sprintf("%s_evidencia_%s", caseName, stamp))

	must(os.MkdirAll(rootDir, 0755))

	// Guardar metadatos de detección
	meta := fmt.Sprintf("ID=%s\nID_LIKE=%s\nFAMILY=%s\nPRETTY=%s\nVERSION_ID=%s\n",
		di.ID, strings.Join(di.IDLike, ","), di.Family, di.Pretty, di.Version)
	must(os.WriteFile(filepath.Join(rootDir, "_distro.txt"), []byte(meta), 0644))

	plan := buildPlan(di)

	// Ejecutar comandos
	for _, c := range plan.Cmds {
		if err := runAndSave(rootDir, c); err != nil {
			// registrar error pero continuar
			appendErr(rootDir, fmt.Errorf("cmd %s %v: %w", c.Name, strings.Join(c.Args, " "), err))
		}
	}

	// Copiar paths
	for _, p := range plan.Paths {
		dst := filepath.Join(rootDir, p.Dest)
		if err := copyPath(p.Src, dst); err != nil {
			appendErr(rootDir, fmt.Errorf("copy %s -> %s: %w", p.Src, dst, err))
		}
	}

	// Empaquetar en ZIP con nombre del equipo
	zipName := fmt.Sprintf("%s_%s.zip", caseName, stamp)
	zipPath := filepath.Join("/tmp", zipName)
	if err := zipFolder(rootDir, zipPath); err != nil {
		appendErr(rootDir, fmt.Errorf("zip: %w", err))
		fmt.Println("Error creando ZIP:", err)
		os.Exit(1)
	}
	fmt.Println("Distribución detectada:", di.Family, "(", di.Pretty, ")")
	fmt.Println("Evidencia forense guardada en:", zipPath)

	// Limpieza opcional del directorio de trabajo (comenta si quieres conservarlo)
	//_ = os.RemoveAll(rootDir)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func appendErr(base string, err error) {
	if err == nil {
		return
	}
	errFile := filepath.Join(base, "_errores.txt")
	f, ferr := os.OpenFile(errFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if ferr != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(time.Now().Format(time.RFC3339) + ": " + sanitizeErr(err) + "\n")
}

func sanitizeErr(err error) string {
	// Evitar exponer rutas internas del runtime si las hubiera
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "\n", " | ")
	return msg
}
