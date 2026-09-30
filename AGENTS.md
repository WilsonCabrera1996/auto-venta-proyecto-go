# autoventa

Arquitectura **desacoplada**: cada capa vive en su propio servidor y se despliega por separado.
Este repo es un greenfield en progreso. No hay CI todavía; este archivo es la fuente
de verdad del proyecto.

## Cómo retomar el trabajo

Todo el código está en disco: apagar y volver a levantar no pierde nada del
backend. Lo que sí se pierde son los **datos**, porque los repos son en memoria.

```bash
cd backend
docker compose up -d --build   # reconstruye y levanta
curl localhost:8080/health     # debe devolver {"status":"ok","env":"development"}
```

Después de cada reinicio, el estado es este:

| Qué | Estado | Por qué |
| --- | --- | --- |
| Código y tests | intacto | vive en el filesystem |
| `dueno@autoventa.com` | **existe** | se re-siembra solo desde `.env`, es idempotente |
| Los vendedores | **hay que recrearlos** | no se siembran; hay que crearlos por API |
| Los vehículos | **hay que recargarlos** | repo en memoria |

Para recuperar a los vendedores tras un reinicio, con el token del dueño:

```bash
API=http://localhost:8080/api/v1
TOK=$(curl -s -X POST $API/session -H 'Content-Type: application/json' \
  -d '{"email":"dueno@autoventa.com","password":"<el de tu .env>"}' \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')

curl -s -X POST $API/usuarios -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOK" \
  -d '{"email":"ana@autoventa.com","nombre":"Ana","password":"Vendedor12345","role":"vendedor"}'
```

`frontend/` e `infrastructure/` están vacíos: existen como carpeta pero sin
contenido. Si se inicializa git, las carpetas vacías no se versionan — eso es
normal, `.gitkeep` no hace falta todavía.

## Capas y dónde se despliega

| Carpeta         | Stack            | Deployment |
| --------------- | ---------------- | ---------- |
| `frontend/`     | Astro            | Vercel     |
| `backend/`      | Go               | Render     |
| `infrastructure/` | Supabase (Postgres + Auth + Storage) | Supabase cloud (plan gratuito) |

## Reglas que no se negocian

- **Cero imports cruzados entre carpetas.** No es un monorepo con paquetes
  compartidos: son tres despliegues independientes. `frontend/` no importa nada de
  `backend/`; `backend/` no importa nada de `frontend/`.
- **La base de datos vive en Supabase, no en el backend.** El backend la consume por
  red, igual que el frontend consume al backend. Si una regla de negocio necesita
  datos, esa regla va en `backend/` y el acceso a la DB también.
- **Cada servicio tiene su propio `.env`.** Al estar en hosts distintos no se puede
  compartir un archivo. Cada carpeta gestiona sus propias variables.
- **Los secretos no salen del backend.** Ninguna credencial de Supabase (especialmente
  la `service_role`, que es admin) llega al frontend.

## Contratos entre capas

- `frontend` → `backend`: HTTP. La URL base del backend viene de una variable de
  entorno pública del frontend, no hardcodeada.
- `backend` → `infrastructure`: cliente de Supabase (Postgres + Auth), por red.

## Backend (`backend/`)

**Es un sistema modular, NO microservicios.** Un solo binario, un solo deploy,
capas internas. No crear un repo/servicio/proto por módulo — la modularidad va por
paquetes, no por red. Contrasta con el desacoplamiento *entre* carpetas, que sí es real.

- **Puerto 8080.** Es el puerto por defecto que Render inyecta vía `PORT`; el código
  lo lee del entorno con fallback a `8080` para que local y producción coincidan.
- **Cada módulo es autonomous.** Viven en `internal/modules/<nombre>/` con sus
  propias tres capas: `domain/` (entidades, invariantes, puertos — cero
  dependencias) → `usecase/` (casos de uso) → `infra/http/` (handler, DTOs,
  `RegistrarRutas`). Un módulo se borra borrando su carpeta.
- **Cross-cutting en `internal/infra/`, nunca dentro de un módulo.**
  `http/router.go` (ensamblado + middlewares), `http/middleware/` (incluye los
  guards de auth), `http/respond/` (forma de las respuestas), `config/`,
  `logger/`, `crypto/` (bcrypt), `platform/id/` (UUID v4). `internal/repos/` =
  implementaciones de los puertos.
