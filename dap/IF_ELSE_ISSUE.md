# Problema del If-Else en el Debugger

## Descripción del Problema

Cuando se hace step-over en un statement if-else, el debugger pasa por ambas ramas (if y else) aunque solo se ejecute una.

## Causa

Este es un comportamiento conocido en debuggers de JavaScript. El problema ocurre porque:

1. **AST Representation**: El parser de JavaScript trata el if-else como un solo statement
2. **Source Mapping**: Las posiciones del source code reportadas por Goja incluyen todo el bloque if-else
3. **Bytecode Generation**: El compilador genera instrucciones que pueden incluir referencias a ambas ramas

## Comportamiento Observado

```javascript
if (sum > 10) {
    console.log("Greater");  // Se ejecuta
} else {
    console.log("Less");     // El debugger pasa por aquí pero no se ejecuta
}
```

El debugger reporta posiciones para:
1. La condición del if
2. El bloque then 
3. El bloque else (aunque no se ejecute)

## Soluciones Posibles

### 1. A nivel de DAP Adapter (Parcial)
- Agregar logging detallado de PC (Program Counter) y columnas
- Filtrar pasos redundantes basándose en el contexto

### 2. A nivel de Goja VM (Completa)
- Modificar el generador de bytecode para no reportar posiciones de ramas no ejecutadas
- Requiere cambios en el core de Goja

### 3. Workaround para Usuarios
- Usar breakpoints en lugar de step-over para este tipo de código
- Usar "Continue" (F5) en lugar de "Step Over" (F10) cuando esté en la rama else

## Estado Actual

He agregado logging mejorado que muestra:
- Línea y columna exacta
- Program Counter (PC)
- Esto ayuda a diagnosticar pero no resuelve completamente el problema

El comportamiento es similar al de otros debuggers de JavaScript (como el debugger de Chrome en algunos casos).

## Recomendación

Para una experiencia de debugging más fluida con if-else:
1. Usa breakpoints dentro de las ramas específicas
2. Usa "Continue" para saltar a los breakpoints
3. El step-over funciona mejor con código lineal sin branches