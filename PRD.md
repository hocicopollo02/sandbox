# PRD — Sandbox CLI para Omarchy

> **Revisión vigente (2026-08-20): aislamiento del host**
>
> La implementación V1 usa **Podman rootless directamente**, no Distrobox. El contenedor no monta el `$HOME` del host, `/run/host`, `/usr/local`, sockets del host ni servicios `systemd --user`. Solo se monta el home gestionado por `sandbox` cuando corresponde. `--shared-home` queda deshabilitado. Cualquier sección posterior que describa Distrobox o shared home es histórica y queda supersedida por esta decisión.

## 1. Resumen

Construir una CLI llamada `sandbox` para Omarchy/Arch Linux que permita crear y administrar entornos Linux aislados basados en contenedores para probar herramientas de software sin instalarlas en el sistema host.

La CLI debe actuar como una capa de experiencia de usuario sobre **Podman rootless**. No debe implementar un runtime OCI propio.

El objetivo principal es que un usuario pueda ejecutar:

```bash
sandbox create
```

y mediante una interfaz interactiva elegir un nombre, distribución y nivel de persistencia, obteniendo un entorno listo para usar en pocos segundos.

Ejemplo de uso:

```text
$ sandbox create

Create sandbox

Name
> gentle-ai

Distribution
> Arch Linux
  Ubuntu
  Fedora
  Debian

Persistence
> Disposable
  Persistent

Home
> Isolated
  Shared

Create sandbox? Yes

✓ Creating container
✓ Creating isolated home
✓ Sandbox ready

Entering gentle-ai...

[gentle-ai ~]$
```

---

## 2. Contexto

El sistema host objetivo es **Omarchy**, basado en Arch Linux. La herramienta debe evitar modificaciones del host fuera de su metadata, storage del runtime y homes gestionados.

El usuario quiere evaluar herramientas como:

- CLIs
- proyectos descargados desde GitHub
- herramientas Node/Bun/npm
- herramientas Go/Rust/Python
- servidores locales
- herramientas de IA
- aplicaciones web
- scripts de instalación
- proyectos que requieran instalar dependencias

sin instalar previamente esas dependencias o herramientas en el host principal.

El propósito de `sandbox` no es proporcionar aislamiento frente a software malicioso. Para software no confiable o que necesite modificar kernel, drivers, networking del host, systemd u otros componentes del sistema operativo se recomendará utilizar una máquina virtual.

---

# 3. Objetivos

La herramienta debe permitir:

1. Crear contenedores interactivos fácilmente.
2. Crear entornos persistentes o desechables.
3. Utilizar distintas distribuciones Linux.
4. Evitar por defecto compartir el `$HOME` real del usuario.
5. Entrar nuevamente en sandboxes persistentes.
6. Listar sandboxes existentes.
7. Detener sandboxes.
8. Eliminar sandboxes y opcionalmente sus datos.
9. Mantener una UX muy sencilla.
10. Funcionar completamente como usuario normal/rootless.
11. No requerir Docker daemon ni privilegios permanentes.
12. Mantener la lógica de contenedores delegada a Podman.

---

# 4. No objetivos

No implementar:

- runtime OCI propio;
- VM manager;
- reemplazo de Podman;
- Kubernetes;
- gestión de Docker Compose;
- sandbox de seguridad para ejecutar malware;
- integración gráfica;
- daemon propio;
- servicios privilegiados;
- administración de contenedores remotos.

La primera versión debe ser estrictamente una CLI local.

---

# 5. Stack técnico

## Lenguaje

Go.

Versión mínima recomendada:

```text
Go 1.24+
```

## Dependencias recomendadas

CLI:

```text
github.com/spf13/cobra
```

Interfaz interactiva:

```text
github.com/charmbracelet/huh
```

Opcional para estilos:

```text
github.com/charmbracelet/lipgloss
```

No introducir frameworks innecesarios.

## Runtime externo

La herramienta depende de:

```text
podman
```

La CLI debe detectar automáticamente si están instalados.

---

# 6. Arquitectura

La arquitectura conceptual debe ser:

```text
User
 │
 ▼
sandbox CLI
 │
 ▼
Podman rootless
 │
 ▼
Linux container
```

