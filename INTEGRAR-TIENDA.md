# Cómo publicar tu stock en Muchi 🐱

Si tienes una tienda de cartas y quieres que tus precios aparezcan en
[Muchi](https://github.com/metaliaw/muchi), hay dos caminos.

## Camino 1: listate en scry.cl (el más fácil)

[scry.cl](https://scry.cl) ya indexa 30 tiendas chilenas y Muchi lee de ahí.
Si te sumás a scry, aparecés en Muchi automáticamente — y en cualquier otra
herramienta que use scry. **No tienes que mantener nada.**

Es la opción recomendada.

## Camino 2: exponé una API de sólo lectura

Si preferís no depender de un tercero, Muchi puede consultar tu tienda directo.
Lo único que necesita es un endpoint `GET` que devuelva JSON.

**Muchi nunca escribe.** Sólo hace `GET`, con un mínimo de 1,5 s entre
peticiones y un `User-Agent` identificable.

### Si usás Supabase

Ya tienes la API: Supabase expone PostgREST. Lo único que falta es una política
que permita **leer sólo lo que quieras publicar**.

La clave `anon` está diseñada para ser pública — lo que la hace segura es el
Row Level Security. Con esto, esa clave sólo puede hacer `SELECT` sobre tu
inventario, y nada más:

```sql
-- 1. Una vista con SÓLO las columnas públicas.
--    Tus costos, proveedores y márgenes no salen de acá.
create view public.stock_publico as
select id, nombre_carta, set_codigo, estado, idioma,
       precio_clp, cantidad, enlace
from public.inventario
where cantidad > 0;

-- 2. RLS activo y una única política: leer.
alter table public.inventario enable row level security;

create policy "lectura publica del stock"
on public.inventario
for select
to anon
using (cantidad > 0);
```

Verificá que quedó bien — esto tiene que fallar:

```bash
curl -X POST "https://TU-PROYECTO.supabase.co/rest/v1/inventario" \
  -H "apikey: TU_ANON_KEY" -H "Content-Type: application/json" \
  -d '{"nombre_carta":"prueba"}'
```

Y esto tiene que funcionar:

```bash
curl "https://TU-PROYECTO.supabase.co/rest/v1/stock_publico?nombre_carta=ilike.*sol%20ring*" \
  -H "apikey: TU_ANON_KEY"
```

### Si tienes otro backend

Cualquier endpoint que reciba un término de búsqueda y devuelva una lista JSON
sirve. Por ejemplo `GET https://tutienda.cl/api/stock?buscar=sol+ring`:

```json
[
  {
    "carta": "Sol Ring",
    "set": "C21",
    "estado": "NM",
    "idioma": "EN",
    "precio": 3500,
    "cantidad": 4,
    "link": "https://tutienda.cl/producto/sol-ring-c21"
  }
]
```

Los nombres de los campos no importan: se mapean en la config.

### Conectarlo

Copiá `store-api.example.json` a `store-api.json` y completá tu tienda.
La clave va en una variable de entorno, nunca en el archivo:

```bash
export WOMBAT_SUPABASE_ANON_KEY="tu-anon-key"
```

## Qué muestra Muchi

Nombre de la carta, edición, estado, idioma, precio y stock — y un link que
manda a comprar **a tu tienda**. Muchi no vende, no cobra comisión y no
automatiza checkout ni pagos.

## Qué NO hace Muchi

- No escribe en tu base de datos
- No guarda datos de tus clientes
- No usa credenciales que no nos hayas dado explícitamente
- No hace scraping de tiendas que no lo permiten

¿Dudas o quieres que te ayudemos a configurarlo? Abre un issue en
[github.com/metaliaw/muchi](https://github.com/metaliaw/muchi/issues).
