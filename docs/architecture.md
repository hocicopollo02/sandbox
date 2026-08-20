# Arquitectura

`sandbox` es una capa fina de UX sobre Podman rootless. No implementa un
runtime OCI propio.

```text
CLI (cmd/)
   │
   ▼
Manager (internal/sandbox/)
   ├── Store (internal/metadata/)
   └── Podman Client (internal/podman/)
          │
          ▼
       podman
```

## Responsabilidades

- `cmd/`: comandos Cobra, señales, entrada/salida y configuración.
- `internal/sandbox/`: reglas de negocio, lifecycle, validaciones y cleanup.
- `internal/metadata/`: records JSON, homes gestionados y protección de rutas.
- `internal/podman/`: adaptación mínima de la CLI `podman`.
- `internal/ui/`: prompts y salida de terminal.

## Creación

1. Se valida el nombre y las opciones.
2. Se comprueba Podman rootless.
3. Se verifica que no exista metadata ni un contenedor externo con el mismo nombre.
4. Se valida la raíz de homes y se crea el home aislado si corresponde.
5. Se crea y arranca el contenedor con `podman create` y `podman start`.
6. Los sandboxes persistentes guardan metadata JSON.
7. Los desechables entran al shell y limpian sus recursos al salir.

## Contenedor

El cliente crea un contenedor con:

- `HOME=/home/sandbox`;
- working directory `/home/sandbox`;
- hostname igual al nombre del sandbox;
- `sleep infinity` como proceso supervisor;
- entrada mediante `podman exec ... /bin/bash`.

Solo se monta el home gestionado en `/home/sandbox`.

## Persistencia

- **Persistent**: mantiene contenedor, metadata y home hasta `delete`.
- **Disposable**: elimina contenedor y home al salir del shell o recibir `Ctrl+C`.

La metadata sirve para descubrir sandboxes administrados, pero el estado de vida
se obtiene siempre del runtime.