`sandbox` no debe interactuar directamente con namespaces, cgroups u OCI. Debe delegar la creación y gestión a comandos de Podman.

No montar `$HOME`, `/run/host`, `/usr/local`, sockets o servicios del host. El único bind mount permitido es el home gestionado por `sandbox`.

Podman puede utilizarse para inspección del estado cuando resulte conveniente.

---

# 7. Comandos

La V1 debe implementar:

```bash
sandbox create
sandbox list
sandbox enter
sandbox exec NAME -- COMMAND [ARG...]
sandbox stop
sandbox delete
sandbox info [--json]
sandbox doctor [--json]
sandbox version
```

Aliases aceptables:

```text
ls      -> list
rm      -> delete
shell   -> enter
```

---

# 8. `sandbox create`

Debe ser el flujo principal.

Ejecutar:

```bash
sandbox create
```

debe abrir un formulario interactivo.

## Pregunta 1 — nombre

```text
Name
> gentle-ai
```

Reglas:

- obligatorio;
- lowercase preferiblemente;
- permitir `a-z`, `0-9`, `_` y `-`;
- rechazar nombres ya existentes, salvo con `--if-not-exists`, que convierte
  la creación repetida en un no-op exitoso;
- eliminar espacios exteriores;
- mostrar error claro si el nombre es inválido.

---

## Pregunta 2 — distribución

```text
Distribution

> Arch Linux
  Ubuntu
  Fedora
  Debian
```

Valor por defecto:

```text
Arch Linux
```

Mapeo inicial:

```text
Arch Linux -> docker.io/library/archlinux:latest
Ubuntu     -> docker.io/library/ubuntu:24.04
Fedora     -> registry.fedoraproject.org/fedora:latest
Debian     -> docker.io/library/debian:stable
```

Debe existir una arquitectura que permita añadir distribuciones fácilmente.

No hardcodear la lógica de creación individualmente para cada distro.

---

# 9. Persistencia

Pregunta:

```text
Persistence

> Disposable
  Persistent
```

## Disposable

El contenedor debe eliminarse automáticamente al finalizar la sesión.

Conceptualmente equivalente a:

```bash
podman run --rm
```

Al salir:

```text
exit
```

debe eliminarse:

- contenedor;
- metadata temporal;
- home temporal si existe.

Mostrar:

```text
Destroying sandbox...
✓ Sandbox removed
```

No dejar residuos previsibles.

---

## Persistent

Debe crear un contenedor que sobreviva entre sesiones.

Ejemplo:

```bash
sandbox create
```

crea:

```text
gentle-ai
```

Después:

```bash
exit
```

no debe eliminarlo.

El usuario podrá ejecutar:

```bash
sandbox enter gentle-ai
```

---

# 10. HOME isolation

Pregunta:

```text
Home

> Isolated
  Shared
```

Valor por defecto:

```text
Isolated
```

## Isolated

El contenedor debe utilizar un home independiente.

Ruta recomendada:

```text
~/.local/share/sandbox/homes/<name>
```

Ejemplo:

```text
~/.local/share/sandbox/homes/gentle-ai
```

Podman debe crear el contenedor mediante un mecanismo equivalente a:

```bash
podman create \
  --name gentle-ai \
  --hostname gentle-ai \
  --workdir /home/sandbox \
  --env HOME=/home/sandbox \
  --volume ~/.local/share/sandbox/homes/gentle-ai:/home/sandbox \
  docker.io/library/archlinux:latest sleep infinity
```

Este debe ser el comportamiento predeterminado. No se permiten otros bind mounts del host.

---

## Shared

Deshabilitado en la V1. El home del host nunca se monta porque permitiría que instaladores modifiquen el sistema local.

---

# 11. Entrada automática

Después de crear correctamente un sandbox, preguntar:

```text
Enter sandbox now?

> Yes
  No
```

Default:

```text
Yes
```

Si se selecciona Yes, ejecutar automáticamente:

```bash
podman exec --interactive --tty <name> /bin/bash
```

---

# 12. Modo no interactivo

Todos los flujos importantes deben poder automatizarse.

Ejemplo:

