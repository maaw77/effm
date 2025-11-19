package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maaw77/effm/config"
	"github.com/maaw77/effm/internal/database"
	"github.com/maaw77/effm/internal/server"

	// === Swagger ===
	_ "github.com/maaw77/effm/docs"            // сгенерировано командой swag init
	swaggerFiles "github.com/swaggo/files"     // ← обязательно с алиасом
	ginSwagger "github.com/swaggo/gin-swagger" // ← обязательно с алиасом
)

func main() {
	connStr := config.InitConnString("config/config.yaml")
	serverCfg := config.InitServerConfig()
	log.Printf("server will listen on :%s", serverCfg.Port)

	db, err := database.NewSubscriptionDatabase(context.Background(), connStr)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	srv := server.NewServer(db)

	// Swagger UI — доступен по http://localhost:8080/swagger/index.html
	srv.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	httpServer := &http.Server{
		Addr:         ":" + serverCfg.Port,
		Handler:      srv.Router,
		ReadTimeout:  serverCfg.ReadTimeout,
		WriteTimeout: serverCfg.WriteTimeout,
	}

	go func() {
		log.Printf("server started at http://localhost:%s", serverCfg.Port)
		log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", serverCfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	} else {
		log.Println("server stopped gracefully")
	}
}
