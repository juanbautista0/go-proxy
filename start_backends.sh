#!/bin/bash

echo "🚀 Starting optimized backend servers..."

# Función para limpiar procesos al salir
cleanup() {
    echo "🛑 Stopping all backend servers..."
    kill $(jobs -p) 2>/dev/null
    exit
}

# Capturar Ctrl+C
trap cleanup SIGINT

# Iniciar servidores en background
echo "Starting Backend 1 on :3001..."
go run backend1.go &

echo "Starting Backend 2 on :3002..."
go run backend2.go &

echo "Starting Backend 3 on :3003..."
go run backend3.go &

echo ""
echo "✅ All backends started!"
echo "📊 Test endpoints:"
echo "   - http://localhost:3001"
echo "   - http://localhost:3002" 
echo "   - http://localhost:3003"
echo ""
echo "Press Ctrl+C to stop all servers"

# Esperar indefinidamente
wait