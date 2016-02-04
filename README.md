# govm
Prototype REST API

1. приєднатись до vmware vsphere
2. отримати список віртуалок
3. викачати vmdk/vmx для віртуалки

use vmware vsphere api

##API
GET localhost:10100/vms (use result of govc ls command)
GET localhost:10100/vms/:alias - for downloading vmdk/vmx file

##dependency:
go get github.com/vmware/govmomi/govc
go get github.com/labstack/echo
go get github.com/labstack/echo/middleware
go get github.com/spf13/viper
