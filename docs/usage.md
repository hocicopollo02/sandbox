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
sandbox stop NAME
sandbox delete NAME [--yes] [--keep-home]
sandbox info NAME
sandbox doctor
sandbox version
```

Aliases: `ls`, `shell`, `rm`.

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