```bash
sandbox create gentle-ai \
  --distro arch \
  --persistent \
  --isolated-home
```

Ejemplo disposable:

```bash
sandbox create gentle-ai \
  --distro arch \
  --disposable
```

Opciones esperadas:

```text
--distro
--persistent
--disposable
--isolated-home
--no-enter
```

Si faltan opciones, utilizar formulario interactivo.

---

# 13. `sandbox list`

Comando:

```bash
sandbox list
```

Salida propuesta:

```text
NAME          DISTRO      TYPE          HOME       STATUS
gentle-ai     arch        persistent    isolated   running
treehouse     arch        persistent    isolated   stopped
firstmate     ubuntu      persistent    isolated   running
```

La salida debe ser clara y alineada.

No utilizar una TUI compleja para esta pantalla.

Opciones:

```bash
sandbox list --json
```

Ejemplo:

```json
[
  {
    "name": "gentle-ai",
    "distro": "arch",
    "persistence": "persistent",
    "home": "isolated",
    "status": "running"
  }
]
```

---

# 14. `sandbox enter`

Uso:

```bash
sandbox enter gentle-ai
```

Debe ejecutar:

```bash
podman exec --interactive --tty gentle-ai /bin/bash
```

Si no existe:

```text
Sandbox "gentle-ai" does not exist.
```

Si el usuario ejecuta:

```bash
sandbox enter
```

sin especificar nombre y existen varios sandboxes, mostrar selector interactivo:

```text
Select sandbox

> gentle-ai
  treehouse
  firstmate
```

---

# 15. `sandbox stop`

Uso:

```bash
sandbox stop gentle-ai
```

Debe detener el contenedor sin eliminarlo.

Utilizar Podman directamente; no existe runtime alternativo.

---

# 16. `sandbox delete`

Uso:

```bash
sandbox delete gentle-ai
```

Mostrar:

```text
Delete sandbox "gentle-ai"?

> No
  Yes
```

Si el home es aislado:

```text
Also delete its isolated home?

> Yes
  No
```

Default:

```text
Yes
```

La eliminación debe gestionar:

- contenedor;
- metadata;
- home aislado si se confirma.

Debe evitar eliminar accidentalmente rutas que no estén dentro del directorio gestionado por `sandbox`.

Nunca ejecutar recursivamente un borrado sobre una ruta arbitraria suministrada por metadata sin comprobarla.

---

# 17. `sandbox info`

Uso:

```bash
sandbox info gentle-ai
```

También admite `--json` para automatización. Sus claves estables y los valores
posibles de `status` forman parte del contrato de la [interfaz para agentes](docs/agents.md).

Salida:

```text
Name          gentle-ai
Distribution  Arch Linux
Image         docker.io/library/archlinux:latest
Type          Persistent
Status        Running
Home          Isolated
Home path     ~/.local/share/sandbox/homes/gentle-ai
Created       2026-08-19 20:15
Container     gentle-ai
```

---

# 18. Metadata

La CLI deberá guardar exclusivamente metadata necesaria.

Ruta:

```text
~/.local/share/sandbox/
```

Propuesta:

```text
~/.local/share/sandbox/
├── sandboxes/
│   ├── gentle-ai.json
│   └── treehouse.json
├── homes/
│   ├── gentle-ai/
│   └── treehouse/
└── config.json
```

Ejemplo de metadata:

```json
{
  "name": "gentle-ai",
  "distribution": "arch",
  "image": "docker.io/library/archlinux:latest",
  "persistence": "persistent",
  "home_mode": "isolated",
  "home_path": "/home/user/.local/share/sandbox/homes/gentle-ai",
  "created_at": "2026-08-19T20:15:00+02:00"
}
```

La metadata no debe considerarse la única fuente de verdad para saber si un contenedor existe.

Contrastar con Podman cuando sea necesario.

---

# 19. Configuración

Archivo:

```text
~/.config/sandbox/config.toml
```

Ejemplo:

```toml
default_distro = "arch"
default_persistence = "disposable"
default_home = "isolated"
auto_enter = true
```

No es necesario implementar edición interactiva de configuración en la V1.

---

# 20. `sandbox doctor`

