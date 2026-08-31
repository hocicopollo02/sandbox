# sandbox

CLI para crear entornos Linux desechables o persistentes usando **Podman
rootless directo**. Permite probar CLIs, proyectos y dependencias sin
instalarlas en el sistema host.

> No es una frontera de seguridad fuerte ni sustituye una VM. No la uses para
> malware o software no confiable.

## Inicio rápido

```bash
sandbox create
```

Flujo no interactivo:

```bash
sandbox create gentle-ai \
  --distro arch \
  --persistent \
  --isolated-home \
  --no-enter \
  --yes

sandbox enter gentle-ai
sandbox list
sandbox stop gentle-ai
sandbox delete gentle-ai --yes
```

Para ejecutar comandos sin TTY en un sandbox, consulta [`sandbox exec`](docs/usage.md#exec).

## Documentación

- [Instalación](docs/installation.md)
- [Uso y configuración](docs/usage.md)
- [Arquitectura](docs/architecture.md)
- [Seguridad y aislamiento](docs/security.md)
- [Desarrollo y releases](docs/development.md)
- [Interfaz para agentes](docs/agents.md)
- [AGENTS.md](AGENTS.md)
- [PRD](PRD.md)

## Garantía principal

El host nunca se monta dentro del sandbox. Solo se monta el home gestionado por
`sandbox` en `/home/sandbox`; Podman se ejecuta siempre como usuario normal y
rootless.
