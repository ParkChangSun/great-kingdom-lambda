require (
	github.com/aws/aws-lambda-go v1.36.1
	github.com/aws/aws-sdk-go-v2 v1.26.1
	github.com/aws/aws-sdk-go-v2/config v1.27.9
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.13.13
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.7.13
	github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi v1.19.4
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.31.1
	github.com/aws/aws-sdk-go-v2/service/sqs v1.31.4
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8

module sam-app

go 1.16
