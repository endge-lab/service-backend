package main

import "github.com/endge-lab/service-backend/internal/bootstrap"

// @title Endge Service Backend API
// @version 1.0.20
// @description Production-ready backend Endge-сервиса с эталонными RedPanda-обёртками и строгими архитектурными границами.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	bootstrap.NewApp().Run()
}