Debe diagnosticar el entorno.

Uso:

```bash
sandbox doctor [--json]
```

Ejemplo:

```text
Sandbox Doctor

✓ Podman installed
✓ Podman runtime working
✓ Podman rootless
✓ User namespaces configured

Everything looks good.
```

Comprobar:

- existencia de `podman`;
- Podman funciona;
- Podman funciona rootless;
- existencia/configuración de `/etc/subuid`;
- existencia/configuración de `/etc/subgid`;
- capacidad de obtener información del runtime.

Si algo falla:

```text
✗ Podman not installed

Install on Arch/Omarchy with:

sudo pacman -S podman
```

No ejecutar instalaciones automáticamente.

---

# 21. Preflight al arrancar

Antes de `create`, comprobar:

```text
podman
```

Si falta alguno, abortar elegantemente.

Ejemplo:

```text
Podman is required but was not found.

On Omarchy/Arch:

  sudo pacman -S podman
```

---

# 22. Seguridad

La CLI debe dejar claro que:

> Podman rootless evita modificaciones del filesystem del host fuera de los recursos gestionados por `sandbox`, pero no es una frontera de seguridad fuerte frente a exploits del kernel. Para software no confiable usa una VM.

No presentarlo como sandbox para software malicioso.

Para herramientas no confiables debería mostrarse eventualmente una recomendación de VM, pero no es necesario hacerlo en cada ejecución.

Especial atención a:

- no montar `$HOME` del host;
- no montar `/run/host`, `/usr/local` ni sockets del host;
- no usar `--privileged`;
- no ejecutar Podman como root;
- no usar `sudo podman`;
- isolated home por defecto;
- no montar `.ssh` explícitamente;
- no eliminar rutas fuera del directorio administrado.

---

# 23. Rootless

La herramienta debe asumir **Podman rootless**.

Nunca debe hacer:

```bash
sudo podman ...
```

El usuario ejecuta:

```bash
sandbox
```

como usuario normal.

Si Podman no está configurado correctamente en rootless, `sandbox doctor` deberá explicarlo.

---

# 24. Manejo de procesos

Al ejecutar Podman:

- conectar stdin/stdout/stderr cuando corresponda;
- propagar correctamente Ctrl+C;
- preservar el exit code;
- evitar ejecutar comandos mediante `sh -c` salvo necesidad absoluta;
- utilizar `exec.CommandContext`;
- pasar argumentos como array;
- evitar command injection.

Correcto:

```go
exec.Command("podman", "exec", "--interactive", "--tty", name, "/bin/bash")
```

Evitar:

```go
exec.Command("podman", "exec", "--interactive", "--tty", name, "/bin/bash")
```

---

# 25. UX

La UX debe inspirarse en herramientas modernas como:

```text
gum
gh
lazygit
once
```

pero sin crear una TUI completa.

Preferir:

- formularios simples;
- selectors;
- confirmaciones;
- spinners únicamente para operaciones que realmente tardan;
- output limpio.

Ejemplo:

```text
Creating gentle-ai

✓ Pulled archlinux:latest
✓ Created isolated home
✓ Created container

Sandbox ready.
```

No saturar la terminal con logs salvo que haya error.

Añadir:

```bash
--verbose
```

para debugging.

---

# 26. Colores

Respetar:

```text
NO_COLOR
```

Si:

```bash
NO_COLOR=1 sandbox list
```

no utilizar códigos ANSI de color.

---

# 27. Distribuciones

Crear una estructura interna similar a:

```go
type Distribution struct {
    ID    string
    Name  string
    Image string
}
```

Ejemplo:

```go
var distributions = []Distribution{
    {
        ID:    "arch",
        Name:  "Arch Linux",
        Image: "docker.io/library/archlinux:latest",
    },
    {
        ID:    "ubuntu",
        Name:  "Ubuntu",
        Image: "docker.io/library/ubuntu:24.04",
    },
}
```

Debe ser fácil añadir nuevas distribuciones.

---

# 28. Estructura sugerida del repositorio

