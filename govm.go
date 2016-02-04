package main

import (
	"flag"
	"log"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/vmware/govmomi/govc/cli"
	_ "github.com/vmware/govmomi/govc/ls"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/spf13/viper"
)

var (
	flAddr   = flag.String("l", ":10100", "http api listen addr:port")
	cnfFName = flag.String("cnf", "config.json", "application use this file name for reading settings for govc")
)

type App struct {
	router *echo.Echo
	vmPath string
}

func NewApp(cnfFileName string) *App {
	log.Printf("httpapi init, read config from: %s\n", cnfFileName)

	app := &App{}
	app.router = echo.New()
	app.router.Use(middleware.Logger(), middleware.Recover())
	app.router.Use(echoJsonCheckErrorMW())
	app.router.Get("/vms", app.handleListVm)
	app.router.Post("/vms/:alias", app.handleDownloadVm)

	viper.SetConfigFile(cnfFileName)
	err := viper.ReadInConfig()
	//if config file exists and configs have been read successfully
	if err == nil {
		for _, cnfAlias := range []string{
			"GOVC_URL",
			"GOVC_USERNAME",
			"GOVC_PASSWORD",
			"GOVC_CERTIFICATE",
			"GOVC_PRIVATE_KEY",
			"GOVC_INSECURE",
			"GOVC_PERSIST_SESSION",
			"GOVC_MIN_API_VERSION",
		} {
			//rewrite only if not set in env scope
			if viper.IsSet(cnfAlias) && os.Getenv(cnfAlias) == "" {
				//log.Printf("write to env: %s=%s\n", cnfAlias, viper.GetString(cnfAlias))
				os.Setenv(cnfAlias, viper.GetString(cnfAlias))
			}
		}
		if viper.IsSet("vm-path") {
			app.vmPath = viper.GetString("vm-path")
			//log.Printf("vm path for host: %s\n", app.vmPath)
		}
	} else {
		log.Fatalf("%s\nThis application need config.json with vm-path key and path to VM on host machine\n", err)
	}

	return app
}

func (app *App) Run(addr string) error {
	log.Printf("run http api on %s", addr)
	return http.ListenAndServe(addr, app.router)
}

func (a *App) handleListVm(ctx *echo.Context) error {
	//get data from Stdout
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	res := cli.Run([]string{"ls", a.vmPath})

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	if res != 0 {
		return fmt.Errorf("%s (%d)", out, res)
	}
	log.Printf("get by api: %s\n", out)
	vms := []string{}
	for _, vmAlias := range strings.Split(string(out), "\n") {
		if len(vmAlias) > len(a.vmPath) {
			vms = append(vms, vmAlias[len(a.vmPath):])
		}
	}
	replyJson(ctx, vms)
	return nil
}

func (a App) handleDownloadVm(c *echo.Context) error {
	c.Response().Header().Set(echo.ContentType, echo.ApplicationJSON)
	c.Response().WriteHeader(http.StatusOK)
	for _, l := range []struct{ H, W int }{{2, 7}, {4, 7}, {8, 7}} {
		if err := json.NewEncoder(c.Response()).Encode(l); err != nil {
			return err
		}
		c.Response().Flush()
	}
	return nil
}

func replyJson(ctx *echo.Context, v interface{}) error {
	return ctx.JSON(200, map[string]interface{}{"response": v})
}

func replyJsonError(ctx *echo.Context, err interface{}) error {
	return ctx.JSON(400, map[string]interface{}{"error": fmt.Sprintf("%s", err)})
}

func echoJsonCheckErrorMW() echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := h(c)
			if err != nil {
				return replyJsonError(c, err)
			}
			return nil
		}
	}
}

func main() {
	var err error
	flag.Parse()
	app := NewApp(*cnfFName)

	if err = app.Run(*flAddr); err != nil {
		log.Fatalf("run error: %s", err)
	}
}
