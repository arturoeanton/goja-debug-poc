# Limitaciones Conocidas del Debugger

## 1. Step Into en Closures

### Problema
Cuando se hace `step into` (s) en una llamada a una función que es un closure (función retornada por otra función), el debugger no entra automáticamente en el closure.

### Ejemplo
```javascript
// Crear closure
let miContador = crearContador(10);

// Step into aquí NO entra en la función incrementar dentro del closure
miContador();  // El debugger pasa de largo
```

### Solución Alternativa
Para debuggear el interior de un closure, coloca un breakpoint dentro de la función closure:

```javascript
function crearContador(inicial) {
    let valor = inicial;
    
    function incrementar() {
        // Agregar breakpoint aquí con: b <línea>
        valor++;
        console.log("Contador incrementado a: " + valor);
        return valor;
    }
    
    return incrementar;
}
```

### Razón Técnica
Los closures en JavaScript son funciones que se crean dinámicamente y se almacenan como valores. La arquitectura de Goja ejecuta estas llamadas a través de valores dinámicos, no como llamadas directas a funciones, lo que impide que el debugger pueda interceptarlas para step-into.

### Ejemplos de Trabajo
Ver el archivo `test_closure.js` incluido para ejemplos completos de cómo debuggear closures efectivamente usando breakpoints.

## 2. Código Nativo y Interno

### Comportamiento
Cuando el debugger pasa por código nativo del motor (como `console.log` internamente), muestra temporalmente "Código interno" pero continúa automáticamente.

### Esto es Normal
El debugger está diseñado para no detenerse en código interno del motor JavaScript, solo en tu código fuente.

## 3. Funciones Eval

### Limitación
El código ejecutado con `eval()` puede no tener información completa de debugging.

### Recomendación
Evita usar `eval()` cuando estés debuggeando. Si necesitas evaluar expresiones, usa el comando `p <expresión>` del debugger.

## Notas para el Desarrollo Futuro

Estas limitaciones son inherentes a cómo funciona el motor Goja y su integración con el debugger. Para un debugging más efectivo:

1. Usa breakpoints estratégicamente en lugar de depender solo de step-into
2. Aprovecha el comando `p` para inspeccionar valores en cualquier momento
3. Usa `console.log` liberalmente para entender el flujo del programa