- **`cmd/api/main.go` es el único composition root.** Solo ahí se conecta
  repo → usecase → handler. Ningún otro paquete arma el grafo.
- **La regla de dependencia es de una sola dirección:** las flechas apuntan
  **hacia** `domain/`. `domain/` no importa nada — ni de `usecase`, ni de `infra`.
  Eso obliga a que los errores de dominio (`domain.Err*`) vivan en el dominio y
  los traduzca `respond.Fail` a códigos HTTP; si el handler empieza a decidir
  reglas, se rompió la Clean Architecture.
- **`respond` importa `domain`, no al revés.** Al añadir un módulo nuevo, sus
  errores se agregan a su `domain/errors.go` y se mapean en `respond.Fail`.
  Errores de transporte (400 de JSON malformado) se construyen con
  `respond.BadRequest`, no se meten al dominio.

### Auth: cómo se autoriza (módulo `session`)

- **Los guards verifican PERMISOS, no roles.** `middleware.RequirePermiso(p)`
  pregunta `actor.Role.Puede(p)`. La matriz rol→permisos está en un solo lugar:
  `session/domain/user.go: permisosPorRol`. Si tocás un permiso, editás ahí.
- **Cada ruta declara su propio permiso** en su `RegistrarRutas`. No hay un
  switch central de autorización que se pueda olvidar de actualizar.
- **El vendedor:** ver, crear, **editar**, `cambiar_estado`. **No** borrar,
  **no** reportes, **no** administrar usuarios. Editar no le da un atajo: `PUT`
  acepta `status` pero igual pasa por la máquina de estados, así que no puede
  deshacer una venta. Hay test que lo fija.
- **Agregar una ruta protegida es un forgot de no olvidar el guard.** `GET
  /api/v1/vehicles` sin `RequirePermiso` sería público aunque el módulo exista.
- **`password_hash` nunca sale en un DTO.** Si agregás un campo al `User`, no lo
  pases por `toResponse` sin pensar; hay test implícito en la suite de sesión.
- **Login devuelve siempre el mismo error** para email inexistente y password
  malo: no se puede enumerar qué emails existen. Hay test que lo fija.

### Tests

`go test ./...` — la suite corre **sin base de datos y sin HTTP**: los casos de uso
se prueban contra `repos.VehicleMemory`, `repos.UserMemory` y
`repos.SessionMemory`. El contrato `domain.Repository` no cambia cuando se
conecte Supabase, así que la suite sigue valiendo.

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.22-alpine \
  sh -c 'gofmt -l . && go vet ./... && go test ./...'
docker run --rm -v "$PWD":/app -w /app golang:1.22-alpine \
  sh -c 'go test ./internal/modules/vehicle/... -run TestTransicion -v'
