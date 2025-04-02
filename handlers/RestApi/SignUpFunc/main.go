package main

import (
	"context"
	"encoding/json"
	"errors"
	"great-kingdom-lambda/lib/auth"
	"great-kingdom-lambda/lib/ddb"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/crypto/bcrypt"
)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	body := auth.Authenticate{}
	json.Unmarshal([]byte(req.Body), &body)

	_, err := ddb.GetUser(ctx, body.Id)
	if !errors.Is(err, ddb.ErrItemNotFound) {
		b, _ := json.Marshal(auth.ErrorResponseBody{Message: "아이디 중복"})
		return auth.RESTResponse(400, auth.CORSHeaders, string(b)), nil
	}

	idlen := regexp.MustCompile(`^[0-9a-zA-Z]{4,15}$`)
	if !idlen.Match([]byte(body.Id)) {
		b, _ := json.Marshal(auth.ErrorResponseBody{Message: "아이디 : 특수기호 없는 영문/숫자 4-15 글자"})
		return auth.RESTResponse(400, auth.CORSHeaders, string(b)), nil
	}

	num := regexp.MustCompile(`[0-9]`)
	eng := regexp.MustCompile(`[a-zA-Z]`)
	bytelen := regexp.MustCompile(`^[a-zA-Z0-9@#$%^&*]{6,30}$`)
	if !num.Match([]byte(body.Password)) || !eng.Match([]byte(body.Password)) || !bytelen.Match([]byte(body.Password)) {
		b, _ := json.Marshal(auth.ErrorResponseBody{Message: "비밀번호 : 영문+숫자 6-30 글자 (포함가능 : @#$%^&*)"})
		return auth.RESTResponse(400, auth.CORSHeaders, string(b)), nil
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)

	err = ddb.PutUser(ctx, body.Id, string(hash))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return auth.RESTResponse(201, auth.CORSHeaders, ""), nil
}

func main() {
	lambda.Start(handler)
}
