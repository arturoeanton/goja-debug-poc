# Guía de Mejora del Debugger para Closures

## Problema Identificado

Cuando se ejecuta código como:
```javascript
let miSumador = crearSumador(100);
miSumador(10);  // Step-into no entra en la función closure
```

El debugger no puede hacer step-into en la función closure porque:
1. Los closures son funciones almacenadas como valores
2. La VM ejecuta estas llamadas de manera diferente a las funciones regulares
3. El mecanismo actual de step-into no detecta estas llamadas

## Limitación Actual de Goja

Después de analizar el código fuente de Goja, identificamos que:

1. **Arquitectura de VM**: Las llamadas a funciones se manejan a través de instrucciones de bytecode compiladas
2. **Closures como valores**: Los closures se almacenan como valores en variables y se ejecutan dinámicamente
3. **Punto de intercepción**: No hay un punto único donde interceptar todas las llamadas a closures

## Solución Alternativa

### Opción 1: Breakpoints Manuales

La solución más práctica es establecer breakpoints dentro de las funciones closure:

```javascript
function crearSumador(inicial) {
    let total = inicial;
    
    function sumar(cantidad) {
        // Establecer breakpoint aquí (línea 8)
        total += cantidad;
        console.log("Total ahora es: " + total);
        return total;
    }
    
    return sumar;
}
```

### Opción 2: Usar Step-Over con Breakpoints

1. Establecer un breakpoint en la primera línea de la función closure
2. Usar step-over hasta llegar a la llamada
3. El debugger se detendrá automáticamente en el breakpoint interno

### Opción 3: Modificación Profunda de Goja (No Recomendada)

Para implementar step-into completo en closures se requeriría:

1. **Modificar el compilador** para generar instrucciones especiales antes de llamadas a valores
2. **Interceptar en la VM** todas las operaciones de llamada a valores que sean funciones
3. **Tracking de contexto** para distinguir entre llamadas regulares y closures
4. **Impacto en performance** significativo al verificar cada llamada

## Recomendación

Dado que:
- La modificación requerida sería muy invasiva
- Afectaría la performance incluso cuando el debugger no está activo
- Requeriría cambios fundamentales en la arquitectura de Goja

**Recomendamos usar la Opción 1**: Establecer breakpoints manuales en las funciones closure que se deseen debuggear.

## Ejemplo de Uso Recomendado

```javascript
// test_closure_debug.js
function crearContador() {
    let count = 0;
    
    return {
        incrementar: function() {
            // BREAKPOINT: Establecer aquí para debuggear
            count++;
            return count;
        },
        
        decrementar: function() {
            // BREAKPOINT: Establecer aquí para debuggear
            count--;
            return count;
        },
        
        obtener: function() {
            // BREAKPOINT: Establecer aquí para debuggear
            return count;
        }
    };
}

let contador = crearContador();

// Usar step-over aquí, el debugger se detendrá en los breakpoints internos
contador.incrementar();  // Se detendrá en línea 7
contador.incrementar();  // Se detendrá en línea 7
contador.decrementar();  // Se detendrá en línea 13
```

## Comandos del Debugger

```bash
# Establecer breakpoints en las funciones closure
b test_closure_debug.js 7    # incrementar
b test_closure_debug.js 13   # decrementar
b test_closure_debug.js 19   # obtener

# Ejecutar hasta el primer breakpoint
c

# Una vez dentro de la función closure, usar step normalmente
n  # step over
s  # step into (para funciones internas)
```

## Conclusión

Aunque el step-into directo en closures no está soportado debido a limitaciones arquitecturales de Goja, el uso de breakpoints estratégicos proporciona una experiencia de debugging efectiva sin comprometer la performance del motor JavaScript.