```

**No hay Go instalado en el host**: `gofmt`, `go vet`, `go build` y `go test`
corren dentro de la imagen `golang:1.22-alpine` montando el repo. Editar código
sin pasar por ahí es no tener garantía de que compila.

### Gotchas que ya muerden

- **`CORS_ALLOWED_ORIGINS` vacío = frontend inutilizable.** Frontend (Vercel) y
  backend (Render) son orígenes distintos; sin esa variable el navegador bloquea
  todo antes de que llegue al backend. `.env` y `.env.example` la traen puesta.
- **Nombre clash:** el paquete `internal/infra/http` se importa como `apihttp`
  porque `net/http` ya ocupa el identificador `http` en `main.go`.
- **El precio es `int64` en centavos.** El dominio no usa float para dinero. El
  DTO acepta `"18500.00"` y también `"18500,50"`; el mapeo está en
  `modules/vehicle/infra/http/mapping.go`.
- **Casting de marca/modelo: no.** El dominio solo hace `trim`, no title-case;
  `BMW` y `KIA` deben sobrevivir intactas. Hay test que lo fija.
- **`vehicleMemory` mantiene un índice `byVIN` además de `byID`.** El lookup por
  VIN no puede recorrer el mapa entero; si se agrega un índice, hay que
  actualizarlo en `Create`, `Update` **y** `Delete`. Lo mismo para
  `UserMemory.byEmail`.
- **`ctxKey` es un solo tipo declarado en `middleware/requestid.go`.** Si lo
  redeclarás en otro archivo del paquete no compila. Las claves de auth arrancan
  en 100 (`iota + 100`) para no pisar `requestIDKey`.
- **`golang.org/x/crypto` está pineado a `v0.31.0`.** La última versión exige Go
  1.26 y el `Dockerfile` va en `golang:1.22-alpine`; un `go get` sin versión
  rompe el build. Es la única dependencia externa del proyecto.
- **Los repos en memoria son por proceso.** Reiniciar el contenedor borra
  vehículos, usuarios y sesiones. Normal hasta que exista Postgres; después,
  `Render` además reinicia en cada deploy del plan gratuito.
- **`OWNER_PASSWORD` va en el panel de Render, no en el repo.** `.env` está en
  `.gitignore`; `.env.example` lleva un placeholder, nunca la contraseña real.
  El siembra del dueño solo corre en el primer arranque y es idempotente.

### Comandos

```bash
docker compose up -d --build      # levantar (recrea si cambió el código)
docker compose logs -f api        # logs
docker compose down               # parar
docker compose down -v            # parar y limpiar
curl localhost:8080/health        # smoke test
```

### Rutas del módulo `vehicle`

Base `/api/v1/vehicles`. Códigos: `201` creado, `204` borrado, `404` no existe,
`409` VIN duplicado o transición de estado inválida, `422` invariantes del
dominio, `400` JSON/Content-Type inválido.

| Método  | Ruta                       | Permiso                  |
| ------- | -------------------------- | ------------------------ |
| `POST`   | `/api/v1/vehicles`         | `vehiculo:crear`         |
| `GET`    | `/api/v1/vehicles`         | `vehiculo:ver`           |
| `GET`    | `/api/v1/vehicles/{id}`    | `vehiculo:ver`           |
| `PUT`    | `/api/v1/vehicles/{id}`    | `vehiculo:editar`        |
| `PATCH`  | `/api/v1/vehicles/{id}/estado` | `vehiculo:cambiar_estado` |
| `DELETE` | `/api/v1/vehicles/{id}`    | `vehiculo:eliminar`      |
| `GET`    | `/api/v1/reportes/ventas`  | `reporte:ventas`         |

Query params de `Listar`: `status`, `brand`, `search`, `min_year`, `max_year`,
`min_price`, `max_price`, `order_by`, `asc`, `page`, `per_page` (topado a 100).
`/api/v1/reportes/ventas` acepta `mes=YYYY-MM` (por defecto, el mes en curso).

### Rutas del módulo `session`

| Método  | Ruta                          | Permiso                 |
| ------- | ----------------------------- | ----------------------- |
| `POST`   | `/api/v1/session`             | — (público: es el login)|
| `POST`   | `/api/v1/session/logout`      | autenticado             |
| `GET`    | `/api/v1/session/yo`          | autenticado             |
| `POST`   | `/api/v1/usuarios`            | `usuario:administrar`   |
| `GET`    | `/api/v1/usuarios`            | `usuario:administrar`   |
| `PATCH`  | `/api/v1/usuarios/{id}/rol`   | `usuario:administrar`   |
| `DELETE` | `/api/v1/usuarios/{id}`       | `usuario:administrar`   |

`401` = no autenticado (siempre con `WWW-Authenticate: Bearer`), `403` =
autenticado pero sin permiso. El token viaja en `Authorization: Bearer <token>`
y se devuelve **en claro una única vez**, en la respuesta del login.

### El contenedor simula Render a propósito

- **`deploy.resources.limits.memory: 512M`** replica la RAM del plan gratuito de
  Render. Si el servicio OOM-killea en local, es señal real de que no cabe allá.
- **Red propia `autoventa_net`**, no la default del host. Refuerza la regla de
  "cero imports cruzados": el backend no alcanza al frontend ni a nada fuera de su red.
- Solo se publica `8080`; el resto del servicio habla por nombre de servicio interno.
- Build multi-stage (`golang:alpine` → `alpine`, `CGO_ENABLED=0`) para que la imagen
  final sea mínima, como en Render.

## Gotchas de despliegue

- **Render (plan gratuito) hiberna el servicio** y el primer request tras un rato
  inactivo tarda (cold start). No poner ahí lógica que requiera baja latencia.
- **Render no tiene filesystem persistente** entre despliegues: nada de guardar
  uploads o archivos generados en disco local. Van a Supabase Storage.
- **En Vercel, solo las variables con prefijo `PUBLIC_`** llegan al bundle del
  cliente. Cualquier otra se ignora en el navegador (a propósito).

## Convention de idioma

El usuario escribe en español — responder en español.
