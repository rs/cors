module github.com/rs/cors/examples

require (
	github.com/codegangsta/inject v0.0.0-20150114235600-33e0aa1cb7c0 // indirect
	github.com/codegangsta/negroni v1.0.0
	github.com/gin-contrib/sse v0.0.0-20170109093832-22d885f9ecc7 // indirect
	github.com/gin-gonic/gin v1.3.0
	github.com/go-martini/martini v0.0.0-20170121215854-22fa46961aab
	github.com/gobuffalo/buffalo v0.12.7
	github.com/gorilla/mux v1.6.2
	github.com/json-iterator/go v1.1.5 // indirect
	github.com/julienschmidt/httprouter v0.0.0-20180715161854-348b672cd90d
	github.com/justinas/alice v0.0.0-20171023064455-03f45bd4b7da
	github.com/martini-contrib/render v0.0.0-20150707142108-ec18f8345a11
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/oxtoacart/bpool v0.0.0-20150712133111-4e1c5567d7c2 // indirect
	github.com/pressly/chi v3.3.3+incompatible
	github.com/rs/cors v1.5.0
	github.com/ugorji/go/codec v0.0.0-20180927125128-99ea80c8b19a // indirect
	github.com/zenazn/goji v0.9.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v8 v8.18.2 // indirect
)

replace github.com/rs/cors/wrapper/gin => ../wrapper

replace github.com/rs/cors => ../
