# Guía de Uso - Debugger de Goja

## Instalación y Compilación

```bash
# Compilar el debugger
go build -o goja-debug

# Dar permisos de ejecución al script de inicio
chmod +x run-debugger.sh
```

## Ejecutar el Debugger

Para evitar problemas con el terminal (Enter, flechas), usa el script wrapper:

```bash
./run-debugger.sh ejemplo_debug.js
```

O directamente:

```bash
./goja-debug ejemplo_debug.js
```

## Interfaz de Usuario

La interfaz está dividida en 4 ventanas principales:

1. **Ventana de Código (izquierda)**: Muestra el código fuente con la línea actual resaltada
2. **Ventana de Variables (derecha)**: Muestra variables locales o globales
3. **Ventana de Consola (abajo)**: Dividida en dos columnas:
   - Columna izquierda: Salida del programa (console.log)
   - Columna derecha: Mensajes del debugger (breakpoints, comandos, etc.)
4. **Ventana de Comandos (inferior)**: Para ingresar comandos

## Comandos Disponibles

### Control de Ejecución
- `c` o `continuar` o `F8`: Continuar la ejecución hasta el próximo breakpoint
- `n` o `siguiente` o `F10`: Ejecutar la siguiente línea (step over)
- `s` o `entrar` o `F11`: Entrar en la función (step into)
- `o` o `salir`: Salir de la función actual (step out)
- `q` o `salir`: Salir del debugger
- `Enter` (vacío): Repetir el último comando ejecutado

### Breakpoints
- `b <línea>`: Agregar un breakpoint en la línea especificada
  - Ejemplo: `b 15` agrega un breakpoint en la línea 15

### Inspección de Variables
- `p <expresión>` o `evaluar <expresión>`: Evaluar una expresión JavaScript
  - Ejemplo: `p contador * 2`
  - Ejemplo: `p persona.nombre`
- `L` o `locales`: Mostrar variables locales
- `G` o `globales`: Mostrar variables globales
- `f <texto>` o `filtrar <texto>`: Filtrar variables por nombre
  - Ejemplo: `f cont` muestra solo variables que contengan "cont"
- `F` o `quitarfiltro`: Quitar el filtro de variables

### Navegación
- `↑` / `↓`: Navegar por el historial de comandos
- `PgUp` / `PgDn`: Hacer scroll en la ventana de variables
- `st` o `stack`: Alternar entre mostrar variables y stack trace

## Características Especiales

### Modo Debug
El debugger ejecuta en modo debug, lo que significa:
- Todas las variables son accesibles para inspección
- No se aplican optimizaciones del compilador
- Se puede evaluar cualquier expresión en el contexto actual

### Historial de Comandos
- Los comandos se guardan en el historial
- Usa las flechas ↑/↓ para navegar por comandos anteriores
- Presiona Enter sin escribir nada para repetir el último comando

### Consola de Dos Columnas
- La salida está separada para mayor claridad:
  - **Columna izquierda**: Muestra la salida del programa (console.log)
  - **Columna derecha**: Muestra mensajes del debugger (breakpoints agregados, comandos repetidos, etc.)
- Esto facilita distinguir entre la salida del programa y la información del debugger

### Filtrado de Variables
- Muy útil cuando hay muchas variables
- El filtro no distingue mayúsculas/minúsculas
- Se muestra el filtro activo en el título de la ventana

## Ejemplos de Uso

### Debugging básico
1. Ejecuta el programa: `./run-debugger.sh ejemplo_debug.js`
2. El programa se detiene en la primera línea
3. Presiona `n` para avanzar línea por línea
4. Presiona `s` para entrar en una función
5. Presiona `c` para continuar hasta el final

### Inspeccionar variables
1. Cuando el programa está pausado, presiona `L` para ver variables locales
2. Presiona `G` para ver variables globales
3. Usa `p nombreVariable` para ver el valor de una variable específica

### Usar breakpoints
1. Presiona `b 20` para poner un breakpoint en la línea 20
2. Presiona `c` para continuar hasta ese breakpoint
3. El programa se detendrá automáticamente en esa línea

### Filtrar variables
1. Si hay muchas variables, usa `f texto` para filtrar
2. Por ejemplo, `f cont` mostrará solo variables que contengan "cont"
3. Presiona `F` para quitar el filtro

## Solución de Problemas

### El Enter muestra ^M
Usa el script `run-debugger.sh` en lugar de ejecutar directamente

### Las flechas no funcionan
Asegúrate de usar el script wrapper o configura tu terminal correctamente

### No veo todas las variables
1. Verifica si estás viendo locales (L) o globales (G)
2. Quita cualquier filtro activo con `F`
3. Usa PgUp/PgDn para hacer scroll si hay muchas variables

### Step Into no entra en closures
Esta es una limitación conocida. Cuando haces step-into en una función que es un closure (como `miContador()`), el debugger no puede entrar automáticamente. 

**Solución**: Coloca un breakpoint dentro del closure con `b <línea>`:
```javascript
let miFunc = crearClosure();
miFunc(); // Step-into aquí no funciona

// En cambio, pon un breakpoint dentro del closure:
function crearClosure() {
    return function() {
        // b 8 (si esta es la línea 8)
        console.log("Dentro del closure");
    }
}
```

## Logs de Debug

El debugger crea un archivo `goja.debug.log` con información detallada de la ejecución. Útil para diagnosticar problemas.