```text
sandbox/
├── cmd/
│   ├── root.go
│   ├── create.go
│   ├── list.go
│   ├── enter.go
│   ├── stop.go
│   ├── delete.go
│   ├── info.go
│   ├── doctor.go
│   └── version.go
│
├── internal/
│   ├── podman/
│   │   └── client.go
│   ├── sandbox/
│   │   ├── manager.go
│   │   └── model.go
│   ├── metadata/
│   │   └── store.go
│   ├── config/
│   │   └── config.go
│   └── ui/
│       └── prompts.go
│
├── main.go
├── go.mod
├── go.sum
├── README.md
├── LICENSE
└── Makefile
```

No sobrearquitecturar.

Si alguna abstracción no aporta valor en V1, simplificarla.

---

# 29. Estado

Estados posibles:

```text
running
stopped
missing
unknown
```

No mantener artificialmente el estado en metadata.

Consultar el runtime.

---

# 30. Errores

Los errores deben ser legibles.

Mal:

```text
exit status 125
```

Bien:

```text
Could not create sandbox "gentle-ai".

Podman failed to pull:
docker.io/library/archlinux:latest

Run with --verbose for more details.
```

En `--verbose`, mostrar stderr original.

---

# 31. Atomicidad

Si crear un sandbox falla a mitad:

```text
create home
↓
pull image
↓
create container ERROR
```

la CLI debe intentar eliminar los recursos que haya creado durante esa operación.

No eliminar recursos que existieran previamente.

---

# 32. Disposable implementation

La implementación de sandboxes disposable debe usar contenedores Podman temporales y cleanup explícito:

```text
podman create
↓
podman start
↓
podman exec
↓
on exit
↓
podman rm --force
↓
delete managed home
```

La solución elegida debe:

- limpiar correctamente;
- sobrevivir a Ctrl+C razonablemente;
- no eliminar otros contenedores;
- tener tests.

---

# 33. Comando opcional futuro: `sandbox run`

No obligatorio para V1.

Concepto:

```bash
sandbox run arch
```

o:

```bash
sandbox run --distro arch
```

crearía inmediatamente un disposable sin formulario.

Ejemplo:

```text
$ sandbox run

Starting temporary Arch sandbox...

[sandbox ~]$
```

Al salir se elimina.

---

# 34. Comando opcional futuro: `sandbox clone`

No implementar en V1.

Concepto:

```bash
sandbox clone dev-base gentle-test
```

para duplicar un entorno persistente.

---

# 35. Comando opcional futuro: templates

No implementar en V1.

Futuro:

```text
node
rust
go
python
ai-tools
```

Ejemplo:

```bash
sandbox create --template node
```

podría crear Arch + Node + pnpm + git.

La V1 no debe incluir esta lógica.

---

# 36. Tests

Debe existir cobertura razonable para:

- validación de nombres;
- mapping de distros;
- metadata;
- configuración;
- generación de argumentos de Podman;
- protección del borrado de homes;
- interpretación del estado;
- cleanup tras errores.

Evitar tests que requieran Podman real salvo suite de integración separada.

Utilizar interfaces/fakes para ejecutar comandos externos.

---

# 37. Tests de integración

Añadir opcionalmente:

```bash
go test -tags=integration ./...
```

Estos tests sí podrán:

- crear un contenedor;
- comprobar que existe;
- detenerlo;
- eliminarlo.

No ejecutarlos por defecto.

---

# 38. Logging

No implementar sistema complejo de logging.

Default:

```text
human-friendly output
```

Con:

```bash
--verbose
```

mostrar:

```text
[podman] podman create ...
[podman] podman inspect ...
```

No mostrar secretos.

---

# 39. Versionado

Implementar:

```bash
sandbox version
```

Ejemplo:

```text
sandbox 0.1.0
```

Preparar variables inyectables mediante `ldflags`:

```text
version
commit
buildDate
```

---

# 40. Build

Debe bastar con:

```bash
go build ./...
```

Y para generar el binario:

```bash
go build -o sandbox .
```

---

# 41. Instalación local

README:

```bash
go build -o sandbox .
install -Dm755 sandbox ~/.local/bin/sandbox
```

Comprobar que:

```text
~/.local/bin
```

esté en `$PATH`.

---

# 42. Arch / Omarchy

