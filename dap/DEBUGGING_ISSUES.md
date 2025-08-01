# Solución de Problemas de Debugging

## Problema: Callbacks Duplicados en VS Code

Si VS Code muestra callbacks duplicados al poner breakpoints, puede ser por estas razones:

### Causas Posibles:

1. **Breakpoints no se limpian correctamente** - El DAP server ahora registra cuando remueve breakpoints
2. **VS Code envía múltiples requests** - Esto es normal, el server debe manejarlos correctamente
3. **El archivo tiene caracteres extraños** - Asegúrate de que el archivo JS esté limpio

### Solución:

1. **Reinicia el DAP server completamente**:
   ```bash
   # Mata cualquier proceso anterior
   pkill -f "gojs.*-port"
   
   # Inicia el server con logs
   ./debug-server.sh
   ```

2. **En VS Code**:
   - Cierra todas las sesiones de debug (Shift+F5)
   - Remueve todos los breakpoints (en el panel de Breakpoints)
   - Recarga la ventana (Cmd+R en Mac)
   - Vuelve a poner los breakpoints
   - Inicia el debug (F5)

3. **Verifica los logs**:
   Los logs del server ahora muestran:
   - Cuando se agregan/remueven breakpoints
   - Cuando se hit un breakpoint
   - El flujo de comandos DAP

### Mejoras Implementadas:

1. **Mejor limpieza de breakpoints**: El server ahora remueve TODOS los breakpoints del archivo antes de agregar nuevos
2. **Logging mejorado**: Cada operación importante se registra
3. **Formateo de console.log**: Los argumentos se formatean correctamente con espacios

## Uso Recomendado:

1. Usa archivos JavaScript simples y limpios
2. Siempre reinicia el server si ves comportamiento extraño
3. Revisa los logs para entender qué está pasando

## Ejemplo de Sesión Normal:

```
[10:23:45] Debug adapter listening on port 5678
[10:23:50] SetBreakpoints request for file: test.js, breakpoints: 1
[10:23:50] Added breakpoint: file=test.js, line=4, gojaID=1, dapID=1
[10:23:52] Hit breakpoint at test.js:4
[10:23:55] Resuming with command: 0
```