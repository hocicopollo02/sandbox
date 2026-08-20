# Seguridad y aislamiento

> El sandbox no es una frontera de seguridad fuerte ni sustituye una VM.
> No ejecutes malware o software no confiable en él.

## Modelo actual

- Podman se ejecuta como usuario normal en modo rootless.
- No se ejecuta `sudo podman`.
- No se monta `/` del host, `$HOME`, `/run/host`, `/usr/local`, sockets ni `.ssh`.
- Solo se monta el home gestionado por `sandbox` en `/home/sandbox`.
- El home gestionado se valida para rechazar symlinks y ancestros symlinked.
- La metadata y los homes se eliminan únicamente dentro de rutas gestionadas.

Las capas de Podman y sus metadatos viven en el storage local del host; eso es
necesario para que exista el contenedor y no equivale a instalar paquetes en el
filesystem del host.

## Límites

El modelo no protege frente a:

- exploits del kernel;
- software que necesite drivers, systemd, kernel o red del host;
- un proceso malicioso del mismo usuario que cambie rutas mientras se ejecuta
  una operación path-based.

Para esos casos usa una VM. Un volumen Podman administrado es una posible
mejora si se necesita reducir todavía más la superficie de rutas del host.

## Comprobación manual

Dentro de un sandbox, una escritura en una ruta del sistema del contenedor no
debe aparecer en el host:

```bash
sandbox enter <name>
touch /usr/local/bin/sandbox-proof
exit
test ! -e /usr/local/bin/sandbox-proof
```

La suite E2E automatiza esta comprobación.
