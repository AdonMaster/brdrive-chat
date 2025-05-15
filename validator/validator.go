package validator

import (
	"chat/helpers"
	"chat/responses"
	"github.com/thedevsaddam/govalidator"
	"net/http"
	"reflect"
	"strings"
)

func extractRulesFromStruct[T any](data T) govalidator.MapData {
	fields := reflect.VisibleFields(reflect.TypeOf(data))
	r := make(govalidator.MapData, len(fields))
	for _, f := range fields {
		vTag, vOk := f.Tag.Lookup("v")
		jTag, jOk := f.Tag.Lookup("json")
		if vOk && jOk {
			r[jTag] = helpers.ArrMap(strings.Split(vTag, "|"), strings.TrimSpace)
		}
	}
	return r
}

// Validate ValidateWithTags ;
// Valida e monta resposta...
// Pelo amor do guarda, use a tag 'json' em conjunto com a tag 'v'
// Ex.:
//
//	type Body struct {
//				Email string `json:"email" v:"required|email"`
//				Password string `json:"password" v:"required|between:4,10"`
//			}
//
// Para documentação da string de validação, use: https://github.com/thedevsaddam/govalidator/
func Validate[T any](w http.ResponseWriter, r *http.Request, model *T) bool {
	rules := extractRulesFromStruct(*model)
	v := govalidator.New(govalidator.Options{
		Data:    model,
		Request: r,
		Rules:   rules,
	})

	errorList := v.ValidateJSON()

	if len(errorList) > 0 {
		response := responses.Payload{
			Status:  http.StatusUnprocessableEntity,
			Message: "Erro de validação",
			Payload: errorList,
		}
		response.Write(w)
		return false
	}

	return true
}
