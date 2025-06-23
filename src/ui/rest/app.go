package rest

import (
	"fmt"

	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type App struct {
	Service domainApp.IAppUsecase
}

func InitRestApp(app *fiber.App, service domainApp.IAppUsecase) App {
	rest := App{Service: service}
	
	// Dashboard routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login")
	})
	app.Get("/login", rest.AppLoginView)
	app.Get("/dashboard", rest.DashboardView)
	
	// API endpoints
	app.Get("/app/login", rest.Login)
	app.Get("/app/login-with-code", rest.LoginWithCode)
	app.Get("/app/logout", rest.Logout)
	app.Get("/app/reconnect", rest.Reconnect)
	app.Get("/app/devices", rest.Devices)

	return App{Service: service}
}

func (handler *App) Login(c *fiber.Ctx) error {
	response, err := handler.Service.Login(c.UserContext())
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Login success",
		Results: map[string]any{
			"qr_link":     fmt.Sprintf("%s://%s/%s", c.Protocol(), c.Hostname(), response.ImagePath),
			"qr_duration": response.Duration,
		},
	})
}

func (handler *App) LoginWithCode(c *fiber.Ctx) error {
	pairCode, err := handler.Service.LoginWithCode(c.UserContext(), c.Query("phone"))
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Login with code success",
		Results: map[string]any{
			"pair_code": pairCode,
		},
	})
}

func (handler *App) Logout(c *fiber.Ctx) error {
	err := handler.Service.Logout(c.UserContext())
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success logout",
		Results: nil,
	})
}

func (handler *App) Reconnect(c *fiber.Ctx) error {
	err := handler.Service.Reconnect(c.UserContext())
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Reconnect success",
		Results: nil,
	})
}

func (handler *App) Devices(c *fiber.Ctx) error {
	devices, err := handler.Service.FetchDevices(c.UserContext())
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Fetch device success",
		Results: devices,
	})
}
// AppLoginView serves the login page
func (handler *App) AppLoginView(c *fiber.Ctx) error {
	// Check if already authenticated
	device, err := handler.Service.GetDevice(c.UserContext())
	if err == nil && device != nil {
		// If already logged in, redirect to dashboard
		return c.Redirect("/dashboard")
	}

	// Serve login page
	return c.SendFile("./views/login.html")
}

// AppDevicesView serves the devices page (deprecated - redirect to dashboard)
func (handler *App) AppDevicesView(c *fiber.Ctx) error {
	return c.Redirect("/dashboard")
}

// DashboardView serves the main dashboard
func (handler *App) DashboardView(c *fiber.Ctx) error {
	return c.SendFile("./views/dashboard.html")
}