package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type HTTPRouterHandler interface {
	Handle(http.ResponseWriter, *http.Request, httprouter.Params)
}
