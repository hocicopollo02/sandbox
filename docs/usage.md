# Uso

## Inicio rápido

Crea un sandbox desechable con el wizard:

```bash
sandbox create
```

El wizard muestra una pregunta por pantalla: nombre, distribución,
persistencia, home y entrada automática.

### Sandbox persistente

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

### Sandbox desechable

```bash
sandbox create firstmate \
  --distro ubuntu \
  --disposable \
  --isolated-home
```

Al salir del shell se eliminan el contenedor y el home temporal.

## Comandos

```text
sandbox create [NAME] [flags]
sandbox list [--json]
sandbox enter [NAME]
sandbox exec NAME -- COMMAND [ARG...]
sandbox stop NAME [--json]
sandbox delete NAME [--yes] [--keep-home] [--if-exists] [--json]
sandbox info NAME [--json]
sandbox doctor [--json]
sandbox upgrade [--json]
sandbox version
```

El flag global `--error-format json` hace que los errores operativos se
emitan como un único objeto JSON en stderr; su contrato está documentado en la
[interfaz para agentes](agents.md#machine-error-codes).

Aliases: `ls`, `shell`, `rm`.

## `upgrade`

Actualiza la instalación de `sandbox` a la última versión estable publicada
como módulo de Go:

```bash
sandbox upgrade
```

El comando necesita Go 1.24+ y conexión a Internet. Cuando hay una actualización,
instala el reemplazo en el directorio configurado por `GOBIN` o, si está vacío,
en `$(go env GOPATH)/bin`. El binario que se está ejecutando debe estar en esa
misma ubicación; de lo contrario, el comando rechaza la actualización con una
indicación para mover el binario o configurar `GOBIN`. Si ya está en la última
versión, no reinstala nada:

```text
sandbox is already up to date (1.2.0)
```

Con `--json` devuelve un único objeto, pensado para agentes:

```json
{"name":"sandbox","current_version":"1.2.0","latest_version":"1.3.0","result":"upgraded"}
{"name":"sandbox","current_version":"1.3.0","latest_version":"1.3.0","result":"unchanged"}
```

## `exec`

Ejecuta un comando dentro de un sandbox **sin TTY**, pensado para
automatización, scripts y agentes de CI:

```bash
sandbox create review-123 \
  --distro ubuntu \
  --persistent \
  --isolated-home \
  --no-enter \
  --yes

sandbox exec review-123 -- git clone https://github.com/org/repo.git
sandbox exec review-123 -- bash -lc 'cd repo && go test ./...'
sandbox delete review-123 --yes
```

- Un sandbox persistente detenido se arranca automáticamente antes de ejecutar
  el comando, igual que `enter`.
- Los argumentos tras `--` llegan al proceso sin reinterpretarse con un shell
  ni con una TTY; stdout y stderr se conservan.
- El código de salida del comando se convierte en el código de salida de la
  CLI.
- Los errores de sandbox inexistente, no administrado o con estado desconocido
  son los mismos que en `enter`.

Fuera de alcance inicial: `--env`, `--workdir`, mounts adicionales, publicación
de puertos y un comando compuesto tipo `sandbox run`.

## `info`

Muestra los metadatos del sandbox y su estado en el runtime:

```bash
sandbox info gentle-ai
```

Con `--json` la salida es máquina-legible, pensada para agentes y scripts. El
contrato de claves estables está documentado en la [interfaz para agentes](agents.md):

```bash
sandbox info gentle-ai --json
```

```json
{
  "name": "gentle-ai",
  "distribution": "arch",
  "image": "docker.io/library/archlinux:latest",
  "persistence": "persistent",
  "home_mode": "isolated",
  "home_path": "/home/user/.local/share/sandbox/homes/gentle-ai",
  "created_at": "2026-08-30T12:00:00.000Z",
  "status": "running"
}
```

El valor de `status` es uno de `running`, `stopped`, `missing` o `unknown`.

## `create`

```text
--distro arch|ubuntu|fedora|debian
--persistent | --disposable
--isolated-home
--no-enter
--yes
--if-not-exists
--json
--verbose
```

`--shared-home` se conserva únicamente para devolver un error claro a scripts
antiguos. El home del host nunca se monta.

Para automatización, `create --json` exige nombre y las opciones explícitas
`--distro`, `--persistent`, `--isolated-home` y `--no-enter`. Devuelve un único
objeto, sin wizard ni mensajes de estado:

```json
{"name":"gentle-ai","result":"created"}
```

`stop NAME --json` usa `stopped` o `unchanged`. `delete NAME --yes --json` usa
`deleted` o `unchanged` y siempre incluye `retained_home`, con `null` salvo que
`--keep-home` conserve el home. Los valores `unchanged` corresponden a los
no-op de `--if-not-exists`, `--if-exists` o a detener un sandbox ya detenido.

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

La metadata no es la única fuente de verdad: el estado se contrasta con Podman.

## Eliminación

Por defecto `delete` elimina el contenedor, la metadata y el home aislado.
Usa `--keep-home` para conservar los datos del home. Usa `--if-exists` para
que la ausencia del sandbox sea un no-op exitoso.
