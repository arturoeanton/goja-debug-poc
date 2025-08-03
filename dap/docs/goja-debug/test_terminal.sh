#\!/bin/bash
# Configurar terminal antes de ejecutar el debugger
stty sane
stty echo
./goja-debug "$@"
