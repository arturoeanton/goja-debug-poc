# Problema: Script Termina Prematuramente

## Descripción

Después de hacer step-over en la línea 4 (`console.log(...)`), el script termina en lugar de continuar a la línea 6 (`if (sum > 10)`).

## Análisis del Flujo

1. **Línea 1-3**: Step-over funciona correctamente ✓
2. **Línea 4**: Para en el breakpoint ✓
3. **Step-over en línea 4**:
   - Columna 1 → 23 (se salta correctamente)
   - Columna 23 → Script termina ❌

## Causa Probable

El problema ocurre porque:

1. El debugger hace step-over en línea 4
2. Goja reporta múltiples posiciones en la misma línea (columnas 1, 23, etc.)
3. Nuestro código SALTA estas posiciones para evitar paradas múltiples
4. Al saltar, el modo es `DebugContinue` que continúa ejecutando
5. El script termina sin parar en la línea 6

## Solución Propuesta

Necesitamos ser más inteligentes sobre cuándo usar `DebugContinue` vs continuar con step-over:

```go
if (misma_línea && step_over) {
    // No usar DebugContinue, sino continuar con step
    // para asegurar que pare en la siguiente línea
    return goja.DebugStepOver  // en lugar de DebugContinue
}
```

## Workaround Actual

Por ahora, cuando estés en un `console.log` o función similar:
1. Usa **Continue (F5)** en lugar de Step Over
2. O pon un breakpoint en la siguiente línea que quieras depurar

## Estado

Este es un problema complejo de la interacción entre:
- El mapeo de columnas de Goja
- Nuestra lógica de skip
- El modo de continuación

Requiere un análisis más profundo del comportamiento de Goja.