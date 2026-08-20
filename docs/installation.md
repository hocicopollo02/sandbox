# Instalación

## Requisitos

- Linux/Omarchy/Arch Linux.
- Podman rootless.
- Go 1.24+ solo si se instala desde código fuente o con `go install`.

Instala Podman en Arch:

```bash
sudo pacman -S podman
```

La CLI nunca ejecuta `sudo podman`.

## Desde una release de GitHub

El repositorio es privado. Necesitas GitHub autenticado para Git:

```bash
gh auth setup-git
```

Instala la release:

```bash
GOPRIVATE=github.com/hocicopollo02/* \\
  go install github.com/hocicopollo02/sandbox@v0.1.0
```

`go install` coloca el binario en `$GOBIN` o, si está vacío, en
`$(go env GOPATH)/bin`. Añade esa ubicación al `PATH`:

```bash
export PATH="$PATH:${GOBIN:-$(go env GOPATH)/bin}"
```

Comprueba la instalación:

```bash
sandbox version
sandbox doctor
```

Para instalar una versión más reciente:

```bash
GOPRIVATE=github.com/hocicopollo02/* \\
  go install github.com/hocicopollo02/sandbox@latest
```

## Desde el código fuente

```bash
git clone https://github.com/hocicopollo02/sandbox.git
cd sandbox
go build -o sandbox .
install -Dm755 sandbox ~/.local/bin/sandbox
```

Asegúrate de que `~/.local/bin` esté en el `PATH`.

## Troubleshooting

Si falta Podman:

```text
Podman is required but was not found.

sudo pacman -S podman
```

Ejecuta `sandbox doctor` para revisar Podman rootless, namespaces y
`/etc/subuid`/`/etc/subgid`.
