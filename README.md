# sandbox

CLI local para crear entornos Linux desechables o persistentes usando **Distrobox + Podman rootless**. Está pensada para Omarchy/Arch Linux: prueba CLIs, proyectos y dependencias sin llenar el host.

> **Limitación:** no es una frontera de seguridad fuerte ni un sustituto de una VM. No la uses para malware o software no confiable.

## Screenshots

Placeholder para screenshots/asciinema de `sandbox create`.

## Instalación

En Omarchy/Arch:

```bash
sudo pacman -S podman distrobox
git clone <repo>
cd sandbox
go build -o sandbox .
install -Dm755 sandbox ~/.local/bin/sandbox
```

Asegúrate de que `~/.local/bin` esté en `$PATH`.

Comprueba la instalación:

```bash
sandbox doctor
```

La herramienta no instala paquetes automáticamente y nunca ejecuta `sudo podman`.

## Quick start

```bash
sandbox create
```

El formulario permite elegir nombre, distribución, persistencia, home y entrada automática. Por defecto usa Arch Linux, un home aislado y un sandbox desechable.

Flujo no interactivo:

```bash
sandbox create gentle-ai \
  --distro arch \
  --persistent \
  --isolated-home

sandbox enter gentle-ai
sandbox list
sandbox stop gentle-ai
sandbox delete gentle-ai --yes
```

Para un entorno temporal:

```bash
sandbox create firstmate --distro ubuntu --disposable --isolated-home
```

Al salir del shell, se elimina el contenedor y el home temporal.

## Comandos

```text
sandbox create [NAME] [flags]
sandbox list [--json]
sandbox enter [NAME]
sandbox stop NAME
sandbox delete NAME [--yes] [--keep-home]
sandbox info NAME
sandbox doctor
sandbox version
```

Aliases: `ls`, `shell`, `rm`.

`create` admite:

```text
--distro arch|ubuntu|fedora|debian
--persistent | --disposable
--isolated-home | --shared-home
--no-enter
--yes
--verbose
```

`--shared-home` expone el home del host al contenedor y siempre requiere confirmación, salvo `--yes`.

## Datos y configuración

Metadata y homes aislados:

```text
~/.local/share/sandbox/
├── sandboxes/*.json
└── homes/<name>/
```

Configuración opcional en `~/.config/sandbox/config.toml`:

```toml
default_distro = "arch"
default_persistence = "disposable"
default_home = "isolated"
auto_enter = true
```

La metadata no se usa como única fuente de verdad: el estado se contrasta con Podman.

## Seguridad y aislamiento

- Ejecuta Podman como usuario normal/rootless.
- No monta `/` del host ni `.ssh` explícitamente.
- El home aislado es el comportamiento predeterminado.
- Distrobox comparte más integración con el host de la que tendría una VM.
- Para software no confiable, drivers, kernel, systemd o cambios de red del host, usa una máquina virtual.

## Troubleshooting

Si falta el runtime:

```text
Distrobox is required but was not found.

sudo pacman -S distrobox podman
```

Ejecuta `sandbox doctor` para revisar Podman, rootless, Distrobox y `/etc/subuid`/`/etc/subgid`. Usa `--verbose` para ver los comandos externos ejecutados.

Si una creación falla, la CLI intenta limpiar el home y el contenedor creados durante esa operación. Los homes aislados solo se borran dentro del directorio gestionado por `sandbox`.

## Desarrollo

Requisitos: Go 1.24+.

```bash
go test ./...
go vet ./...
go build ./...
go build -o sandbox .
```

La suite E2E opcional usa un pseudo-terminal y crea contenedores reales. Requiere Podman rootless, Distrobox, red y permiso para descargar la imagen. Se activa explícitamente para no ejecutar contenedores durante un test normal:

```bash
SANDBOX_E2E=1 go test -tags=integration ./...
```

Cubre `doctor`, ciclo persistent (incluyendo `stop`), ciclo disposable y cleanup tras `Ctrl+C`.

Las variables de build están preparadas para versionado:

```bash
go build -ldflags "-X github.com/pablo/sandbox/cmd.Version=0.1.0 -X github.com/pablo/sandbox/cmd.Commit=$(git rev-parse --short HEAD) -X github.com/pablo/sandbox/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o sandbox .
```
