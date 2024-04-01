require (
	github.com/aws/aws-lambda-go v1.36.1
	github.com/aws/aws-sdk-go-v2 v1.26.1
	github.com/aws/aws-sdk-go-v2/config v1.27.9
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.13.12
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.31.0
	github.com/aws/aws-sdk-go-v2/service/sqs v1.31.4
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8

module sam-app

go 1.16
