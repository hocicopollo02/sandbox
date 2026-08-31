# Desarrollo

Requisitos: Go 1.24+ y Podman rootless para los E2E.

## Validación local

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

## Pre-push

Activa el hook versionado una vez por clon:

```bash
make hooks
```

El hook ejecuta `go test ./...` y `go vet ./...`. Los E2E no corren por
defecto porque requieren imágenes, red y varios minutos.

Para incluirlos:

```bash
SANDBOX_E2E=1 git push
```

## E2E

La suite usa un pseudo-terminal y contenedores Podman reales:

```bash
SANDBOX_E2E=1 go test -tags=integration ./...
```

Cubre doctor, wizard paso a paso, lifecycle persistent/disposable, cleanup tras
`Ctrl+C`, rechazo de shared home, colisiones, nombres inválidos, aislamiento
de rutas del host y preflight sin runtime.

## Build versionado

```bash
go build -ldflags "-X github.com/hocicopollo02/sandbox/cmd.Version=1.2.0 -X github.com/hocicopollo02/sandbox/cmd.Commit=$(git rev-parse --short HEAD) -X github.com/hocicopollo02/sandbox/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o sandbox .
```

## Release

1. Actualiza la versión en `cmd/root.go`.
2. Ejecuta la validación completa.
3. Publica el commit mediante no-mistakes.
4. Crea y publica un tag SemVer, por ejemplo `v1.2.0`.
5. Comprueba `go install github.com/hocicopollo02/sandbox@v1.2.0`.
