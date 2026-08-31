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
sandbox stop NAME
sandbox delete NAME [--yes] [--keep-home]
sandbox info NAME [--json]
sandbox doctor [--json]
sandbox version
```

Aliases: `ls`, `shell`, `rm`.

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
--verbose
```

`--shared-home` se conserva únicamente para devolver un error claro a scripts
antiguos. El home del host nunca se monta.

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
Usa `--keep-home` para conservar los datos del home.