El proyecto debe estar diseñado principalmente para:

```text
Arch Linux
Omarchy
```

Pero evitar dependencias internas específicas de Omarchy si no son necesarias.

Instalación de runtime en el README:

```bash
sudo pacman -S podman
```

No modificar automáticamente paquetes del sistema.

---

# 43. README

Debe contener:

1. descripción;
2. screenshots/asciinema placeholder;
3. instalación;
4. dependencias;
5. quick start;
6. comandos;
7. disposable vs persistent;
8. isolated home;
9. seguridad;
10. troubleshooting;
11. desarrollo.

Quick start:

```bash
sudo pacman -S podman

git clone <repo>
cd sandbox
go build -o sandbox

./sandbox doctor
./sandbox create
```

---

# 44. Flujo ideal de usuario

Usuario descubre una herramienta.

Por ejemplo:

```text
Gentle AI
```

Ejecuta:

```bash
sandbox create
```

Selecciona:

```text
Name: gentle-ai
Distro: Arch
Persistence: Disposable
Home: Isolated
```

Entra:

```text
[gentle-ai ~]$
```

Después:

```bash
sudo pacman -S git nodejs npm
git clone ...
cd ...
npm install
npm run dev
```

Prueba la herramienta.

Cuando termina:

```bash
exit
```

`sandbox` elimina:

```text
container
temporary home
metadata
```

El Omarchy host permanece sin esas dependencias.

---

# 45. Criterios de aceptación V1

La V1 se considerará completada cuando:

- `sandbox doctor` funciona en Omarchy/Arch;
- `sandbox create` crea un entorno Arch;
- permite elegir disposable o persistent;
- isolated home funciona;
- puede entrar automáticamente;
- disposable se elimina al salir;
- persistent sobrevive al salir;
- `sandbox list` muestra los persistentes;
- `sandbox enter NAME` funciona;
- `sandbox exec NAME -- COMMAND [ARG...]` ejecuta comandos sin TTY y propaga su código de salida;
- `sandbox stop NAME` funciona;
- `sandbox delete NAME` funciona;
- delete limpia el home aislado;
- no necesita root;
- no ejecuta Podman con sudo;
- funciona con Ctrl+C sin corromper metadata;
- tiene tests unitarios básicos;
- README documenta instalación y limitaciones.

---

# 46. Prioridad de implementación

Implementar en este orden:

```text
1. project skeleton
2. command runner abstraction
3. doctor
4. distro definitions
5. metadata store
6. create persistent
7. isolated home
8. enter
9. list
10. delete
11. stop
12. disposable
13. interactive UI
14. non-interactive flags
15. polish/errors
16. tests
17. README
```

No dedicar tiempo inicialmente a features futuras.

---

# 47. Decisiones importantes

Tomar estas decisiones como requisitos del producto:

```text
Runtime                  Podman rootless
Container UX             Podman abstraction
Language                 Go
CLI framework            Cobra
Interactive prompts      Huh
Default distro           Arch Linux
Default home             Isolated
Root                     Never required
Primary platform         Omarchy / Arch Linux
Security sandbox         No
VM replacement           No
```

---

# 48. Filosofía del producto

`sandbox` debe sentirse como una herramienta que el usuario pueda utilizar varias veces al día:

```bash
sandbox create
```

en lugar de pensar en:

```text
images
containers
volumes
OCI
namespaces
Podman flags
```

La CLI debe ocultar esa complejidad y presentar simplemente:

```text
"I want a temporary Linux environment to try this software."
```

Ese es el producto.

---

# 49. Interfaz para agentes

Los agentes sin TTY son usuarios de primera clase. El contrato máquina completo
vive en [`docs/agents.md`](docs/agents.md), y las instrucciones para agentes
que contribuyen al repositorio en `AGENTS.md`.

Roadmap de ergonomía (pendiente de implementar):

| Ítem | Beneficio para el agente |
|------|--------------------------|
| `info --json` | Detalles de un sandbox sin parsear texto |
| `doctor --json` | Veredicto de preparación del host en un round-trip |
| Códigos de error máquina (`E_EXISTS`, `E_NOT_FOUND`) | Distinguir clases de error sin matching de texto |
