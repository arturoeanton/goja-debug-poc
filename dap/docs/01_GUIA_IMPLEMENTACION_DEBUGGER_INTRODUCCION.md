# Guía de Implementación del Debugger de Goja - Introducción

## Índice de Documentos

1. **01_GUIA_IMPLEMENTACION_DEBUGGER_INTRODUCCION.md** (Este archivo)
   - Visión general del proyecto
   - Lista de archivos a modificar
   - Resumen de funcionalidades

2. **02_GUIA_IMPLEMENTACION_DEBUGGER_CORE.md**
   - Implementación del debugger en el core de Goja
   - Modificaciones en debugger.go

3. **03_GUIA_IMPLEMENTACION_RUNTIME_VM.md**
   - Integración con el Runtime
   - Modificaciones en runtime.go y vm.go

4. **04_GUIA_IMPLEMENTACION_COMPILER.md**
   - Modo debug en el compilador
   - Prevención de optimizaciones

5. **05_GUIA_IMPLEMENTACION_API.md**
   - API pública del debugger
   - Estructuras y métodos expuestos

6. **06_GUIA_IMPLEMENTACION_EJEMPLO_CONSOLA.md**
   - Implementación de la consola de debugging
   - UI interactiva completa

7. **07_GUIA_VERIFICACION_PERFORMANCE.md**
   - Tests de performance para verificar zero impact
   - Benchmarks comparativos

8. **08_GUIA_MEJORA_CLOSURES.md**
   - Limitaciones con closures y soluciones alternativas
   - Mejores prácticas para debugging de closures

## Visión General

Este conjunto de guías documenta cómo implementar un debugger completo para Goja, el motor de JavaScript en Go. Las mejoras incluyen:

### Funcionalidades Principales

1. **Debugger Integrado**
   - Breakpoints por línea
   - Step Into/Over/Out
   - Inspección de variables (locales, closure, globales)
   - Evaluación de expresiones en contexto
   - Stack trace completo

2. **Modo Debug del Compilador**
   - Desactiva todas las optimizaciones
   - Fuerza todas las variables al stash para inspección
   - Preserva información de debugging

3. **Consola Interactiva**
   - UI estilo QBasic con múltiples ventanas
   - Soporte para terminal con colores
   - Historial de comandos con repetición (Enter vacío repite último comando)
   - Filtrado de variables
   - Salida en dos columnas: programa vs mensajes del debugger
   - Stack trace con comando 'st'

### Archivos a Modificar en Goja

```
github.com/dop251/goja/
├── debugger.go              (NUEVO - Core del debugger)
├── runtime.go               (Modificar - Integración con debugger)
├── vm.go                    (Modificar - Hooks para debugging)
├── compiler.go              (Modificar - Modo debug)
├── compiler_expr.go         (Modificar - Prevenir optimizaciones)
├── value.go                 (Modificar - Exportar información de valores)
└── examples/
    └── debugger/
        └── goja-debug/      (NUEVO - Ejemplo de consola)
            ├── main.go
            ├── go.mod
            ├── run-debugger.sh
            └── *.md (documentación)
```

## Requisitos Previos

- Go 1.16 o superior
- Conocimiento básico de la arquitectura de Goja
- Familiaridad con conceptos de debugging

## Proceso de Implementación

### Fase 1: Core del Debugger
1. Crear `debugger.go` con la estructura básica
2. Definir interfaces y tipos de datos
3. Implementar sistema de breakpoints

### Fase 2: Integración con Runtime/VM
1. Modificar Runtime para soportar debugger
2. Agregar hooks en la VM para pausar ejecución
3. Implementar sistema de stepping

### Fase 3: Modo Debug del Compilador
1. Agregar flag debugMode al compilador
2. Modificar generación de código para preservar información
3. Desactivar optimizaciones en modo debug

### Fase 4: API Pública
1. Exponer métodos para control del debugger
2. Crear estructuras para inspección de estado
3. Documentar la API

### Fase 5: Ejemplo de Implementación
1. Crear consola interactiva
2. Implementar UI con ventanas
3. Agregar comandos y funcionalidades

## Notas Importantes

1. **Compatibilidad**: Todos los cambios son retrocompatibles. El debugger es opcional y no afecta el rendimiento cuando no está activo.

2. **Thread Safety**: El debugger NO es thread-safe. Cada Runtime debe tener su propio debugger.

3. **Performance**: En modo debug, el rendimiento se reduce significativamente debido a la desactivación de optimizaciones.

4. **Limitaciones Conocidas**:
   - Step Into no funciona automáticamente con closures (ver [08_GUIA_MEJORA_CLOSURES.md](08_GUIA_MEJORA_CLOSURES.md))
   - El código eval() tiene soporte limitado
   - No hay soporte para debugging remoto

## Siguiente Paso

Continúa con [02_GUIA_IMPLEMENTACION_DEBUGGER_CORE.md](02_GUIA_IMPLEMENTACION_DEBUGGER_CORE.md) para comenzar la implementación del core del debugger.